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

func parseSegment(seg, context string) ([]Node, error) {
	toks, err := tokenize(seg)
	if err != nil {
		return nil, fmt.Errorf("glob: invalid pattern %q: %w", context, err)
	}
	if len(toks) == 2 && !toks[0].esc && toks[0].r == '*' && !toks[1].esc && toks[1].r == '*' {
		return []Node{{Kind: KindDoubleStar}}, nil
	}
	return parseTokens(toks, context)
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

// tryParseBrace looks at toks[0] == '{' and decides whether it opens
// an alternation group. If it does not (no matching '}', or a
// matching one with no top-level comma), ok is false and the caller
// treats '{' as an ordinary literal character.
func tryParseBrace(toks []charTok, context string) (node Node, consumed int, ok bool, err error) {
	depth := 1
	hasComma := false
	hasNested := false
	closeIdx := -1

	for i := 1; i < len(toks) && closeIdx < 0; i++ {
		t := toks[i]
		if t.esc {
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
	}

	if closeIdx < 0 || !hasComma {
		return Node{}, 0, false, nil
	}
	if hasNested {
		return Node{}, 0, false, fmt.Errorf("glob: invalid pattern %q: nested braces are not supported", context)
	}

	parts := splitOnUnescapedComma(toks[1:closeIdx])
	alts := make([][]Node, 0, len(parts))
	for _, p := range parts {
		sub, err := parseTokens(p, context)
		if err != nil {
			return Node{}, 0, false, err
		}
		alts = append(alts, sub)
	}
	return Node{Kind: KindBrace, Alts: alts}, closeIdx + 1, true, nil
}

func splitOnUnescapedComma(toks []charTok) [][]charTok {
	var parts [][]charTok
	start := 0
	for i, t := range toks {
		if !t.esc && t.r == ',' {
			parts = append(parts, toks[start:i])
			start = i + 1
		}
	}
	return append(parts, toks[start:])
}
