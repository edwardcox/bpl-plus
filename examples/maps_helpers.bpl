print "Maps helpers"

m = {"name": "Edward", "lang": "BPL+", "year": 2026}

print "has name? " + str(has(m, "name"))
print "has missing? " + str(has(m, "missing"))

print "get(name, default): " + get(m, "name", "???")
print "get(missing, default): " + get(m, "missing", "default-value")

print "keys: " + str(keys(m))
print "values: " + str(values(m))

print "done"
