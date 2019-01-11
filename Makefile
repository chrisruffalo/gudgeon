# Go parameters
GOCMD=go
GODOWN=$(GOCMD) mod download
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
UPXCMD=upx
BUILD_DIR=build
BINARY_NAME=gudgeon

VERSION:=$(shell git rev-parse --abbrev-ref HEAD)
GITHASH:=$(shell git rev-parse HEAD | head -c6)

all: test build minimize
.PHONY: all test build clean minimize

build: 
		$(GODOWN)
		$(GOBUILD) -a -tags netgo -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitHash=$(GITHASH) -extldflags \"static\"" -o $(BUILD_DIR)/$(BINARY_NAME) -v

test: 
		$(GODOWN)
		$(GOTEST) -v ./...

clean: 
		$(GOCLEAN)
		rm -rf $(BUILD_DIR)

minimize: build
		$(UPXCMD) -q $(BUILD_DIR)/$(BINARY_NAME)
		rm -f $(BUILD_DIR)/$(BINARY_NAME).upx
