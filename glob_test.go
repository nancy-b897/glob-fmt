package glob

import (
	"reflect"
	"testing"
)

func TestParseValid(t *testing.T) {
	cases := []string{
		"*.go",
		"**/*.go",
		"a/**/b",
		"**",
		"file?.txt",
		"[abc].go",
		"[!abc].go",
		"[^abc].go",
		"[a-z0-9_].go",
		"{a,b,c}.go",
		"src/{a,b}.go",
		"a/b/c",
		"a//b",
		`\*.txt`,
		`\{literal\}.txt`,
		"[]abc].go",
		"{abc}.go",
		"{abc",
		`a\,b`,
		"{[a,b],c}.go",
		"{[!,]x,y}.go",
		"a/{b/c,d}/e",
		"{a/b,c}",
		"src/{x/y/z,w}",
		`a\/b`,
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, err := Parse(in); err != nil {
				t.Fatalf("Parse(%q) returned unexpected error: %v", in, err)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	cases := []string{
		"",
		"a**",
		"**b",
		"a**b",
		"***",
		`a\`,
		"[abc",
		"[z-a]",
		"{a,{b,c}}",
		"[]",
		"[!]",
		"pre{a/b,c}",
		"{a/b,c}post",
		"{**,b}",
		"{a/**,b}",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, err := Parse(in); err == nil {
				t.Fatalf("Parse(%q) succeeded, want error", in)
			}
		})
	}
}

// TestRoundTrip checks that Print produces text that reparses into an
// equivalent tree, even when the printed text differs from the
// original (e.g. "[^a]" prints as "[!a]").
func TestRoundTrip(t *testing.T) {
	cases := []string{
		"*.go",
		"**/*.go",
		"a/**/b",
		"file?.txt",
		"[abc].go",
		"[!abc].go",
		"[^abc].go",
		"[a-z0-9_].go",
		"{a,b,c}.go",
		"src/{a,b}.go",
		`\*.txt`,
		`\{literal\}.txt`,
		"[]abc].go",
		`a\,b`,
		"{[a,b],c}.go",
		"a/{b/c,d}/e",
		"{a/b,c}",
		`a\/b`,
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			p1, err := Parse(in)
			if err != nil {
				t.Fatalf("Parse(%q): %v", in, err)
			}
			out := Print(p1)
			p2, err := Parse(out)
			if err != nil {
				t.Fatalf("Parse(Print(%q)) = Parse(%q): %v", in, out, err)
			}
			if !reflect.DeepEqual(p1, p2) {
				t.Fatalf("round trip mismatch for %q (printed %q)\nfirst:  %#v\nsecond: %#v", in, out, p1, p2)
			}
		})
	}
}

func TestNegationCanonicalized(t *testing.T) {
	out, err := Format("[^abc]")
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if out != "[!abc]" {
		t.Fatalf("Format(%q) = %q, want %q", "[^abc]", out, "[!abc]")
	}
}

func TestLeadingBracketIsLiteral(t *testing.T) {
	p, err := Parse("[]a]")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	seg := p.Segments[0]
	if len(seg) != 1 || seg[0].Kind != KindClass {
		t.Fatalf("expected a single class node, got %#v", seg)
	}
	want := []ClassItem{{Lo: ']', Hi: ']'}, {Lo: 'a', Hi: 'a'}}
	if !reflect.DeepEqual(seg[0].Class.Items, want) {
		t.Fatalf("Items = %#v, want %#v", seg[0].Class.Items, want)
	}
}

func TestBraceWithoutCommaIsLiteral(t *testing.T) {
	p, err := Parse("{abc}")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	seg := p.Segments[0]
	if len(seg) != 1 || seg[0].Kind != KindLiteral || seg[0].Literal != "{abc}" {
		t.Fatalf("expected a single literal node %q, got %#v", "{abc}", seg)
	}
}

// TestBraceSplitIsBracketAware checks that a comma inside a "[...]"
// class does not get mistaken for the comma that separates brace
// alternatives.
func TestBraceSplitIsBracketAware(t *testing.T) {
	p, err := Parse("{[a,b],c}")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	seg := p.Segments[0]
	if len(seg) != 1 || seg[0].Kind != KindBrace {
		t.Fatalf("expected a single brace node, got %#v", seg)
	}
	alts := seg[0].Alts
	if len(alts) != 2 {
		t.Fatalf("expected 2 alternatives, got %d: %#v", len(alts), alts)
	}
	// Each alternative here is a single path segment, so it holds
	// exactly one []Node entry.
	if len(alts[0]) != 1 || len(alts[0][0]) != 1 || alts[0][0][0].Kind != KindClass {
		t.Fatalf("expected first alternative to be a class, got %#v", alts[0])
	}
	want := []ClassItem{{Lo: 'a', Hi: 'a'}, {Lo: ',', Hi: ','}, {Lo: 'b', Hi: 'b'}}
	if !reflect.DeepEqual(alts[0][0][0].Class.Items, want) {
		t.Fatalf("Items = %#v, want %#v", alts[0][0][0].Class.Items, want)
	}
	if len(alts[1]) != 1 || len(alts[1][0]) != 1 || alts[1][0][0].Kind != KindLiteral || alts[1][0][0].Literal != "c" {
		t.Fatalf("expected second alternative to be literal %q, got %#v", "c", alts[1])
	}
}

// TestBraceCanSpanSeparator checks that an alternative containing '/'
// is parsed as multiple path segments within its own Alt, and that
// the brace must be the whole path segment to do so.
func TestBraceCanSpanSeparator(t *testing.T) {
	p, err := Parse("a/{b/c,d}/e")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p.Segments) != 3 {
		t.Fatalf("expected 3 top-level segments, got %d: %#v", len(p.Segments), p.Segments)
	}
	mid := p.Segments[1]
	if len(mid) != 1 || mid[0].Kind != KindBrace {
		t.Fatalf("expected the middle segment to be a lone brace, got %#v", mid)
	}
	alts := mid[0].Alts
	if len(alts) != 2 || len(alts[0]) != 2 || len(alts[1]) != 1 {
		t.Fatalf("expected alternatives of 2 and 1 path segments, got %#v", alts)
	}

	for _, in := range []string{"pre{a/b,c}", "{a/b,c}post"} {
		if _, err := Parse(in); err == nil {
			t.Fatalf("Parse(%q) succeeded, want error", in)
		}
	}
}

func TestDoubleStarMustBeWholeSegment(t *testing.T) {
	p, err := Parse("a/**/b")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p.Segments) != 3 || len(p.Segments[1]) != 1 || p.Segments[1][0].Kind != KindDoubleStar {
		t.Fatalf("expected middle segment to be a lone double star, got %#v", p.Segments)
	}
}
