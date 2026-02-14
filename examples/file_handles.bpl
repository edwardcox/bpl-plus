print "file handles demo"

open #1, "tmp/handles_demo.txt", "w"
print #1, "Hello from BPL+"
print #1, "Line 2"
close #1

open #2, "tmp/handles_demo.txt", "a"
print #2, "Appended line"
close #2

print "wrote tmp/handles_demo.txt"
