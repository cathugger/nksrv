#!/bin/sh
export GOPATH=`realpath .`
exec go test "$@"
