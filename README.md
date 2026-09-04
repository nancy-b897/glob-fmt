# glob-fmt

A small Go library for working with shell-style glob patterns: `*.go`,
`**/*.go`, `src/{a,b}/*.txt`, `[0-9]*.log`, and so on.

## Why

`filepath.Match` and most glob implementations either accept a pattern
or fail with an unhelpful error the moment you try to use it. If a
config file lets users write their own glob patterns — an ignore
file, a build tool, a file watcher config — you usually want to
validate those patterns up front and say exactly what's wrong, rather
than surface a cryptic error the first time some file happens to
trigger the broken part of the pattern.

This package parses a glob pattern into an explicit syntax tree,
validating it as it goes, and can pretty-print that tree back into a
canonical string.

## Usage

```go
p, err := glob.Parse("src/**/*.{go,txt}")
if err != nil {
    log.Fatal(err)
}

fmt.Println(glob.Print(p)) // src/**/*.{go,txt}
```

Bad patterns are rejected with a specific reason instead of silently
matching nothing or panicking the first time they're used:

```go
_, err := glob.Parse("a**b")
// glob: invalid pattern "a**b": '**' is only valid as an entire path segment

_, err = glob.Parse("[z-a]")
// glob: invalid pattern "[z-a]": character range z-a is backwards
```

`glob.Format` is a shortcut for parse-then-print, useful as a
`gofmt`-style normalizer:

```go
out, err := glob.Format("[^abc]*.log")
// out == "[!abc]*.log" -- negation is normalized to '!'
```

Once parsed, a pattern can match paths directly:

```go
p, _ := glob.Parse("src/**/*.go")
p.Match("src/glob/parser.go") // true
p.Match("src/glob.go")        // true
p.Match("bin/glob")           // false
```

## Supported syntax

Per `/`-separated path segment:

| Pattern   | Meaning                                                     |
|-----------|--------------------------------------------------------------|
| `*`       | any run of characters, not including `/`                     |
| `**`      | any run of characters, including `/` — must be its own path segment (`a/**/b`, not `a**b`) |
| `?`       | exactly one character, not including `/`                     |
| `[abc]`   | one character from the set                                   |
| `[a-z]`   | one character from the range                                 |
| `[!abc]`  | one character not in the set (`^` is also accepted on input) |
| `{a,b,c}` | one of several sub-patterns                                  |
| `\x`      | the literal character `x`                                    |

A brace group with no top-level comma, like `{abc}`, is not treated as
an alternation — it's left as literal text, matching how most shells
behave.

## Known limitations

- A brace alternative can't contain a character class with a literal
  comma in it (`{[a,b],c}`) — the comma gets misread as the
  alternative separator.
- `**` and `{...}` can't span a `/`, since the pattern is split into
  path segments before either is parsed.

## License

MIT, see LICENSE.
