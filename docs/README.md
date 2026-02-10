# BPL+ Documentation

Welcome to **BPL+** — a modern, BASIC-inspired language focused on clarity, fast iteration, and “developer joy”.

## Quick links

- **Getting Started:** `docs/getting-started.md`
- **Language Tour:** `docs/language-tour.md`
- **Cookbook (copy/paste recipes):** `docs/cookbook.md`
- **Examples folder:** `examples/`

## What’s in BPL+ right now?

Implemented (current MVP core):
- Variables, numbers, strings, booleans
- `print`
- Arithmetic + comparisons
- `if / else / end`
- `while <cond> ... end`
- `for i = start to end [step expr] ... end`
- `for each x in array ... end`
- `for each x, i in array ... end` (index variable)
- Functions: `function name(args...) ... return expr ... end`
- Built-ins: `str()`, `num()`, `len()`, `input()`
- Arrays: `[]` literals, indexing `a[i]`, assignment `a[i] = x`
- Array built-ins: `push(a, x)`, `pop(a)`, `insert(a, i, x)`, `remove(a, i)`
- Runtime errors include filename + line + caret + source line

## Philosophy (short version)

BPL+ is **BASIC-inspired**, not BASIC-compatible.
We borrow the best parts (readability, approachability, fast feedback) and modernize everything else.

## Contributing

- Issues: bugs, feature requests, design discussions
- PRs: welcome (small + focused is best)
- Please include a tiny example program for any language change.

## Roadmap (near-term)

- Arrays polish (more helpers, maybe slicing later)
- File I/O
- Modules/imports
- Better numeric formatting options
- More standard library functions
