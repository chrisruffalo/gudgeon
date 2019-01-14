# Go parameters
GOCMD=go
GODOWN=$(GOCMD) mod download
GOOS_LIST?=linux
GARCH_LIST?=386 amd64 arm
GOBUILD=gox -os "$(GOOS_LIST)" -arch "$(GARCH_LIST)"
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
UPXCMD=upx
BUILD_DIR=build
BINARY_NAME=gudgeon

ARCH_LIST?=

VERSION?=$(shell git rev-parse --abbrev-ref HEAD)
GITHASH?=$(shell git rev-parse HEAD | head -c6)

all: test build minimize
.PHONY: all test build clean minimize

build: 
		$(GODOWN)
		$(GOCMD) get github.com/mitchellh/gox
		rm -f $(BUILD_DIR)/$(BINARY_NAME)*
		$(GOBUILD) -cgo -tags netgo -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitHash=$(GITHASH) -extldflags \"static\"" -output "$(BUILD_DIR)/$(BINARY_NAME)_{{.OS}}_{{.Arch}}" -verbose

test: 
		$(GODOWN)
		$(GOTEST) -v ./...

clean: 
		$(GOCLEAN)
		rm -rf $(BUILD_DIR)

minimize: build
		$(UPXCMD) -q $(BUILD_DIR)/$(BINARY_NAME)*
		rm -f $(BUILD_DIR)/*.upx
