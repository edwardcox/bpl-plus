print "Cookbook — foreach"
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
