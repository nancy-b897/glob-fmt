// Package glob parses shell-style file glob patterns into an explicit
// syntax tree, validates them, can render that tree back into a
// canonical string form, and can match the pattern against a path.
//
// Supported syntax, per '/'-separated path segment:
//
//	*        any run of characters, not including '/'
//	**       any run of characters, including '/' — only valid as a
//	         whole path segment on its own (e.g. "a/**/b", not "a**b")
//	?        exactly one character, not including '/'
//	[abc]    one character from the set
//	[a-z]    one character from the range
//	[!abc]   one character NOT in the set ('^' is also accepted on input)
//	{a,b,c}  one of several literal or glob sub-patterns
//	\x       the literal character x, with no special meaning
//
// A brace group with no top-level comma (e.g. "{abc}") is not an
// alternation; it is left as literal text, which is how most shells
// treat it too.
package glob

import (
	"fmt"
	"strings"
)

// NodeKind identifies the kind of a parsed pattern node.
type NodeKind int

const (
	KindLiteral NodeKind = iota
	KindStar
	KindDoubleStar
	KindAny
	KindClass
	KindBrace
)

// Node is one element of a parsed path segment.
type Node struct {
	Kind    NodeKind
	Literal string     // set when Kind == KindLiteral
	Class   *CharClass // set when Kind == KindClass
	Alts    [][]Node   // set when Kind == KindBrace; one slice per alternative
}

// CharClass is a parsed "[...]" character class.
type CharClass struct {
	Negate bool
	Items  []ClassItem
}

// ClassItem is a single character (Lo == Hi) or an inclusive range.
type ClassItem struct {
	Lo, Hi rune
}

// Pattern is a fully parsed glob pattern, split into path segments.
type Pattern struct {
	Segments [][]Node
}

// Parse validates pattern and builds its syntax tree. It returns an
// error describing the first problem found, rather than accepting the
// pattern and failing later inside a match call.
func Parse(pattern string) (*Pattern, error) {
	if pattern == "" {
		return nil, fmt.Errorf("glob: empty pattern")
	}
	rawSegs := strings.Split(pattern, "/")
	segs := make([][]Node, len(rawSegs))
	for i, raw := range rawSegs {
		nodes, err := parseSegment(raw, pattern)
		if err != nil {
			return nil, err
		}
		segs[i] = nodes
	}
	return &Pattern{Segments: segs}, nil
}
