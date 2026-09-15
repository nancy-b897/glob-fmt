package glob

import "strings"

// Print renders p back into a valid glob pattern. The result is not
// guaranteed to be byte-identical to whatever text was originally
// parsed — escaping and the negation marker are normalized — but
// parsing it again produces an equivalent syntax tree.
func Print(p *Pattern) string {
	segs := make([]string, len(p.Segments))
	for i, seg := range p.Segments {
		segs[i] = printNodes(seg)
	}
	return strings.Join(segs, "/")
}

// Format parses pattern and, if valid, returns its canonical form.
func Format(pattern string) (string, error) {
	p, err := Parse(pattern)
	if err != nil {
		return "", err
	}
	return Print(p), nil
}

func (p *Pattern) String() string { return Print(p) }

func printNodes(nodes []Node) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Kind {
		case KindLiteral:
			b.WriteString(escapeLiteral(n.Literal))
		case KindStar:
			b.WriteByte('*')
		case KindDoubleStar:
			b.WriteString("**")
		case KindAny:
			b.WriteByte('?')
		case KindClass:
			b.WriteString(printClass(n.Class))
		case KindBrace:
			b.WriteString(printBrace(n.Alts))
		}
	}
	return b.String()
}

func escapeLiteral(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '*', '?', '[', ']', '{', '}', ',', '/':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func printClass(c *CharClass) string {
	var b strings.Builder
	b.WriteByte('[')
	if c.Negate {
		b.WriteByte('!')
	}
	for i, it := range c.Items {
		if it.Lo == it.Hi {
			r := it.Lo
			needsEscape := r == '\\' || r == '-' ||
				(r == ']' && i != 0) ||
				((r == '!' || r == '^') && i == 0 && !c.Negate)
			if needsEscape {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
			continue
		}
		if it.Lo == '\\' || it.Lo == ']' {
			b.WriteByte('\\')
		}
		b.WriteRune(it.Lo)
		b.WriteByte('-')
		if it.Hi == '\\' || it.Hi == ']' {
			b.WriteByte('\\')
		}
		b.WriteRune(it.Hi)
	}
	b.WriteByte(']')
	return b.String()
}

func printBrace(alts []Alt) string {
	parts := make([]string, len(alts))
	for i, a := range alts {
		segs := make([]string, len(a))
		for j, seg := range a {
			segs[j] = printNodes(seg)
		}
		parts[i] = strings.Join(segs, "/")
	}
	return "{" + strings.Join(parts, ",") + "}"
}
