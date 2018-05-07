#!/bin/sh
#exec find ./ -iname '*.go' -exec gofmt -s -w '{}' ';'
GOPATH=`realpath .` exec goimports -local 'nekochan/' -w .
