#!/bin/sh

#exec find ./ -iname '*.go' -exec gofmt -s -w '{}' ';'

[ x"$1" = x'-u' ] && go get -u golang.org/x/tools/cmd/goimports

GOPATH=`pwd` exec goimports -local 'nekochan/' -w src/nekochan
