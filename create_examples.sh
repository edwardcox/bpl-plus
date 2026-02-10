#!/usr/bin/env bash
set -euo pipefail

# create_examples.sh
# Creates BPL+ Tour + Cookbook example programs under ./examples
# Run from repo root:  bash scripts/create_examples.sh
# Or:                bash create_examples.sh

ROOT_DIR="$(pwd)"
EXAMPLES_DIR="$ROOT_DIR/examples"

mkdir -p "$EXAMPLES_DIR"

write_file() {
  local path="$1"
  local content="$2"
  mkdir -p "$(dirname "$path")"
  printf "%s" "$content" > "$path"
  echo "✅ wrote $(realpath --relative-to="$ROOT_DIR" "$path" 2>/dev/null || echo "$path")"
}

write_file "$EXAMPLES_DIR/tour_basics.bpl" \
'print "BPL+ Language Tour — Basics"
print "------------------------"

x = 10
name = "Edward"
ok = true

print x
print name
print ok

print "Math:"
print 10 + 5
print 10 * 3

print "Comparisons:"
print 10 > 3
print "a" < "b"

print "done"
'

write_file "$EXAMPLES_DIR/tour_control_flow.bpl" \
'print "BPL+ Language Tour — Control Flow"
print "--------------------------------"

x = 5

if x >= 3
  print "x is at least 3"
else
  print "x is small"
end

print "While loop:"
i = 1
while i <= 5
  print i
  i = i + 1
end

print "For loop (count up):"
for j = 1 to 5
  print j
end

print "For loop (count down):"
for k = 5 to 1
  print k
end

print "For loop (step 2):"
for m = 1 to 9 step 2
  print m
end

print "done"
'

write_file "$EXAMPLES_DIR/tour_functions.bpl" \
'print "BPL+ Language Tour — Functions"
print "------------------------------"

function add(a, b)
  return a + b
end

function greet(name)
  return "Hello " + name
end

print add(2, 3)
print greet("Edward")

print "done"
'

write_file "$EXAMPLES_DIR/tour_arrays.bpl" \
'print "BPL+ Language Tour — Arrays"
print "---------------------------"

a = [10, 20, 30]
print a
print a[2]

a[1] = 99
print a

push(a, 123)
print a
print len(a)

print "Array concat:"
b = [1, 2]
c = [3, 4]
d = b + c
print d

print "done"
'

write_file "$EXAMPLES_DIR/tour_foreach.bpl" \
'print "BPL+ Language Tour — foreach"
print "----------------------------"

a = [10, 20, 30]

print "Value only:"
for each x in a
  print x
end

print "Value + index:"
for each x, i in a
  print i
  print x
end

print "done"
'

write_file "$EXAMPLES_DIR/tour_errors.bpl" \
'print "BPL+ Language Tour — Errors"
print "--------------------------"

# This example intentionally triggers a runtime error.
# You should see filename, line number, source line, and caret.

print x
'

write_file "$EXAMPLES_DIR/cookbook_arrays.bpl" \
'print "Cookbook — Arrays"
print "-----------------"

print "Create + index + assign"
a = [10, 20, 30]
print a[0]
a[1] = 99
print a

print "push / pop"
b = [1, 2]
push(b, 3)
print b
x = pop(b)
print x
print b

print "insert / remove"
c = [10, 20]
insert(c, 1, 99)
print c
y = remove(c, 0)
print y
print c

print "done"
'

write_file "$EXAMPLES_DIR/cookbook_foreach.bpl" \
'print "Cookbook — foreach"
print "------------------"

a = [10, 20, 30]

print "Value only:"
for each v in a
  print v
end

print "Value + index:"
b = ["a", "b", "c"]
for each v, i in b
  print i
  print v
end

print "done"
'

write_file "$EXAMPLES_DIR/cookbook_functions.bpl" \
'print "Cookbook — Functions"
print "-------------------"

function add(a, b)
  return a + b
end

function safe_first(a)
  if len(a) == 0
    return "empty"
  end
  return a[0]
end

print add(2, 3)

x = [10, 20]
print safe_first(x)

y = []
print safe_first(y)

print "done"
'

write_file "$EXAMPLES_DIR/cookbook_errors.bpl" \
'print "Cookbook — Errors (intentional)"
print "-------------------------------"

print "1) Undefined variable (will error)"
print x
'

echo ""
echo "✅ Tour/Cookbook examples created in: $EXAMPLES_DIR"
echo ""
echo "Try one:"
echo "  go run ./cmd/bpl run examples/tour_basics.bpl"
echo "  go run ./cmd/bpl run examples/tour_arrays.bpl"
echo "  go run ./cmd/bpl run examples/tour_foreach.bpl"
echo "  go run ./cmd/bpl run examples/tour_errors.bpl   # intentional runtime error"
