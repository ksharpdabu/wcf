#!/bin/bash

function build() {
    GOOS=$1
    GOARCH=$2 
    CGO_ENABLED=$3
    EXT=$4
    cur_dir=`pwd`
    cd wcf/src/wcf/cmd/local
    go build -o wcf_local_$1_$2$4
    cd -
    cd wcf/src/wcf/cmd/server
    go build -o wcf_server_$1_$2$4
    cd -
}

function tarall() {
    cd $1
    tar -czf $2 $3
    cd -
    mv $1/$2 $4
}

function main() {
    export GOPATH=$GOPATH:`pwd`"/wcf"
    build windows 386 0 ".exe"
    build windows amd64 0 ".exe"
    build linux 386 1 ""
    build linux amd64 1 ""
    build darwin 386 1 ""
    build darwin amd64 1 ""
    
    rm ./releases -rf 
    mkdir ./releases 
    dt=`date +%Y_%m_%d_%H_%M_%S`
    tarall wcf/src/wcf/cmd/local wcf_local_"$dt".tar.gz wcf_local_* ./releases/
    tarall wcf/src/wcf/cmd/server wcf_server_"$dt".tar.gz wcf_server_* ./releases/
}

main