package glob

import "fmt"

type charTok struct {
	r   rune
	esc bool
}

// tokenize resolves backslash escapes across the whole segment up
// front, so the structural parsing below never has to think about
// backslashes itself — it just checks the esc flag on each token.
func tokenize(seg string) ([]charTok, error) {
	rs := []rune(seg)
	toks := make([]charTok, 0, len(rs))
	for i := 0; i < len(rs); i++ {
		if rs[i] == '\\' {
			if i+1 >= len(rs) {
				return nil, fmt.Errorf("trailing backslash")
			}
			toks = append(toks, charTok{r: rs[i+1], esc: true})
			i++
			continue
		}
		toks = append(toks, charTok{r: rs[i]})
	}
	return toks, nil
}

// parseSegments splits toks into path segments on unescaped '/' — but
// a '/' inside a balanced "{...}" is not a split point here, since it
// might belong to a brace alternative that spans a path separator.
// That '/' gets its chance to split things when the alternative's own
// tokens are parsed, recursively, by tryParseBrace.
func parseSegments(toks []charTok, context string) ([][]Node, error) {
	rawSegs := splitTopLevel(toks, '/')
	segs := make([][]Node, len(rawSegs))
	for i, raw := range rawSegs {
		nodes, err := parseSegmentTokens(raw, context)
		if err != nil {
			return nil, err
		}
		if err := checkSpanningBracePlacement(nodes, context); err != nil {
			return nil, err
		}
		segs[i] = nodes
	}
	return segs, nil
}

// checkSpanningBracePlacement rejects a brace node with an
// alternative spanning more than one path segment unless that brace
// is the entire segment — see the package doc comment for why mixing
// it with a literal prefix or suffix is disallowed.
func checkSpanningBracePlacement(nodes []Node, context string) error {
	for _, n := range nodes {
		if n.Kind != KindBrace {
			continue
		}
		spans := false
		for _, alt := range n.Alts {
			if len(alt) != 1 {
				spans = true
				break
			}
		}
		if spans && len(nodes) != 1 {
			return fmt.Errorf("glob: invalid pattern %q: a brace alternative spanning '/' must be the entire path segment", context)
		}
	}
	return nil
}

func parseSegmentTokens(toks []charTok, context string) ([]Node, error) {
	if len(toks) == 2 && !toks[0].esc && toks[0].r == '*' && !toks[1].esc && toks[1].r == '*' {
		return []Node{{Kind: KindDoubleStar}}, nil
	}
	return parseTokens(toks, context)
}

// splitTopLevel splits toks on unescaped occurrences of sep, skipping
// over bracket classes and balanced brace groups so that a separator
// inside either of those is never mistaken for a split point.
func splitTopLevel(toks []charTok, sep rune) [][]charTok {
	var parts [][]charTok
	start := 0
	i := 0
	for i < len(toks) {
		t := toks[i]
		if !t.esc && t.r == '[' {
			if n, ok := classTokenLen(toks[i:]); ok {
				i += n
				continue
			}
		}
		if !t.esc && t.r == '{' {
			if n, ok := braceTokenLen(toks[i:]); ok {
				i += n
				continue
			}
		}
		if !t.esc && t.r == sep {
			parts = append(parts, toks[start:i])
			start = i + 1
		}
		i++
	}
	return append(parts, toks[start:])
}

func parseTokens(toks []charTok, context string) ([]Node, error) {
	var nodes []Node
	var lit []rune
	flush := func() {
		if len(lit) > 0 {
			nodes = append(nodes, Node{Kind: KindLiteral, Literal: string(lit)})
			lit = nil
		}
	}

	i := 0
	for i < len(toks) {
		t := toks[i]
		if t.esc {
			lit = append(lit, t.r)
			i++
			continue
		}
		switch t.r {
		case '*':
			if i+1 < len(toks) && !toks[i+1].esc && toks[i+1].r == '*' {
				return nil, fmt.Errorf("glob: invalid pattern %q: '**' is only valid as an entire path segment", context)
			}
			flush()
			nodes = append(nodes, Node{Kind: KindStar})
			i++
		case '?':
			flush()
			nodes = append(nodes, Node{Kind: KindAny})
			i++
		case '[':
			flush()
			class, consumed, err := parseClass(toks[i:], context)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, Node{Kind: KindClass, Class: class})
			i += consumed
		case '{':
			flush()
			brace, consumed, ok, err := tryParseBrace(toks[i:], context)
			if err != nil {
				return nil, err
			}
			if ok {
				nodes = append(nodes, brace)
				i += consumed
				continue
			}
			// no matching brace, or one with no top-level comma: '{'
			// is just a literal character, one at a time.
			lit = append(lit, '{')
			i++
		default:
			lit = append(lit, t.r)
			i++
		}
	}
	flush()
	return nodes, nil
}

