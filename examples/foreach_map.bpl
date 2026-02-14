print "foreach map demo"

m = {"b": 2, "a": 1, "c": 99}

print "keys"
foreach k in m
  print k
end

print "pairs"
foreach k, v in m
  print k + "=" + v
end
