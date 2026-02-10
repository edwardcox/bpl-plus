print "BPL+ Language Tour — Control Flow"
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
