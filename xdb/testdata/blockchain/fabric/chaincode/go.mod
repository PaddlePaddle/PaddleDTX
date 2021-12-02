module chaincode

go 1.13

require (
	github.com/PaddlePaddle/PaddleDTX/crypto v0.0.0
	github.com/PaddlePaddle/PaddleDTX/xdb v0.0.0
	github.com/hyperledger/fabric v1.4.4
)

replace github.com/PaddlePaddle/PaddleDTX/crypto => icode.baidu.com/baidu/blockchain/PaddleDTX/crypto v0.0.0-20211126064712-ebf0cea6cd0f

replace github.com/PaddlePaddle/PaddleDTX/xdb => icode.baidu.com/baidu/blockchain/PaddleDTX/xdb v0.0.0-20211126073144-7dd28975e514
