print "foreach demo with index/value"

a = ["x", "y", "z"]
foreach x, i in a
  print str(i) + ": " + x
end

m = {"b": 2, "a": 1, "c": 99}
foreach k, v in m
  print k + "=" + str(v)
end