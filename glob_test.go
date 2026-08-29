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

func TestDoubleStarMustBeWholeSegment(t *testing.T) {
	p, err := Parse("a/**/b")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p.Segments) != 3 || len(p.Segments[1]) != 1 || p.Segments[1][0].Kind != KindDoubleStar {
		t.Fatalf("expected middle segment to be a lone double star, got %#v", p.Segments)
	}
}
