// What Go's regexp, which has RE2's semantics, says of the cases in a
// test (test/match.egg, or the file named), in the form it prints: run it
// from the package as `go run test/go/main.go > test/match.expected` or
// `go run test/go/main.go test/random.egg > test/random.expected`
// (and test/replace.egg likewise).
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Egg's \u{N} as Go's \UNNNNNNNN, so that strconv can read the literals.
var eggUnicode = regexp.MustCompile(`\\u\{([0-9a-fA-F]+)\}`)

func goEscapes(line string) string {
	return eggUnicode.ReplaceAllStringFunc(line, func(m string) string {
		n, err := strconv.ParseUint(m[3:len(m)-1], 16, 32)
		if err != nil {
			panic(m)
		}
		return fmt.Sprintf("\\U%08x", n)
	})
}

// The string literals of a line `t(io, a, &n, "pattern", "text")` or
// `r(io, a, &n, "pattern", "text", "template")`, and which it is.
func literals(line string) (string, []string) {
	for _, kind := range []string{"t", "r"} {
		head := kind + "(io, a, &n, "
		i := strings.Index(line, head)
		if i < 0 || (i > 0 && line[i-1] != '.' && line[i-1] != ' ') {
			continue
		}
		rest := line[i+len(head):]
		var out []string
		for {
			q, err := strconv.QuotedPrefix(rest)
			if err != nil {
				panic(line)
			}
			v, _ := strconv.Unquote(q)
			out = append(out, v)
			rest = rest[len(q):]
			if !strings.HasPrefix(rest, ", ") {
				break
			}
			rest = rest[2:]
		}
		return kind, out
	}
	return "", nil
}

// s in quotes, bytes outside printable ASCII, " and \ as \xNN, as
// test/lib/check.egg writes it.
func quoted(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 32 && c < 127 && c != '"' && c != '\\' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "\\x%02x", c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func main() {
	path := "test/match.egg"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		kind, args := literals(goEscapes(sc.Text()))
		if kind == "" {
			continue
		}
		n++
		re, err := regexp.Compile(args[0])
		if err != nil {
			fmt.Printf("%d: error\n", n)
			continue
		}
		text := args[1]
		if kind == "r" {
			first := text
			if m := re.FindStringSubmatchIndex(text); m != nil {
				first = text[:m[0]] + string(re.ExpandString(nil, args[2], text, m)) + text[m[1]:]
			}
			fmt.Printf("%d: %s first %s\n", n, quoted(re.ReplaceAllString(text, args[2])), quoted(first))
			continue
		}
		var b strings.Builder
		m := re.FindStringSubmatchIndex(text)
		if m == nil {
			fmt.Fprintf(&b, "no %v", re.MatchString(text))
		} else {
			for i := 0; i < len(m); i += 2 {
				if m[i] < 0 {
					b.WriteString("(-)")
				} else {
					fmt.Fprintf(&b, "(%d,%d)", m[i], m[i+1])
				}
			}
			fmt.Fprintf(&b, " %v all", re.MatchString(text))
			for _, a := range re.FindAllStringIndex(text, -1) {
				fmt.Fprintf(&b, " %d-%d", a[0], a[1])
			}
		}
		fmt.Fprintf(&b, " repl %s", quoted(re.ReplaceAllString(text, "<$0|${1}>")))
		fmt.Fprintf(&b, " lit %s", quoted(re.ReplaceAllLiteralString(text, "#")))
		b.WriteString(" split")
		for _, p := range re.Split(text, -1) {
			b.WriteString(" " + quoted(p))
		}
		fmt.Printf("%d: %s\n", n, b.String())
	}
}
