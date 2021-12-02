#!/bin/bash

OUTPUT=output
mkdir -p $OUTPUT

echo 'start build plugins'
go build --buildmode=plugin -o $OUTPUT/crypto-default.so.1.0.0 github.com/PaddlePaddle/PaddleDTX/crypto/client/xchain/
