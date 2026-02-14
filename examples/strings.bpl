# examples/strings.bpl
print "strings demo"

s = "  Hello, World!  "
print "s=" + s

print "lower: " + lower(s)
print "upper: " + upper(s)

print "trim(space): [" + trim(s) + "]"
print "ltrim(space): [" + ltrim(s) + "]"
print "rtrim(space): [" + rtrim(s) + "]"

print "trim(cutset ' H! '): [" + trim(s, " H! ") + "]"
print "ltrim(cutset ' '): [" + ltrim(s, " ") + "]"
print "rtrim(cutset ' '): [" + rtrim(s, " ") + "]"

print "contains 'World': " + contains(s, "World")
print "contains 'banana': " + contains(s, "banana")

print "startswith '  He': " + startswith(s, "  He")
print "endswith '  ': " + endswith(s, "  ")

print "replace World->BPL+: " + replace(s, "World", "BPL+")
print "replace l->X (n=2): " + replace(s, "l", "X", 2)

csv = "a,b,c"
parts = split(csv, ",")
print "split a,b,c => " + parts

print "join(parts,'|') => " + join(parts, "|")

print "indexof 'World' => " + indexof(s, "World")
print "indexof 'zzz' => " + indexof(s, "zzz")
print "lastindexof 'l' => " + lastindexof(s, "l")

print "repeat('ha', 5) => " + repeat("ha", 5)

# substr(s, start[, count]) is rune-safe
u = "🙂🙃😉"
print "u=" + u
print "len(u) => " + len(u)
print "substr(u,0,1) => " + substr(u, 0, 1)
print "substr(u,1,2) => " + substr(u, 1, 2)
print "substr(u,2) => " + substr(u, 2)

# substr on trimmed string
t = trim(s)
print "t=" + t
print "substr(t,0,5) => " + substr(t, 0, 5)
print "substr(t,7) => " + substr(t, 7)

print "done"
