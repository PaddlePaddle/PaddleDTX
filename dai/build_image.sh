#!/bin/sh
VERSION=1.0
mirrorAddr=registry.baidubce.com/paddledtx

# build executor image
executorBinary=executor
executorClientBinary=executor-cli
requesterClientBinary=requester-cli
#使用临时容器编译私有化产出，使得工作镜像更精简
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


