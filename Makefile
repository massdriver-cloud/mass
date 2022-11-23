INSTALL_PATH ?= /usr/local/bin
GIT_SHA := $(shell git log -1 --pretty=format:"%H")
LD_FLAGS := "-X github.com/massdriver-cloud/mass/pkg/version.version=dev -X github.com/massdriver-cloud/mass/pkg/version.gitSHA=local-dev-${GIT_SHA}"

MASSDRIVER_PATH?=../massdriver
MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MKFILE_DIR := $(dir $(MKFILE_PATH))
API_DIR := internal/api

.PHONY: clean
clean:
	rm -rf internal/api/{schema.graphql,zz_generated.go}

.PHONY: generate
generate: clean ${API_DIR}/zz_generated.go

.PHONY: test
test:
	go test ./... -cover

bin:
	mkdir bin

.PHONY: build.macos
build.macos: bin
	GOOS=darwin GOARCH=arm64 go build -o bin/mass-darwin-arm64 -ldflags=${LD_FLAGS}

.PHONY: build.linux
build.linux: bin
	GOOS=linux GOARCH=amd64 go build -o bin/mass-linux-amd64 -ldflags=${LD_FLAGS}

.PHONY: install.macos
install.macos: build.macos
	rm -f ${INSTALL_PATH}/mass
	cp bin/mass-darwin-arm64 ${INSTALL_PATH}/mass

.PHONY: install.linux
install.linux: build.linux
	cp -f bin/mass-linux-amd64 ${INSTALL_PATH}/mass

${API_DIR}/schema.graphql:
	cd ${MASSDRIVER_PATH} && mix absinthe.schema.sdl ${MKFILE_DIR}/${API_DIR}/schema.graphql

${API_DIR}/zz_generated.go: ${API_DIR}/schema.graphql
	cd ${API_DIR} && go generate
