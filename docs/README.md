# BPL+  
```txt
██████╗ ██████╗ ██╗         ██████╗ ██╗     ██╗   ██╗ ██████╗
██╔══██╗██╔══██╗██║         ██╔══██╗██║     ██║   ██║██╔════╝
██████╔╝██████╔╝██║         ██████╔╝██║     ██║   ██║╚█████╗ 
██╔══██╗██╔═══╝ ██║         ██╔═══╝ ██║     ██║   ██║ ╚═══██╗
██████╔╝██║     ███████╗    ██║     ███████╗╚██████╔╝██████╔╝
╚═════╝ ╚═╝     ╚══════╝    ╚═╝     ╚══════╝ ╚═════╝ ╚═════╝
```
**A modern, BASIC-inspired programming language for real programs.**

BPL+ is an open-source, non-commercial programming language inspired by the clarity and approachability of classic BASIC — redesigned for modern development.

It is not a retro interpreter.  
It is a clean, structured, extensible language built from scratch.

---

## Current Status

BPL+ is in active development.

### Implemented Features

- Variables
- Numbers, strings, booleans
- Arrays (reference semantics)
- Maps / dictionaries (string keys)
- Arithmetic + comparison operators
- Boolean logic (`and`, `or`, `not`)
- `if / else / end`
- `while`
- `for ... to ... [step]`
- Functions (explicit `return`, no implicit return)
- File I/O
- Module system (`import`)
- Built-in functions:
  - `print`
  - `str`
  - `num`
  - `len`
  - `input`
  - `push`, `pop`, `insert`, `remove`
  - `has`, `get`, `keys`, `values`
  - `readfile`, `writefile`, `exists`

---

## Installation

You need Go 1.21+ installed.

Clone the repository:

```bash
git clone https://github.com/edwardcox/bpl-plus.git
cd bpl-plus
Build the CLI:

go build -o bplplus ./cmd/bpl
Run a program:

./bplplus examples/hello.bpl
You should see output immediately.

Language Overview
Variables
x = 10
name = "Edward"
flag = true
Control Flow
if x > 5
    print "big"
else
    print "small"
end
while x < 10
    x = x + 1
end
for i = 1 to 5
    print i
end
Functions
function add(a, b)
    return a + b
end

print add(2, 3)
Explicit return required

No implicit return

Arrays
a = [1, 2, 3]
push(a, 4)
print a
print a[0]
Arrays are reference types.

Maps (Dictionaries)
m = {"a": 1, "b": 2}

print m["a"]
m["c"] = 99
print keys(m)
Map helpers:

has(m, "a")
get(m, "x", 0)
keys(m)
values(m)
File I/O
writefile("test.txt", "Hello\n")
print readfile("test.txt")
print exists("test.txt")
Modules
Import another file:

import "lib/math.bpl"
Rules:

Relative to the importing file

.bpl extension auto-added if omitted

Fallback to lib/ directory

Circular imports are detected and blocked

Duplicate imports are ignored (cached)

Example:

import "examples/lib/math.bpl"

print add(2, 3)
Project Structure
cmd/bpl/          CLI entry point
lexer/            Tokenizer
parser/           AST builder
ast/              Node definitions
interpreter/      Runtime engine
examples/         Example programs
docs/             Documentation
Philosophy
BPL+ is designed to be:

Clear

Predictable

Structured

Extensible

Suitable for real programs

The goal is not to copy old BASIC.
The goal is to build what BASIC might look like if designed today.

Roadmap (v0.3.x)
Maps Step C (iteration helpers)

REPL

Standard library modules

Import refinements

Bytecode / VM

Optimization passes

Contributing
Ideas welcome.

Open an Issue:

Feature requests

Language design suggestions

Documentation improvements

This project is early — thoughtful feedback matters.

License
Open Source. Non-commercial intent.