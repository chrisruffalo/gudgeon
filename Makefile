# Go parameters
GOCMD=go
GODOWN=$(GOCMD) mod download
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BUILD_DIR=build
BINARY_NAME=gudgeon

VERSION:=$(shell git rev-parse --abbrev-ref HEAD)
GITHASH:=$(shell git rev-parse HEAD | head -c6)

all: test build
build: 
		$(GODOWN)
		$(GOBUILD) -a -tags netgo -ldflags "-w -X main.Version=$(VERSION) -X main.GitHash=$(GITHASH) -extldflags \"static\"" -o $(BUILD_DIR)/$(BINARY_NAME) -v
test: 
		$(GODOWN)
		$(GOTEST) -v ./...
clean: 
		$(GOCLEAN)
		rm -rf $(BUILD_DIR)
			
