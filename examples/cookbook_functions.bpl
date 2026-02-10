print "Cookbook — Functions"
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
