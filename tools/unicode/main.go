// Writes src/unicode.egg, the Unicode tables of egg-regex, from Go's
// unicode package, so that \p{...} and (?i) agree with Go's regexp,
// which the tests compare against. Run from the package:
//
//	go run tools/unicode/main.go > src/unicode.egg
//
// A table is a str of numbers, each in base 32, most significant digit
// first: a digit that ends a number is one of the first 32 characters of
// DIGITS, any other digit one of the last 32, so that the text needs no
// escapes. Ranges are two numbers each: the gap from the end of the one
// before (plus one), then the length (minus one). A folding orbit, a
// set of code points that fold to one another, is its size, its smallest
// member as the gap from the smallest of the orbit before, then each
// other member as the gap from its smallest.
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
)

const digits = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func number(b *strings.Builder, n int) {
	if n < 0 {
		panic(n)
	}
	var ds []int
	for {
		ds = append([]int{n % 32}, ds...)
		n /= 32
		if n == 0 {
			break
		}
	}
	for i, d := range ds {
		if i < len(ds)-1 {
			b.WriteByte(digits[32+d])
		} else {
			b.WriteByte(digits[d])
		}
	}
}

// The ranges of t, strides spread out and neighbours joined.
func ranges(t *unicode.RangeTable) [][2]int {
	var cps []int
	for _, r := range t.R16 {
		for c := int(r.Lo); c <= int(r.Hi); c += int(r.Stride) {
			cps = append(cps, c)
		}
	}
	for _, r := range t.R32 {
		for c := int(r.Lo); c <= int(r.Hi); c += int(r.Stride) {
			cps = append(cps, c)
		}
	}
	sort.Ints(cps)
	var out [][2]int
	for _, c := range cps {
		if len(out) > 0 && out[len(out)-1][1]+1 == c {
			out[len(out)-1][1] = c
		} else {
			out = append(out, [2]int{c, c})
		}
	}
	return out
}

func table(t *unicode.RangeTable) string {
	var b strings.Builder
	next := 0
	for _, r := range ranges(t) {
		number(&b, r[0]-next)
		number(&b, r[1]-r[0])
		next = r[1] + 1
	}
	return b.String()
}

func orbits() string {
	var b strings.Builder
	prev := 0
	for c := rune(0); c <= unicode.MaxRune; c++ {
		if unicode.SimpleFold(c) == c {
			continue
		}
		// Once per orbit: from its smallest member.
		members := []int{int(c)}
		smallest := true
		for f := unicode.SimpleFold(c); f != c; f = unicode.SimpleFold(f) {
			if f < c {
				smallest = false
				break
			}
			members = append(members, int(f))
		}
		if !smallest {
			continue
		}
		sort.Ints(members)
		number(&b, len(members))
		number(&b, members[0]-prev)
		for _, m := range members[1:] {
			number(&b, m-members[0])
		}
		prev = members[0]
	}
	return b.String()
}

func main() {
	var names []string
	all := map[string]*unicode.RangeTable{}
	for n, t := range unicode.Categories {
		names = append(names, n)
		all[n] = t
	}
	for n, t := range unicode.Scripts {
		if _, ok := all[n]; ok {
			panic(n)
		}
		names = append(names, n)
		all[n] = t
	}
	sort.Strings(names)
	w := os.Stdout
	fmt.Fprintf(w, "// The Unicode %s tables of egg-regex, from Go's unicode package by\n", unicode.Version)
	fmt.Fprintln(w, "// tools/unicode/main.go, which says how they are written: do not edit.")
	fmt.Fprintln(w, "// table(name) is a general category (\"L\", \"Lu\") or a script (\"Greek\")")
	fmt.Fprintln(w, "// as Go's unicode.Categories and unicode.Scripts have them; ORBITS the")
	fmt.Fprintln(w, "// sets of code points simple case folding makes one.")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "const VERSION = %q\n\n", unicode.Version)
	fmt.Fprintf(w, "const DIGITS = %q\n\n", digits)
	fmt.Fprintf(w, "const ORBITS = %q\n\n", orbits())
	fmt.Fprintln(w, "// The ranges of the category or script called name, or null.")
	fmt.Fprintln(w, "fn table(name: []u8) ?str {")
	for _, n := range names {
		fmt.Fprintf(w, "    if (name == %q) {\n        return %q\n    }\n", n, table(all[n]))
	}
	fmt.Fprintln(w, "    return null")
	fmt.Fprintln(w, "}")
}
