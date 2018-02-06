#!/bin/sh
exec find ./ -iname '*.go' -exec gofmt -w '{}' ';'
