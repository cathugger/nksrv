#!/bin/sh
set -eux

root=$(realpath $(dirname $0))

: ${outdir:=_demo/demoib0/static}
#: ${sasscmd:=sassc -t compressed}
: ${sasscmd:=sassc -t expanded}

tmpdir=$(mktemp -d -t nksrv-sass-XXXXXXXXXXXXXXX)
trap 'rm -rf "$tmpdir"' EXIT TERM INT

cd "$root/sass" 2>/dev/null &&
	cp -rf -t "$tmpdir" * 2>/dev/null || :

cd "$root/usr/sass" 2>/dev/null &&
	cp -rf -t "$tmpdir" * 2>/dev/null || :

cd "$tmpdir"
find -type f -iregex '\./[^_].*\.scss' | \
	while read -r file
do
	$sasscmd "$file" "${file%.scss}.css"
done

otmp="$root"/$outdir/_tmp
mkdir -p "$otmp"
trap 'rm -rf "$tmpdir" "$otmp"' EXIT TERM INT
find -type f -iregex '\./[^_].*\.css' | \
	xargs -r mv -f -t "$otmp"
cd "$otmp"
mv -f -t "$root"/$outdir *

cd
