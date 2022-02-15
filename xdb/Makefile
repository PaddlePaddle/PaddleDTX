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
	$(GO) env -w GOPROXY=https://goproxy.cn
	$(GO) env -w GONOSUMDB=\*

#make prepare, download dependencies
prepare: gomod

gomod: set-env
	$(GOMOD) download

#make compile
compile: build

build:
	$(GOBUILD) -o $(HOMEDIR)/bin/xdb
	$(GOBUILD) -o $(HOMEDIR)/bin/xdb-cli ./cmd/client

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
