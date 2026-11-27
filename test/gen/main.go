// Writes test/random.egg: random small patterns over a and b, each run
// on a few random texts, for comparing with Go's regexp (test/go). Run
// from the package: `go run test/gen/main.go > test/random.egg`. The seed
// is fixed, so the file is the same each time.
package main

import (
	"fmt"
	"math/rand"
	"strings"
)

var r = rand.New(rand.NewSource(1))

func atom(depth int) string {
	switch r.Intn(12) {
	case 0, 1, 2:
		return "a"
	case 3, 4:
		return "b"
	case 5:
		return "."
	case 6:
		return []string{"[ab]", "[^a]", "[a-b ]", "\\\\w", "\\\\s", "\\\\pL", "\\\\p{Greek}", "\\\\PL", "[[:upper:]]", "σ", "k"}[r.Intn(11)]
	case 7:
		return []string{"^", "$", "\\\\b", "\\\\B"}[r.Intn(4)]
	case 8, 9:
		if depth < 3 {
			return "(" + expr(depth+1) + ")"
		}
		return "a"
	case 10:
		if depth < 3 {
			return "(?:" + expr(depth+1) + ")"
		}
		return "b"
	default:
		return ""
	}
}

func repeat(depth int) string {
	x := atom(depth)
	if x == "" || x == "^" || x == "$" {
		return x
	}
	switch r.Intn(10) {
	case 0:
		return x + "*"
	case 1:
		return x + "+"
	case 2:
		return x + "?"
	case 3:
		return x + []string{"*?", "+?", "??"}[r.Intn(3)]
	case 4:
		lo := r.Intn(3)
		switch r.Intn(3) {
		case 0:
			return fmt.Sprintf("%s{%d}", x, lo)
		case 1:
			return fmt.Sprintf("%s{%d,}", x, lo)
		default:
			return fmt.Sprintf("%s{%d,%d}", x, lo, lo+r.Intn(3))
		}
	default:
		return x
	}
}

func concat(depth int) string {
	var b strings.Builder
	n := 1 + r.Intn(4)
	for i := 0; i < n; i++ {
		b.WriteString(repeat(depth))
	}
	return b.String()
}

func expr(depth int) string {
	s := concat(depth)
	if depth == 0 && r.Intn(5) == 0 {
		s = "(?i)" + s
	}
	for r.Intn(4) == 0 {
		s += "|" + concat(depth)
	}
	return s
}

// Mostly a and b, now and then a letter that case folding or \pL
// treats apart: K, the Kelvin sign, the long s, sigmas, é.
func text() string {
	var b strings.Builder
	n := r.Intn(9)
	for i := 0; i < n; i++ {
		if r.Intn(6) == 0 {
			b.WriteString([]string{"K", "\\u{212a}", "\\u{17f}", "s", "Σ", "σ", "ς", "é"}[r.Intn(8)])
		} else {
			b.WriteByte("aab b"[r.Intn(5)])
		}
	}
	return b.String()
}

func main() {
	fmt.Println("// Random patterns, made by test/gen, each on a few random texts:")
	fmt.Println("// test/random.expected is what Go's regexp says of them (test/go).")
	fmt.Println("uses alloc, io")
	fmt.Println()
	fmt.Println("const check = @import(\"lib/check.egg\")")
	fmt.Println()
	fmt.Println("fn main(io: Io, a: Allocator) void {")
	fmt.Println("    var n = 0")
	for i := 0; i < 200; i++ {
		p := expr(0)
		for j := 0; j < 3; j++ {
			fmt.Printf("    check.t(io, a, &n, \"%s\", \"%s\")\n", p, text())
		}
	}
	fmt.Println("}")
}
