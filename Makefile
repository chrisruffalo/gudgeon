# information
MAINTAINER=Chris Ruffalo
WEBSITE=https://github.com/gudgeon
DESCRIPTION=Gudgeon is a flexible blocking DNS proxy/cache

# get $(SED) command
SED=$(shell which gsed || which sed)

# enable CGO
CGO_ENABLED=1

# set relative paths
MKFILE_DIR:=$(abspath $(dir $(realpath $(firstword $(MAKEFILE_LIST)))))

# local arch (changed to standard names for build for debian/travis)
LOCALARCH=$(shell uname -m | $(SED) 's/x86_64/amd64/' | $(SED) -r 's/i?686/386/' | $(SED) 's/i386/386/' )

# enable passthrough of architecture flags to go
GOOS?=$(shell echo $(shell uname) | tr A-Z a-z)
GOARCH?=$(LOCALARCH)
GOARM?=
GOMIPS?=softfloat

# go commands and paths
GOPATH?=$(HOME)/go
GOBIN?=$(GOPATH)/bin/
GOCMD?=go
GODOWN=$(GOCMD) mod download
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# downloading things
CURLCMD=curl

# rice command
RICECMD=$(abspath $(GOBIN)/rice)
RICEPATHS=-i ./engine/ -i ./web/ -i ./rule/

# fpm command (gem for creating packages)
FPMCMD=fpm

# the build targets
BUILD_DIR=$(MKFILE_DIR)/build
BINARY_NAME=gudgeon

# get version and hash from git if not pas$(SED) in
VERSION?=$(shell git describe --tags $$(git rev-list --tags --max-count=1) | $(SED) -r -e 's/([^0-9.-]*)?-?v?([0-9.]*)-?([^-]*)?-?([^-]*)?/v\2/')
LONGVERSION?=$(shell git describe --tags | $(SED) 's/^$$/$(VERSION)/')
GITHASH?=$(shell git rev-parse HEAD | head -c7)
NUMBER?=$(shell echo $(LONGVERSION) | $(SED) -r -e 's/([^0-9.-]*)?-?v?([0-9.]*)-?([^-]*)?-?([^-]*)?/\2/' )
RELEASE?=$(shell echo $(LONGVERSION) | $(SED) -r -e 's/([^0-9.-]*)?-?v?([0-9.]*)-?([^-]*)?-?([^-]*)?/\3/' | $(SED) 's/^$$/1/' )
DESCRIPTOR?=$(shell echo $(LONGVERSION) | $(SED) -r -e 's/([^0-9.-]*)?-?v?([0-9.]*)-?([^-]*)?-?([^-]*)?/\1/' | $(SED) 's/^v$$//' | $(SED) 's/^\s$$//' )

# npm webpack
NPM?=$(shell which npm)
WEBPACK=$(MKFILE_DIR)/node_modules/.bin/webpack
WEBPACKCLI=$(WEBPACK)-cli

# docker stuff
DOCKER=$(shell which docker)
DOCKER_PATH?=gudgeon
DOCKER_NAME?=gudgeon
DOCKER_TAG?=$(NUMBER)
CONTAINER_PATH=$(DOCKER_PATH)/$(DOCKER_NAME):$(DOCKER_TAG)
DOCKERFILE?=Dockerfile

# build targets for dockerized commands (build deb, build rpm)
OS_TYPE?=$(GOOS)
OS_VERSION?=7
OS_BIN_ARCH?=amd64
OS_ARCH?=x86_64
BINARY_TARGET?=$(BINARY_NAME)-$(OS_TYPE)-$(OS_BIN_ARCH)

# set static flag
STATIC_FLAG?=-extldflags "-static"
ifeq ("$(GOOS)", "darwin")
  STATIC_FLAG=""
endif

# build tags can change by target platform, only linux builds for now though
GO_BUILD_TAGS?=netgo $(GOOS) sqlite3 jsoniter json1
GO_LD_FLAGS?=-s -w $(STATIC_FLAG) -X "github.com/chrisruffalo/gudgeon/version.Version=$(VERSION)" -X "github.com/chrisruffalo/gudgeon/version.GitHash=$(GITHASH)" -X "github.com/chrisruffalo/gudgeon/version.Release=$(RELEASE)" -X "github.com/chrisruffalo/gudgeon/version.Descriptor=$(DESCRIPTOR)"

# common FPM commands
FMPARCH?=$(shell echo "$(OS_ARCH)" | $(SED) -r 's/arm-?5/armhf/g' | $(SED) -r 's/arm64/armhf/g' | $(SED) -r 's/arm-?6/armhf/g' | $(SED) -r 's/arm-?7/armhf/g')
FPMCOMMON=-a $(FMPARCH) -n $(BINARY_NAME) -v $(NUMBER) --iteration "$(RELEASE)" --url "$(WEBSITE)" -m "$(MAINTAINER)" --config-files="/etc/gudgeon" --config-files="/etc/gudgeon/gudgeon.yml" --directories="/var/log/gudgeon" --directories="/var/lib/$(BINARY_NAME)" --description "$(DESCRIPTION)" --prefix / -C $(BUILD_DIR)/pkgtmp
FPMSCRIPTS=$(FPMCOMMON) --before-install $(MKFILE_DIR)/resources/before_install.sh --after-install $(MKFILE_DIR)/resources/after_install.sh

all: test build
.PHONY: all announce prepare test build clean minimize package rpm deb docker tar npm webpack bench hash

announce: ## Debugging versions mainly for build and travis-ci
		@echo "$(BINARY_NAME)" on "$(OS_ARCH)"
		@echo "=============================="
		@echo "longversion = $(LONGVERSION)"
		@echo "version = $(VERSION)"
		@echo "number = $(NUMBER)"
		@echo "release = $(RELEASE)"
		@echo "hash = $(GITHASH)"
		@echo "descriptor = $(DESCRIPTOR)"
		@echo "=============================="

