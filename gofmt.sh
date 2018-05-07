#!/bin/sh
export GOPATH=`realpath .`
find ./ -iname '*.go' -exec gofmt -s -w '{}' ';'
goimports -w .
