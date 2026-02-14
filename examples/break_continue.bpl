print "break/continue smoke"

a = [10, 20, 30, 40]

foreach x, i in a
  if x == 20
    continue
  end
  print i + ": " + x
  if x == 30
    break
  end
end