prepare: ## Get all go tools and required libraries
		$(GOCMD) get -u github.com/GeertJohan/go.rice/rice
		$(GODOWN)

npm: ## download project npm dependencies
		$(NPM) install 	

webpack: ## prepare assets and build distribution
		$(NPM) run build:prod

build: announce  ## Build Binary
		$(GODOWN)
		# create build output dir
		mkdir -p $(BUILD_DIR)
		# create embeded resources
		$(RICECMD) embed-go $(RICEPATHS)
		$(GOBUILD) --tags "$(GO_BUILD_TAGS)" -ldflags "$(GO_LD_FLAGS)" -o "$(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(GOARM)" cmd/gudgeon.go
		# remove rice artifacts
		$(RICECMD) clean $(RICEPATHS)		

test: ## Do Unit Tests
		$(GODOWN)
		$(GOTEST) -v ./...

bench: ## Do tests and benchmarks
		$(GODOWN)
		$(GOTEST) -v ./... -bench .

clean: ## Remove build artifacts
		# do go clean steps
		$(GOCLEAN)
		# remove rice artifacts
		$(RICECMD) clean $(RICEPATHS)
		# remove dist from static assets
		rm -rf ./web/static/*
		# remove build dir
		rm -rf $(BUILD_DIR)

package: announce # Build consistent package structure
		rm -rf $(BUILD_DIR)/pkgtmp
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)/lists
		mkdir -p $(BUILD_DIR)/pkgtmp/usr/bin/
		mkdir -p $(BUILD_DIR)/pkgtmp/var/lib/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/lib/systemd/system
		mkdir -p $(BUILD_DIR)/pkgtmp/var/log/gudgeon
		cp $(BUILD_DIR)/$(BINARY_TARGET) $(BUILD_DIR)/pkgtmp/usr/bin/$(BINARY_NAME)
		cp $(MKFILE_DIR)/resources/gudgeon.socket $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.socket
		cp $(MKFILE_DIR)/resources/gudgeon.service $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.service
		cp $(MKFILE_DIR)/resources/gudgeon.yml $(BUILD_DIR)/pkgtmp/etc/gudgeon/gudgeon.yml	

rpm: package ## Build target linux/redhat RPM for $OS_BIN_ARCH/$OS_ARCH
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)-VERSION-$(RELEASE).$(OS_ARCH).rpm" -t rpm $(FPMSCRIPTS)
		rm -rf $(BUILD_DIR)/pkgtmp

deb: package ## Build deb file for $OS_BIN_ARCH/$OS_ARCH
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)_VERSION-$(RELEASE)_$(OS_ARCH).deb" -t deb $(FPMSCRIPTS)
		rm -rf $(BUILD_DIR)/pkgtmp

tar: announce ## Root directory TAR without systemd bits and a slightly different configuration
		rm -rf $(BUILD_DIR)/pkgtmp
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)/lists
		mkdir -p $(BUILD_DIR)/pkgtmp/usr/local/bin/
		mkdir -p $(BUILD_DIR)/pkgtmp/usr/local/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/var/log/gudgeon
		cp $(BUILD_DIR)/$(BINARY_TARGET) $(BUILD_DIR)/pkgtmp/usr/local/bin/$(BINARY_NAME)
		cp $(MKFILE_DIR)/resources/gudgeon-nosystemd.yml $(BUILD_DIR)/pkgtmp/etc/gudgeon/gudgeon.ym
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)-$(NUMBER)-$(GITHASH).$(OS_ARCH).tar" -t tar $(FPMCOMMON)
		gzip "$(BUILD_DIR)/$(BINARY_NAME)-$(NUMBER)-$(GITHASH).$(OS_ARCH).tar"
		rm -rf $(BUILD_DIR)/pkgtmp

docker: announce ## Create container and mark as latest as well
		$(DOCKER) build -f docker/$(DOCKERFILE) --build-arg BINARY_TARGET="$(BINARY_TARGET)" --rm -t $(CONTAINER_PATH) .
		$(DOCKER) tag $(CONTAINER_PATH) $(DOCKER_PATH)/$(DOCKER_NAME):latest

buildah: announce ## Create container with buildah
		./buildah.sh $(BINARY_TARGET) $(DOCKER_NAME)

dockerpush: ## Push image at path to remote
		$(DOCKER) push $(CONTAINER_PATH)

install:
		mkdir -p $(DESTDIR)/bin
		install -m 0755 $(BUILD_DIR)/$(BINARY_NAME)-$(OS_TYPE)-$(LOCALARCH) $(DESTDIR)/bin/$(BINARY_NAME)
		mkdir -p $(DESTDIR)/etc/gudgeon
		install -m 0664 $(MKFILE_DIR)/resources/gudgeon.yml $(DESTDIR)/etc/gudgeon/gudgeon.yml
		mkdir -p $(DESTDIR)/var/lib/gudgeon
		mkdir -p $(DESTDIR)/lib/systemd/system
		install -m 0644 $(MKFILE_DIR)/resources/gudgeon.socket $(DESTDIR)/lib/systemd/system/gudgeon.socket
		install -m 0644 $(MKFILE_DIR)/resources/gudgeon.service $(DESTDIR)/lib/systemd/system/gudgeon.service
		mkdir -p $(DESTDIR)/var/log/gudgeon

# build sha files for release artifacts
hash:
		# make hashes for all files in build directory
		find $(BUILD_DIR) -type f ! -name "*.sha*" -exec sh -c 'sha256sum $$0 > $$0.sha256' {} \;
