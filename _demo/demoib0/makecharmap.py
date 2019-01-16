#!/usr/bin/env python2

from __future__ import print_function

str = ''
step = 0x20

str = "<table border=\"1\">\n"

str += "<tr><th>dec</th><th>hex</th>"
for i in range(0, step):
	str += "<th>%02X</th>" % i
str += "<tr>\n"

def appendrange(start, length):
	global str
	x = 0
	while x < length:
		num = start + x
		str += "<tr><th>%04d</th><th>%04X</th>" % (num, num)
		for i in range(0, step):
			str += "<td>" + unichr(start + x) + "</td>"
			x = x + 1
		str += "</tr>\n"

def appendempty():
	global str
	str += "<tr><th></th><th></th>"
	for i in range(0, step):
		str += "<td></td>"
	str += "</tr>\n"


appendrange(0x2100, 0x100)
appendrange(0x2200, 0x100)
appendrange(0x2300, 0x100)
appendrange(0x2400, 0x100)
appendrange(0x2500, 0x100)
appendrange(0x2600, 0x100)
appendrange(0x2700, 0x100)
appendempty()
appendrange(0x2A00, 0x100)
appendrange(0x2B00, 0x100)
appendempty()
appendrange(0x1F700, 0x100)


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
