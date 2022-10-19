module github.com/PaddlePaddle/PaddleDTX/dai

go 1.16

require (
	github.com/PaddlePaddle/PaddleDTX/crypto v0.0.0-20220705024525-b5b6c6a3ad76
	github.com/PaddlePaddle/PaddleDTX/xdb v0.0.0-20221019054435-b0e5dde7c0c5
	github.com/cjqpker/slidewindow v1.0.2
	github.com/docker/docker v1.4.2-0.20191101170500-ac7306503d23
	github.com/docker/go-connections v0.4.1-0.20180821093606-97c2040d34df
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hyperledger/fabric v1.4.4
	github.com/hyperledger/fabric-sdk-go v1.0.0-beta1
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/xuperchain/xuperchain v0.0.0-20210208123615-2d08ff11de3e
	golang.org/x/net v0.0.0-20210917221730-978cfadd31cf
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa
	google.golang.org/grpc v1.41.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
)

replace github.com/go-kit/kit => github.com/go-kit/kit v0.8.0
