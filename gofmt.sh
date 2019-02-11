#!/bin/sh

#exec find ./ -iname '*.go' -exec gofmt -s -w '{}' ';'

if [ x"$1" = x'-u' ]
then
	go get -u golang.org/x/tools/cmd/goimports
	echo "Updated." >&2
	exit
fi

export GOPATH=`pwd`

if [ x"$1" = x"-all" ]
then
	exec goimports -local 'centpd/' -w src/centpd
fi

if [ "$#" -lt 1 ]
then
	printf "Usage:\n\t%s a.go ...\n\t%s -all\n\t%s -u\n" "$0" "$0" "$0" >&2
	exit 1
fi

exec goimports -local 'centpd/' -w "$@"
