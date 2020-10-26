#!/bin/sh

#exec find ./ -iname '*.go' -exec gofmt -s -w '{}' ';'

if [ x"$1" = x'-u' ]
then
	cd # otherwise it'll add dep to current go.mod
	go get -u golang.org/x/tools/cmd/goimports
	echo "Updated." >&2
	exit
fi

export GOPATH=`go env GOPATH`:`pwd`

if [ x"$1" = x"-all" ]
then
	find src/nksrv -type f -name '*.go' -not -regex ".*_nofmt[._].*" -exec goimports -local 'nksrv/' -w '{}' ';'
	exit
fi

if [ "$#" -lt 1 ]
then
	printf "Usage:\n\t%s a.go ...\n\t%s -all\n\t%s -u\n" "$0" "$0" "$0" >&2
	exit 1
fi

exec goimports -local 'nksrv/' -w "$@"
