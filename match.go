package glob

import "strings"

// Match reports whether path satisfies the pattern. path is split on
// '/' the same way the pattern was, with no notion of a filesystem
// behind it — this is pure string matching against the syntax tree.
func (p *Pattern) Match(path string) bool {
	return matchSegments(p.Segments, strings.Split(path, "/"))
}

// matchSegments walks the pattern's path segments against the path's
// path segments in lockstep, except for a lone "**" segment, which is
// allowed to absorb zero or more path segments before the rest of the
// pattern resumes.
func matchSegments(pats [][]Node, segs []string) bool {
	if len(pats) == 0 {
		return len(segs) == 0
	}
	if isDoubleStar(pats[0]) {
		for i := 0; i <= len(segs); i++ {
			if matchSegments(pats[1:], segs[i:]) {
				return true
			}
		}
		return false
	}
	if len(segs) == 0 {
		return false
	}
	return matchNodes(pats[0], []rune(segs[0])) && matchSegments(pats[1:], segs[1:])
}

func isDoubleStar(seg []Node) bool {
	return len(seg) == 1 && seg[0].Kind == KindDoubleStar
}

// matchNodes reports whether the node sequence matches all of s. Star
// and brace nodes require backtracking, since how much of s they
// consume depends on what comes after them.
func matchNodes(nodes []Node, s []rune) bool {
	if len(nodes) == 0 {
		return len(s) == 0
	}

	n := nodes[0]
	rest := nodes[1:]
	switch n.Kind {
	case KindLiteral:
		lit := []rune(n.Literal)
		if len(s) < len(lit) {
			return false
		}
		for i, r := range lit {
			if s[i] != r {
				return false
			}
		}
		return matchNodes(rest, s[len(lit):])
	case KindAny:
		if len(s) == 0 {
			return false
		}
		return matchNodes(rest, s[1:])
	case KindClass:
		if len(s) == 0 || !classMatches(n.Class, s[0]) {
			return false
		}
		return matchNodes(rest, s[1:])
	case KindStar:
		for i := 0; i <= len(s); i++ {
			if matchNodes(rest, s[i:]) {
				return true
			}
		}
		return false
	case KindBrace:
		for _, alt := range n.Alts {
			combined := make([]Node, 0, len(alt)+len(rest))
			combined = append(combined, alt...)
			combined = append(combined, rest...)
			if matchNodes(combined, s) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func classMatches(c *CharClass, r rune) bool {
	in := false
	for _, it := range c.Items {
		if r >= it.Lo && r <= it.Hi {
			in = true
			break
		}
	}
	if c.Negate {
		return !in
	}
	return in
}
