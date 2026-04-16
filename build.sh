#!/bin/bash
cd $(dirname $(readlink -f $0))
GOOS=windows GOARCH=amd64 go build -ldflags -H=windowsgui .
