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
| `next(a, s, &it)` | each match in turn, from `var it = regex.iter()` (freed with `a.free(it)`), then null |
| `next_captures(a, s, &it, &caps)` | the same with its groups in caps, then false |
| `find_all(a, s, &out)` | every match, pushed onto `out: Buf(regex.Span)` |
| `replace_all(a, &out, s, template)` | s at the end of `out: Buf(u8)` with every match replaced by template, its `$` expanded |
| `replace(a, &out, s, template)` | the same for the first match only |
| `replace_all_literal(a, &out, s, with)` | every match replaced by with as it is |
| `expand(a, &out, s, template, caps)` | template with its `$` expanded from the groups in caps |
| `split(a, s, &out)` | the pieces of s between the matches, as `Span`s pushed onto out |
| `group_count()` | how many groups the pattern has, not counting group 0 |
| `group_index(name)` | `?i64`: the number of the group `(?P<name>...)` or `(?<name>...)` |

Positions are byte offsets, so `s[m.start..m.end]` is the match. Of the
matches at the leftmost position, the one the pattern prefers wins, as in
Perl, RE2, Go and Rust: `a|ab` finds `a` in `ab`, `a*?` the shortest.
`find_at` and `captures_at` find nothing from a start outside
`0..@len(s)`.

```egg
var it = regex.iter()
defer a.free(it)
while (re.next(a, s, &it)) |m| {
    io.print(s[m.start..m.end])
}
```

An `Iter` keeps the DFA its searches build (below) from one match to the
next, so it is freed when done. `next` goes through the matches as Go's
`FindAll` does: each search
starts at the end of the last match, or a character past it when that was
empty, and an empty match right after the last match is skipped, so `a*`
on `a` is `0..1` alone, not `0..1` and `1..1`. `find_all`, `replace_all`,
`replace_all_literal` and `split` go the same way, and give what Go's
`FindAllString`, `ReplaceAllString`, `ReplaceAllLiteralString` and
`Split(s, -1)` give.

A template is Go's: `$1` or `${1}` is group 1, `$name` or `${name}` the
named group, `$$` a `$`. A name is the longest run of letters, digits and
`_`, so `$1x` is the group called `1x`; write `${1}x`. A group that took no
part, or that the pattern does not have, is nothing, and a `$` that starts
none of these is itself.

```egg
const date = try regex.compile(a, "(\\d{4})-(\\d{2})-(\\d{2})")
defer a.free(date)
var out: Buf(u8) = .empty
defer a.free(out)
date.replace_all(a, &out, "on 2026-11-27", "$3/$2/$1")   // on 27/11/2026
```

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
program of instructions in Thompson's way (src/prog.egg), which one of
three engines runs, each O(text × pattern) whatever the pattern, as in
RE2 and Go:

- a backtracker (src/regex.egg), when the program's instructions times
  the text's length is at most 262,144: depth first, with a bit for each
  (instruction, position) so that none is tried twice. It is the fastest
  on short texts.
- a lazy DFA (src/dfa.egg) on longer ones: a state is the set of
  instructions the NFA's threads are at, made the first time a search
  reaches it, its transitions filled in as they are taken. It steps on
  classes of code points, so its semantics are the NFA's. One DFA finds
  where the leftmost-first match ends, another, over the program
  backwards, where it starts; groups, when wanted, are then found by the
  NFA on that span alone. `^`, `$` and `\b` wait in a state for the
  character that settles them, as in RE2.
- a Pike VM (src/regex.egg), when a DFA would take more than 4 MB of
  states (a pattern like `(a|b)*a(a|b){16}`): every thread steps over a
  character together, in memory that does not grow with the text.

The backtracker and the DFA skip, while no match is under way, to the
next byte a match can start with, worked out when compiling (`S` for
`Sherlock`, a digit for `\d+`). A call allocates once or nearly for the
NFA; the DFA's states live as long as the `Iter`, or the `find_all`,
`replace_all` or `split` call, that runs the searches.

`bench/bench.egg` times some cases (`eggc build bench/bench.egg --release
--pkg regex=src`), and `bench/go/main.go` the same with Go's `regexp`. On
a 2019 Intel Mac, ms:

| | egg-regex | Go |
| --- | --- | --- |
| every `Sherlock` in 1 MB | 3.7 | 5.9 |
| every `Sherlock\|Watson\|Holmes` | 9.3 | 58.1 |
| every `\w+@\w+\.com` | 10.2 | 35.5 |
| every `\d{4}-\d{2}-\d{2}` | 1.0 | 25.0 |
| every `(?i)sherlock` | 5.2 | 36.7 |
| `zzz+`, found nowhere | 0.7 | 0.3 |
| 100,000 `is_match` on short texts | 38.8 | 15.4 |
| 100,000 `captures` on short texts | 26.5 | 29.2 |

## Tests

```
clutch test
```

`test/match.egg` and `test/random.egg` (200 random patterns, made by
`test/gen`) print, for each case, the groups of the leftmost match, every
match, the text with the matches replaced, and its pieces split;
`test/replace.egg` has templates. Their expected output is what Go's
`regexp` says of the same cases (`go run test/go/main.go`), which CI
checks again. `test/errors.egg`
has what is not a pattern, and `test/api.egg` the rest of the API. Every
case is run by every engine as well (the DFA with room and with too little,
so that it gives up), which must agree; `test/long.egg` checks the API on
a 60 KB text, where it runs the DFA, against the Pike VM.
