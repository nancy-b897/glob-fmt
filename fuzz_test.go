package glob

import (
	"path/filepath"
	"testing"
)

// FuzzMatchAgainstFilepathMatch checks Pattern.Match against
// path/filepath.Match wherever the two are supposed to agree: patterns
// built only from literals, '*', '?', and non-negated character
// classes with no '**' or brace groups, none of which filepath.Match
// understands. Negated classes are excluded too, since this package
// accepts '!' as the primary negation spelling while filepath.Match
// only recognizes '^' — a bare '!' means something different to each
// of them, rather than one simply erroring out.
func FuzzMatchAgainstFilepathMatch(f *testing.F) {
	seeds := []struct{ pattern, name string }{
		{"*.go", "main.go"},
		{"*.go", "main.txt"},
		{"file?.txt", "file1.txt"},
		{"[abc].go", "b.go"},
		{"[abc].go", "d.go"},
		{"[a-z0-9_].go", "5.go"},
		{"a/b/c", "a/b/c"},
		{"a/*/c", "a/b/c"},
		{"a/*/c", "a/b/x/c"},
		{"src/*.go", "src/main.go"},
		{`\*.txt`, "*.txt"},
		{`a\/b`, "a/b"},
	}
	for _, s := range seeds {
		f.Add(s.pattern, s.name)
	}

	f.Fuzz(func(t *testing.T, pattern, name string) {
		p, err := Parse(pattern)
		if err != nil {
			return
		}
		if !filepathCompatible(p) {
			return
		}
		want, err := filepath.Match(pattern, name)
		if err != nil {
			return
		}
		if got := p.Match(name); got != want {
			t.Fatalf("Parse(%q).Match(%q) = %v, filepath.Match(%q, %q) = %v", pattern, name, got, pattern, name, want)
		}
	})
}

func filepathCompatible(p *Pattern) bool {
	for _, seg := range p.Segments {
		for _, n := range seg {
			switch n.Kind {
			case KindDoubleStar, KindBrace:
				return false
			case KindClass:
				if n.Class.Negate {
					return false
				}
			}
		}
	}
	return true
}
