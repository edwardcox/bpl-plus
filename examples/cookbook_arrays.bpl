print "Cookbook — Arrays"
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
