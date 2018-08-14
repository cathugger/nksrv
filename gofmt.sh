#!/bin/sh
#exec find ./ -iname '*.go' -exec gofmt -s -w '{}' ';'
GOPATH=`pwd` exec goimports -local 'nekochan/' -w src/nekochan
