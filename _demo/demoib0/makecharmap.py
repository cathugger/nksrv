#!/usr/bin/env python2

from __future__ import print_function

pfx = 9600
str = ''
count = 256
step = 16

str = "<table border=\"1\">\n"

str += "<tr><th>dec</th><th>hex</th>"
for i in range(0, step):
	str += "<th>%x</th>" % i
str += "<tr>\n"

x = 0
while x < count:
	num = pfx + x
	str += "<tr><th>%d</th><th>%x</th>" % (num, num)
	for i in range(0, step):
		str += "<td>" + unichr(pfx + x) + "</td>"
		x = x + 1
	str += "</tr>\n"

str += "</table>\n"

s = """<!DOCTYPE html>
<html>

<head>
<title>unicode runes test</title>
<link rel="stylesheet" type="text/css" href="style.css" />
</head>

<body>
"""
print(s, end='')

print(str, end='')

s = """</body>

</html>
"""
print(s, end='')
