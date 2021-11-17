#!/bin/sh
VERSION=1.0
serverBinary=xdata
clientBinary=xdata-cli
dataOwnerImageName=xdata-dataowner
storageImageName=xdata-storage
mirrorAddr=registry.baidubce.com/paddledtx

docker run -it --rm \
    -v ${PWD}:/workspace \
    -v ~/.ssh:/root/.ssh \
    -w /workspace \
    -e GONOPROXY=**.baidu.com** \
    -e GONOSUMDB=* \
    -e GOPROXY=https://goproxy.baidu-int.com \
    -e GO111MODULE=on \
    golang:1.13.4 bash -c "go build -o ./$serverBinary && go build -o ./$clientBinary cmd/client/main.go"
# build image 
docker rmi -f $mirrorAddr/$dataOwnerImageName:${VERSION}
docker rmi -f $mirrorAddr/$storageImageName:${VERSION}

docker build -t $mirrorAddr/$dataOwnerImageName:${VERSION} .
docker tag $mirrorAddr/$dataOwnerImageName:${VERSION} $mirrorAddr/$storageImageName:${VERSION} 

rm ./$serverBinary ./$clientBinary
