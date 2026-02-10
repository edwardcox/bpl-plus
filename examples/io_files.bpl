print "File I/O demo"

path = "tmp/demo.txt"

writefile(path, "Hello from BPL+\n")
appendfile(path, "Line 2\n")

print "exists? " + str(exists(path))
print "contents:"
print readfile(path)

print "done"
