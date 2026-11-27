# egg-regex

Regular expressions for [Egg](https://github.com/femto/egg), in pure Egg,
matched in time linear in the text, as [RE2](https://github.com/google/re2)
and Rust's [regex](https://github.com/rust-lang/regex) do. No pattern can
make a match take exponential time: `(x+x+)+y` on 100,000 `x`s takes
milliseconds. A match costs at most one step per instruction of the
compiled pattern for each character of the text, and a pattern compiles
to at most 100,000 instructions (`error.TooBig` past that), so text from
outside is safe to match; a pattern from outside is bounded by its size,
so limit how long one may be as well. The price is what needs
backtracking: back-references and lookaround are not supported.

```
clutch add regex --git https://github.com/femto/egg-regex
```

```egg
uses alloc, io

const regex = @import("regex")

fn main(io: Io, a: Allocator) !void {
    const re = try regex.compile(a, "(?P<user>\\w+)@(?P<host>[\\w.]+)")
    defer a.free(re)
    const s = "write to bob@example.com"
    var caps: Buf(?regex.Span) = .empty
    defer a.free(caps)
    if (re.captures(a, s, &caps)) {
        const host = caps[re.group_index("host") orelse 0] orelse return
        io.print(s[host.start..host.end])           // example.com
    }
}
```

It needs [Egg](https://github.com/femto/egg) 0.2.2 or later.

## Matching

`regex.compile(a, pattern)` gives a `regex.Regex`, freed with `a.free(re)`,
or an error such as `error.MissingParen`; `regex.compile_where(a, pattern,
&at)` also leaves `at` at the byte offset of the problem. A `Regex` is
only read once compiled, so threads may share one. Matching takes an
`Allocator` for the VM's lists, freed before it returns.

| `re.` | gives |
| --- | --- |
| `is_match(a, s)` | whether it matches anywhere in s |
| `find(a, s)` | the leftmost match, a `?regex.Span` (`start`, `end`, `len()`) |
| `find_at(a, s, start)` | the leftmost match starting at start or later; `^`, `\b` and the like still see the text before start |
| `captures(a, s, &caps)` | false if no match; else `caps`, a `Buf(?regex.Span)`, holds group 0 (the whole match) and each group, null for one that took no part |
| `captures_at(a, s, start, &caps)` | the same from start |
| `group_count()` | how many groups the pattern has, not counting group 0 |
| `group_index(name)` | `?i64`: the number of the group `(?P<name>...)` or `(?<name>...)` |

Positions are byte offsets, so `s[m.start..m.end]` is the match. Of the
matches at the leftmost position, the one the pattern prefers wins, as in
Perl, RE2, Go and Rust: `a|ab` finds `a` in `ab`, `a*?` the shortest. For
every match in turn, call `find_at` from the end of the last one, or one
character past it when it was empty, and stop when there is no match or
the start has gone past the end of the text (after an empty match at the
end); `find_at` and `captures_at` find nothing from a start outside
`0..@len(s)`. To find what Go's `FindAll` finds, also skip an empty match
that starts where the last match ended: `a*` on `a` is then `0..1` alone,
not `0..1` and `1..1`. test/lib/check.egg does just that.

## Syntax

RE2's, as Go's `regexp` has it:

| | |
| --- | --- |
| `x` `\.` `\x41` `\x{65e5}` `\n` `\t` `\r` `\f` `\v` `\a` | a character; a backslash before any ASCII but a letter or digit is that character |
| `.` | any character but `\n` (any with `(?s)`) |
| `[abc]` `[^a-z]` `[[:alpha:]]` `[[:^digit:]]` | classes, with ranges and POSIX classes |
| `\d` `\w` `\s` `\D` `\W` `\S` | ASCII digit, word character, space, and their negations |
| `xy` `x\|y` | concatenation, alternation |
| `x*` `x+` `x?` `x{n}` `x{n,}` `x{n,m}` | repetition, greedy (n and m at most 1000) |
| `x*?` `x+?` `x??` `x{n,m}?` | the same, lazy |
| `(x)` `(?P<name>x)` `(?<name>x)` `(?:x)` | groups, named, not capturing |
| `^` `$` | start and end of text (of a line with `(?m)`) |
| `\A` `\z` | start and end of text |
| `\b` `\B` | an ASCII word boundary, and not one |
| `(?i)` `(?m)` `(?s)` `(?U)` `(?i-m)` `(?i:x)` | flags: case-insensitive, multi-line, `.` matches `\n`, lazy by default |

Patterns and text are UTF-8: `.` and classes match a code point, and a
byte that is not UTF-8 matches as U+FFFD would. Differences from Go's
`regexp`, for now: `(?i)` folds ASCII letters only; `\p{...}` Unicode
classes, `\Q...\E` and octal escapes are not there yet; and a group name
may not be used twice or start with a digit (as in RE2 and Rust).

## How

The pattern is parsed to a tree (src/parse.egg) and compiled to a
program of instructions in Thompson's way (src/prog.egg), which a Pike
VM runs (src/regex.egg): every thread steps over a character together,
at most one per instruction, so the time is O(text × pattern) whatever
the pattern.

## Tests

```
clutch test
```

`test/match.egg` and `test/random.egg` (200 random patterns, made by
`test/gen`) print, for each case, the groups of the leftmost match and
every match; their expected output is what Go's `regexp` says of the same
cases (`go run test/go/main.go`), which CI checks again. `test/errors.egg`
has what is not a pattern, and `test/api.egg` the rest of the API.
