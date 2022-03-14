#!/bin/sh
VERSION=1.1
serverBinary=xdb
clientBinary=xdb-cli
dataOwnerImageName=xdb-dataowner
storageImageName=xdb-storage
mirrorAddr=registry.baidubce.com/paddledtx

docker run -it --rm \
    -v ${PWD}:/workspace \
    -v ~/.ssh:/root/.ssh \
    -w /workspace \
    -e GONOSUMDB=* \
    -e GOPROXY=https://goproxy.cn \
    -e GO111MODULE=on \
    golang:1.13.4 bash -c "go build -o ./$serverBinary && go build -o ./$clientBinary cmd/client/main.go"
# build image 
docker rmi -f $mirrorAddr/$dataOwnerImageName:${VERSION}
docker rmi -f $mirrorAddr/$storageImageName:${VERSION}

docker build -t $mirrorAddr/$dataOwnerImageName:${VERSION} .
docker tag $mirrorAddr/$dataOwnerImageName:${VERSION} $mirrorAddr/$storageImageName:${VERSION} 

rm -f ./$serverBinary ./$clientBinary
