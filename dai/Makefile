# init project path
HOMEDIR := $(shell pwd)
OUTDIR  := $(HOMEDIR)/output

# init command params
GO      := go
GOROOT  := $(shell $(GO) env GOROOT)
GOPATH  := $(shell $(GO) env GOPATH)
GOMOD   := $(GO) mod
GOBUILD := $(GO) build
GOTEST  := $(GO) test -gcflags="-N -l"
GOPKGS  := $$($(GO) list ./...| grep -vE "vendor")

# make, make all
all: prepare compile package

# set proxy env
set-env:
	$(GO) env -w GO111MODULE=on
	$(GO) env -w GONOSUMDB=\*

# make build-pb 
build-pb: set-env	
	protoc -I protos protos/common/*.proto \
		--go_out=plugins=grpc,paths=source_relative:protos
	protoc -I protos protos/mpc/*.proto \
		--go_out=plugins=grpc,paths=source_relative:protos
	protoc -I protos protos/mpc/learners/linear_reg_vl/*.proto \
		--go_out=plugins=grpc,paths=source_relative:protos
	protoc -I protos protos/mpc/learners/logic_reg_vl/*.proto \
		--go_out=plugins=grpc,paths=source_relative:protos
	protoc -I protos protos/task/*.proto \
		--go_out=plugins=grpc,paths=source_relative:protos
#make prepare, download dependencies
prepare: gomod

gomod: set-env
	$(GOMOD) download

#make compile
compile: build

build:
	$(GOBUILD) -o $(HOMEDIR)/bin/executor
	$(GOBUILD) -o $(HOMEDIR)/bin/executor-cli ./executor/cmd
	$(GOBUILD) -o $(HOMEDIR)/bin/requester-cli ./requester/cmd

# make test, test your code
test: prepare test-case
test-case:
	$(GOTEST) -v -cover $(GOPKGS)

# make package
package: package-bin
package-bin:
	mkdir -p $(OUTDIR)
	mv -f bin/* $(OUTDIR)/
	cp -r conf $(OUTDIR)/
	rm -rf bin


# make clean
clean:
	$(GO) clean
	rm -rf $(OUTDIR)
	rm -rf $(GOPATH)/pkg/darwin_amd64

# avoid filename conflict and speed up build 
.PHONY: all prepare compile test package clean build
