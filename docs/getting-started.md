# Getting Started with BPL+

This guide will get you **up and running with BPL+ in minutes**.

BPL+ is a modern, BASIC-inspired programming language designed for clarity, fast feedback, and approachability — without feeling old or limited.

---

## What you’ll need

BPL+ is written in Go, so you only need:

- **Go** (version 1.21 or newer recommended)
- macOS, Linux, or Windows

That’s it. No build tools, no dependencies, no package managers.

---

## 1. Install Go

### Check if Go is already installed
Open a terminal and run:

```bash
go version
If you see a version number, you’re good to go.

Install Go (if needed)
Ubuntu / Debian
sudo apt update
sudo apt install golang-go
macOS (Homebrew)
brew install go
Windows
Download and install from:
https://go.dev/dl/

After installation, open a new terminal and confirm:

go version
2. Get the BPL+ source code
Clone the repository:

git clone https://github.com/edwardcox/bpl-plus.git
cd bpl-plus
3. Run your first BPL+ program
BPL+ includes example programs in the examples/ folder.

Run the classic “hello” example:

go run ./cmd/bpl run examples/hello.bpl
You should see output similar to:

Hello Edward
...
done
🎉 That means BPL+ is working.

4. Write your own program
Create a new file:

nano examples/my_first.bpl
Paste this:

print "Hello from BPL+"

a = [1, 2, 3]
push(a, 99)

print a
print "done"

Save and run it:

go run ./cmd/bpl run examples/my_first.bpl
Expected output:

Hello from BPL+
[1, 2, 3, 99]
done

5. Basic language overview
Variables
x = 10
name = "Edward"
ok = true
Printing
print x
print "Hello"
If / else
if x > 5
  print "big"
else
  print "small"
end
Loops
for i = 1 to 5
  print i
end
while x < 10
  x = x + 1
end
Arrays
a = [10, 20, 30]
print a[1]

push(a, 99)
print a
foreach
for each value in a
  print value
end
for each value, index in a
  print index
  print value
end
Functions (explicit return)
function add(a, b)
  return a + b
end

print add(2, 3)
6. Runtime errors (helpful by design)
BPL+ runtime errors show:

filename

line number

source line

caret pointing at the problem

Example:

print x
Output:

Runtime error at examples/my_first.bpl:1:7
  Undefined variable "x"
  1 | print x
            ^
This is intentional — errors are meant to teach, not frustrate.

7. Formatting & development (contributors)
If you’re modifying the interpreter itself:

go fmt ./...
What’s next?
📘 Language Tour: docs/language-tour.md

🍳 Cookbook (recipes): docs/cookbook.md

📂 Examples: examples/

💡 Feature ideas: GitHub Issues → Feature Wishlist / Ideas

Final note
If you’ve ever thought:
“I love the idea of BASIC, but I want something modern.”
You’re exactly who BPL+ is for.

Welcome aboard.