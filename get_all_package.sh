#!/bin/bash

export GOPATH=$GOPATH:`pwd`"/wcf"

cd wcf

go get -u github.com/juju/ratelimit
go get -u github.com/sirupsen/logrus
go get -u github.com/mattn/go-sqlite3
go get -u github.com/golang/protobuf/proto
go get -u github.com/xtaci/kcp-go
go get -u github.com/xtaci/smux