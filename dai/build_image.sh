#!/bin/sh
VERSION=1.1
mirrorAddr=registry.baidubce.com/paddledtx

# build executor image
executorBinary=executor
executorClientBinary=executor-cli
requesterClientBinary=requester-cli

# use temporary container to compile output, making working image more simplified
docker run -it --rm \
    -v ${PWD}:/workspace \
    -v ~/.ssh:/root/.ssh \
    -w /workspace \
    -e GONOSUMDB=* \
    -e GOPROXY=https://goproxy.cn \
    -e GO111MODULE=on \
    golang:1.13.4 bash -c "go build -o ./bin/$executorBinary && go build -o ./bin/$executorClientBinary ./executor/cmd \
    && go build -o ./bin/$requesterClientBinary ./requester/cmd && chmod 777 ./bin" \

# build image
docker rmi -f $mirrorAddr/paddledtx-dai:${VERSION}
docker build -t $mirrorAddr/paddledtx-dai:${VERSION} .

# clean binary
rm -rf ./bin


