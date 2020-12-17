#!/bin/sh
set -uex

mkdir -p bin
bin=`realpath bin`

for x in "$@"
do
	x=`realpath "$x"`
	(cd src/nksrv; go build -o "$bin" "$x")
done