// parseClass parses a "[...]" class starting at toks[0] == '['. It
// returns the class and the number of tokens consumed, including both
// brackets.
func parseClass(toks []charTok, context string) (*CharClass, int, error) {
	idx := 1
	cls := &CharClass{}

	if idx < len(toks) && !toks[idx].esc && (toks[idx].r == '!' || toks[idx].r == '^') {
		cls.Negate = true
		idx++
	}

	// A ']' in the first content position is a literal member, not
	// the closing bracket — the same convention POSIX shells use.
	first := true
	closed := false
	for idx < len(toks) {
		t := toks[idx]
		if !t.esc && t.r == ']' && !first {
			closed = true
			idx++
			break
		}
		first = false

		lo := t.r
		if idx+2 < len(toks) && !toks[idx+1].esc && toks[idx+1].r == '-' &&
			!(!toks[idx+2].esc && toks[idx+2].r == ']') {
			hi := toks[idx+2].r
			if hi < lo {
				return nil, 0, fmt.Errorf("glob: invalid pattern %q: character range %c-%c is backwards", context, lo, hi)
			}
			cls.Items = append(cls.Items, ClassItem{Lo: lo, Hi: hi})
			idx += 3
		} else {
			cls.Items = append(cls.Items, ClassItem{Lo: lo, Hi: lo})
			idx++
		}
	}
	if !closed {
		return nil, 0, fmt.Errorf("glob: invalid pattern %q: unterminated character class", context)
	}
	return cls, idx, nil
}

// classTokenLen reports the length, in tokens, of a bracket class
// starting at toks[0] == '[', following the same "first position" rule
// parseClass uses for a literal ']'. It returns ok == false if there is
// no matching ']', in which case the caller should treat '[' as an
// ordinary character rather than skip over its contents — callers that
// need brace/comma structure must not mistake a ',' or '}' inside an
// unrelated, unterminated class for one at the top level.
func classTokenLen(toks []charTok) (n int, ok bool) {
	idx := 1
	if idx < len(toks) && !toks[idx].esc && (toks[idx].r == '!' || toks[idx].r == '^') {
		idx++
	}
	first := true
	for idx < len(toks) {
		if !toks[idx].esc && toks[idx].r == ']' && !first {
			return idx + 1, true
		}
		first = false
		idx++
	}
	return 0, false
}

// braceExtent scans toks, which must start with an unescaped '{', for
// its matching unescaped '}'. It also reports whether a top-level
// comma or a nested '{' was seen along the way. Bracket classes are
// skipped as a unit, so a comma or brace inside "[...]" — as in
// "{[a,b],c}" — never confuses the search for the alternation's
// structure.
func braceExtent(toks []charTok) (closeIdx int, hasComma, hasNested, ok bool) {
	depth := 1
	closeIdx = -1

	for i := 1; i < len(toks) && closeIdx < 0; {
		t := toks[i]
		if !t.esc && t.r == '[' {
			if n, ok := classTokenLen(toks[i:]); ok {
				i += n
				continue
			}
		}
		if t.esc {
			i++
			continue
		}
		switch t.r {
		case '{':
			depth++
			if depth == 2 {
				hasNested = true
			}
		case '}':
			depth--
			if depth == 0 {
				closeIdx = i
			}
		case ',':
			if depth == 1 {
				hasComma = true
			}
		}
		i++
	}

	if closeIdx < 0 {
		return 0, false, false, false
	}
	return closeIdx, hasComma, hasNested, true
}

// braceTokenLen reports the length, in tokens, of a balanced "{...}"
// group starting at toks[0] == '{', for splitTopLevel to skip over as
// a unit. It returns ok == false if there is no matching '}'.
func braceTokenLen(toks []charTok) (n int, ok bool) {
	closeIdx, _, _, found := braceExtent(toks)
	if !found {
		return 0, false
	}
	return closeIdx + 1, true
}

// tryParseBrace looks at toks[0] == '{' and decides whether it opens
// an alternation group. If it does not (no matching '}', or a
// matching one with no top-level comma), ok is false and the caller
// treats '{' as an ordinary literal character.
func tryParseBrace(toks []charTok, context string) (node Node, consumed int, ok bool, err error) {
	closeIdx, hasComma, hasNested, found := braceExtent(toks)
	if !found || !hasComma {
		return Node{}, 0, false, nil
	}
	if hasNested {
		return Node{}, 0, false, fmt.Errorf("glob: invalid pattern %q: nested braces are not supported", context)
	}

	parts := splitOnUnescapedComma(toks[1:closeIdx])
	alts := make([]Alt, 0, len(parts))
	for _, p := range parts {
		segs, err := parseSegments(p, context)
		if err != nil {
			return Node{}, 0, false, err
		}
		if hasDoubleStar(segs) {
			return Node{}, 0, false, fmt.Errorf("glob: invalid pattern %q: '**' is not allowed inside a brace group", context)
		}
		alts = append(alts, Alt(segs))
	}
	return Node{Kind: KindBrace, Alts: alts}, closeIdx + 1, true, nil
}

// hasDoubleStar reports whether any segment of an alternative is a
// lone "**". That form only has meaning as an entire path segment of
// the pattern itself, not as one option among brace alternatives, so
// it is rejected here rather than silently never matching.
func hasDoubleStar(segs [][]Node) bool {
	for _, seg := range segs {
		if len(seg) == 1 && seg[0].Kind == KindDoubleStar {
			return true
		}
	}
	return false
}

func splitOnUnescapedComma(toks []charTok) [][]charTok {
	var parts [][]charTok
	start := 0
	i := 0
	for i < len(toks) {
		t := toks[i]
		if !t.esc && t.r == '[' {
			if n, ok := classTokenLen(toks[i:]); ok {
				i += n
				continue
			}
		}
		if !t.esc && t.r == ',' {
			parts = append(parts, toks[start:i])
			start = i + 1
		}
		i++
	}
	return append(parts, toks[start:])
}
