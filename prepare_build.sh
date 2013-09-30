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
git checkout b25a0555844dea9ed4a45a85405ef853c1e5e918
cp -R codec ../src/codec
cd ..

echo "Run make to build"



