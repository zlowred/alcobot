#!/bin/sh
go generate
OP=`pwd`
cd $GOPATH/src/github.com/zlowred/goqt/bin
go build -ldflags "-r ." -o ~/alcobot/alcobot -v github.com/zlowred/alcobot
cd $OP
echo Done
