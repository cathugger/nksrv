#!/bin/sh
set -eux

: ${outdir:=_demo/demoib0/static}
#: ${sasscmd:=sassc -t compressed}
: ${sasscmd:=sassc -t expanded}

tmpdir=$(mktemp -d -t nksrv-sass-XXXXXXXXXXXXXXX)
trap "rm -rf $tmpdir" EXIT SIGTERM SIGINT

cp -rf -t "$tmpdir" sass/* 2>/dev/null || :
cp -rf -t "$tmpdir" usr/sass/* 2>/dev/null || :

pushd "$tmpdir"

find -type f -iregex '\./[^_].*\.scss' | while read -r file
do
	$sasscmd "$file" "${file%.scss}.css"
done

popd

mkdir -p $outdir
cp -rf -t $outdir $tmpdir/*.css 2>/dev/null || :
