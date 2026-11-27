// What Go's regexp, which has RE2's semantics, says of the cases in a
// test (test/match.egg, or the file named), in the form it prints: run it
// from the package as `go run test/go/main.go > test/match.expected` or
// `go run test/go/main.go test/random.egg > test/random.expected`.
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// The two string literals of a line `t(io, a, &n, "pattern", "text")`.
func literals(line string) (string, string, bool) {
	const head = "t(io, a, &n, "
	i := strings.Index(line, head)
	if i < 0 {
		return "", "", false
	}
	rest := line[i+len(head):]
	p, err := strconv.QuotedPrefix(rest)
	if err != nil {
		panic(line)
	}
	rest = strings.TrimPrefix(rest[len(p):], ", ")
	s, err := strconv.QuotedPrefix(rest)
	if err != nil {
		panic(line)
	}
	pattern, _ := strconv.Unquote(p)
	text, _ := strconv.Unquote(s)
	return pattern, text, true
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
		pattern, text, ok := literals(sc.Text())
		if !ok {
			continue
		}
		n++
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Printf("%d: error\n", n)
			continue
		}
		m := re.FindStringSubmatchIndex(text)
		if m == nil {
			fmt.Printf("%d: no %v\n", n, re.MatchString(text))
			continue
		}
		var b strings.Builder
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
		fmt.Printf("%d: %s\n", n, b.String())
	}
}
