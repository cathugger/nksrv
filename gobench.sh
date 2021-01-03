#!/bin/sh
set -uex

for x in "$@"
do
	x=`realpath "$x"`
	(cd src/nksrv; go test -bench=. "$x")
done
