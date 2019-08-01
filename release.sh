#!/usr/bin/env bash

rm -rf release
mkdir -p release
GOOS=darwin GOARCH=amd64 go build -v -o swapper-Darwin-x86_64
GOOS=linux GOARCH=amd64 go build -v -o swapper-Linux-x86_64

mv swapper-Darwin-x86_64 release/
mv swapper-Linux-x86_64 release/
