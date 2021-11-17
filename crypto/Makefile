ifeq ($(OS),Windows_NT)
  PLATFORM="Windows"
else
  ifeq ($(shell uname),Darwin)
    PLATFORM="MacOS"
  else
    PLATFORM="Linux"
  endif
endif

all: build 
export GO111MODULE=on
export GOFLAGS=-mod=vendor
export OUTPUT=./output/

build:
	PLATFORM=$(PLATFORM) ./build.sh

test:
	go test -cover ./...

clean:
	rm -rf $(OUTPUT)

.PHONY: all test clean
