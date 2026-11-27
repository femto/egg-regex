// The cases of bench/bench.egg with Go's regexp, for comparison: run
// from the package as `go run bench/go/main.go`.
package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The same text as bench/bench.egg makes.
func text() string {
	var b strings.Builder
	words := []string{"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog", "Sherlock", "Holmes", "and", "Watson"}
	var seed uint64 = 1
	for i := 0; b.Len() < 1000000; i++ {
		seed = seed*6364136223846793005 + 1442695040888963407
		b.WriteString(words[(seed>>33)%12])
		if i%97 == 0 {
			b.WriteString(" bob" + strconv.Itoa(i) + "@example.com")
		}
		if i%131 == 0 {
			b.WriteString(" 2026-11-" + strconv.Itoa(10+i%18))
		}
		if i%13 == 12 {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func count(name, pattern, s string) {
	re := regexp.MustCompile(pattern)
	t0 := time.Now()
	n := len(re.FindAllStringIndex(s, -1))
	fmt.Printf("%s: %d matches in %.1f ms\n", name, n, float64(time.Since(t0).Microseconds())/1000)
}

func main() {
	s := text()
	count("literal Sherlock", "Sherlock", s)
	count("alternation", "Sherlock|Watson|Holmes", s)
	count("email", `\w+@\w+\.com`, s)
	count("date", `\d{4}-\d{2}-\d{2}`, s)
	count("case-insensitive", "(?i)sherlock", s)
	count("rare", "zzz+", s)
	email := regexp.MustCompile(`^[\w.+-]+@[\w-]+(\.[\w-]+)+$`)
	tries := []string{"bob@example.com", "not an email", "a.b+c@d-e.org", "x@y"}
	t0 := time.Now()
	ok := 0
	for k := 0; k < 100000; k++ {
		if email.MatchString(tries[k%4]) {
			ok++
		}
	}
	fmt.Printf("100000 short is_match: %d true in %.1f ms\n", ok, float64(time.Since(t0).Microseconds())/1000)
	date := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	t1 := time.Now()
	found := 0
	for j := 0; j < 100000; j++ {
		if date.FindStringSubmatchIndex("due 2026-11-27 or so") != nil {
			found++
		}
	}
	fmt.Printf("100000 short captures: %d in %.1f ms\n", found, float64(time.Since(t1).Microseconds())/1000)
}
