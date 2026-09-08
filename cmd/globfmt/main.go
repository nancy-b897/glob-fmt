// Command globfmt formats shell-style glob patterns into their
// canonical form, so a script can validate or normalize patterns
// without linking Go code against the glob package directly.
//
// Usage:
//
//	globfmt 'src/**/*.{go,txt}' '[^abc]*.log'
//	echo '{a,b,c}.go' | globfmt
//
// Patterns are read from the command line if any are given, or one
// per line from stdin otherwise. Each valid pattern is printed in its
// canonical form. An invalid pattern is reported on stderr and makes
// the exit status nonzero, but does not stop the rest from being
// processed.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	glob "github.com/nancy-b897/glob-fmt"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	patterns := args
	if len(patterns) == 0 {
		var err error
		patterns, err = readLines(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "globfmt: %v\n", err)
			return 1
		}
	}

	status := 0
	for _, p := range patterns {
		out, err := glob.Format(p)
		if err != nil {
			fmt.Fprintf(stderr, "globfmt: %v\n", err)
			status = 1
			continue
		}
		fmt.Fprintln(stdout, out)
	}
	return status
}

func readLines(r io.Reader) ([]string, error) {
	var lines []string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines, sc.Err()
}
