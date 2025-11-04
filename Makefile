INSTALL_PATH ?= /usr/local/bin
GIT_SHA := $(shell git log -1 --pretty=format:"%H")
LD_FLAGS := "-X github.com/massdriver-cloud/mass/pkg/version.version=dev -X github.com/massdriver-cloud/mass/pkg/version.gitSHA=local-dev-${GIT_SHA}"

MASSDRIVER_PATH?=../massdriver
MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MKFILE_DIR := $(dir $(MKFILE_PATH))
API_DIR := pkg/api
SCHEMA_URL ?= https://api.massdriver.cloud/graphql/schema.graphql

.DEFAULT_GOAL := install

all.macos: clean generate install.macos
all.linux: clean generate install.linux

.PHONY: check
check: clean generate test ## Run tests and linter locally
	golangci-lint run

.PHONY: clean
clean:
	rm -rf ${API_DIR}/schema.graphql
	rm -rf ${API_DIR}/zz_generated.go
	rm -f ./mass

.PHONY: generate
generate:
	curl -s ${SCHEMA_URL} -o ${API_DIR}/schema.graphql
	cd ${API_DIR} && go generate

.PHONY:
swagger-gen:
	swag fmt -g cmd/server.go
	swag init -g cmd/server.go --pd --ot go,yaml

.PHONY: test
test:
	go test ./... -cover

bin:
	mkdir bin

.PHONY: lint
lint:
	golangci-lint run

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

.PHONY: install
install: ## Install mass CLI (auto-detects macOS or Linux)
	@OS="$$(uname -s)"; \
	if [ "$$OS" = "Darwin" ]; then \
		$(MAKE) install.macos; \
	elif [ "$$OS" = "Linux" ]; then \
		$(MAKE) install.linux; \
	else \
		echo "Unsupported OS: $$OS"; \
		exit 1; \
	fi
