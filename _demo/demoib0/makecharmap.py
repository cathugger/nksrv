#!/usr/bin/env python2

from __future__ import print_function

pfx = 9600
str = ''
count = 0x100
step = 0x20

str = "<table border=\"1\">\n"

str += "<tr><th>dec</th><th>hex</th>"
for i in range(0, step):
	str += "<th>%02x</th>" % i
str += "<tr>\n"

def appendrange(start, length):
	global str
	x = 0
	while x < length:
		num = start + x
		str += "<tr><th>%04d</th><th>%04x</th>" % (num, num)
		for i in range(0, step):
			str += "<td>" + unichr(start + x) + "</td>"
			x = x + 1
		str += "</tr>\n"

appendrange(pfx, count)

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
