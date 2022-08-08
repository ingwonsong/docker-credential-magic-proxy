BASE_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

export HUB ?= gcr.io/igsong-oss
export KO_DOCKER_REPO = $(HUB)/docker-credential-magic-proxy

all: publish

publish:
	ko publish --platform=linux/amd64 -B github.com/ingwonsong/docker-credential-magic-proxy/cmd/proxy -t latest
	docker-credential-magician mutate gcr.io/igsong-oss/docker-credential-magic-proxy/proxy:latest

publish-debug:
	KO_CONFIG_PATH=${BASE_DIR}/.ko/debug ko publish --platform=linux/amd64 -B github.com/ingwonsong/docker-credential-magic-proxy/cmd/proxy -t latest
	docker-credential-magician mutate gcr.io/igsong-oss/docker-credential-magic-proxy/proxy:latest

build: ${BASE_DIR}/out/proxy

${BASE_DIR}/out/proxy:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${BASE_DIR}/out/proxy ${BASE_DIR}/cmd/proxy/...

clean:
	rm -rf ${BASE_DIR}/out
