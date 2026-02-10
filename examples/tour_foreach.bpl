print "BPL+ Language Tour — foreach"
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
