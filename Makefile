BASE_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

all: build

build: ${BASE_DIR}/out/proxy

${BASE_DIR}/out/proxy:
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) go build -o ${BASE_DIR}/out/proxy ${BASE_DIR}/cmd/proxy/...

clean:
	rm -rf ${BASE_DIR}/out
