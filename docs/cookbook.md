# Cookbook

These are practical “copy/paste and modify” snippets.

---

## Arrays

### Create + index + assign

```bpl
a = [10, 20, 30]
print a[0]
a[1] = 99
print a
push / pop
a = [1, 2]
push(a, 3)
print a

x = pop(a)
print x
print a
insert / remove
a = [10, 20]
insert(a, 1, 99)
print a     # [10, 99, 20]

x = remove(a, 0)
print x     # 10
print a     # [99, 20]
foreach loops
Value only
a = [10, 20, 30]
for each x in a
  print x
end
Value + index (0-based)
a = ["a", "b", "c"]
for each x, i in a
  print i
  print x
end
Functions
Simple function
function greet(name)
  return "Hello " + name
end

print greet("Edward")
Guard patterns (manual checks)
function safe_first(a)
  if len(a) == 0
    return "empty"
  end
  return a[0]
end
Errors
Undefined variable
print x
Array index out of bounds
a = [1, 2]
print a[99]
Wrong type (example)
a = "not an array"
print a[0]
Output will show file + line + caret so you can fix quickly.


---

## 6) Examples folder: suggested files + contents

If you want, you can create these as-is.

### `examples/tour_arrays.bpl`
```bpl
a = [10, 20, 30]
print a
print a[2]
a[1] = 99
print a
push(a, 123)
print a
print "done"
examples/tour_foreach.bpl
a = [10, 20, 30]

for each x in a
  print x
end

for each x, i in a
  print i
  print x
end

print "done"
examples/tour_functions.bpl
function add(a, b)
  return a + b
end

print add(2, 3)
print "done"
examples/tour_errors.bpl
print x

