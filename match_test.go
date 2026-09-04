package glob

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.txt", false},
		{"*.go", "sub/main.go", false},
		{"file?.txt", "file1.txt", true},
		{"file?.txt", "file10.txt", false},
		{"file?.txt", "file.txt", false},
		{"[abc].go", "a.go", true},
		{"[abc].go", "d.go", false},
		{"[!abc].go", "d.go", true},
		{"[!abc].go", "a.go", false},
		{"[a-z0-9_].go", "5.go", true},
		{"[a-z0-9_].go", "_.go", true},
		{"[a-z0-9_].go", "A.go", false},
		{"{a,b,c}.go", "b.go", true},
		{"{a,b,c}.go", "d.go", false},
		{"src/{a,b}.go", "src/a.go", true},
		{"src/{a,b}.go", "src/c.go", false},
		{"a/**/b", "a/b", true},
		{"a/**/b", "a/x/b", true},
		{"a/**/b", "a/x/y/b", true},
		{"a/**/b", "a/b/c", false},
		{"**", "anything", true},
		{"**", "a/b/c", true},
		{"**/*.go", "main.go", true},
		{"**/*.go", "pkg/sub/main.go", true},
		{"**/*.go", "pkg/sub/main.txt", false},
		{`\*.txt`, "*.txt", true},
		{`\*.txt`, "x.txt", false},
		{"a/b/c", "a/b/c", true},
		{"a/b/c", "a/b", false},
		{"a/b/c", "a/b/c/d", false},
		{"[]abc].go", "].go", true},
		{"[]abc].go", "a.go", true},
	}
	for _, c := range cases {
		t.Run(c.pattern+" "+c.path, func(t *testing.T) {
			p, err := Parse(c.pattern)
			if err != nil {
				t.Fatalf("Parse(%q): %v", c.pattern, err)
			}
			if got := p.Match(c.path); got != c.want {
				t.Fatalf("Parse(%q).Match(%q) = %v, want %v", c.pattern, c.path, got, c.want)
			}
		})
	}
}
