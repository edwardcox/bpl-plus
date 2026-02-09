a = [1, 2]
push(a, 3)
push(a, "four")
push(a, true)

print a
print len(a)

i = 0
push(a, a[i] + 10)

print a
print "done"
