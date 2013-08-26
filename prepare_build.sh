#!/bin/bash

CUR_DIR=$(pwd)

echo "Set go path"
export GOPATH=$CUR_DIR

echo "Fetch go third-party GO package: codec"
#go get github.com/ugorji/go/codec
# Broken compability
git clone https://github.com/ugorji/go.git 
cd go
echo "Fetch proper revision"
git checkout 5a8bee99a997d101bbfa0d6c5f098e227fa44121
cp -R codec ../src/codec
cd ..

echo "Run make to build"



