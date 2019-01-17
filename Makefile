# Go parameters
GOCMD=go
GODOWN=$(GOCMD) mod download

# use GOX to build certain architectures
GOOS_LIST?=linux
GARCH_LIST?=386 amd64 arm

# information
MAINTAINER=Chris Ruffalo
WEBSITE=https://github.com/gudgeon
DESCRIPTION=Gudgeon is a flexible blocking DNS proxy/cache

# go commands
GOBUILD=gox -os "$(GOOS_LIST)" -arch "$(GARCH_LIST)"
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# upx command (minimizes/compresses binaries)
UPXCMD=upx

# fpm command (gem for creating packages)
FPMCMD=fpm

# the build targets
BUILD_DIR=build
BINARY_NAME=gudgeon

# get version and hash from git if not passed in
VERSION?=$(shell git rev-parse --abbrev-ref HEAD)
GITHASH?=$(shell git rev-parse HEAD | head -c6)
NUMBER?=$(shell git tag | tail -n 1 | cut --complement -b 1)

# build targets for dockerized commands (build deb, build rpm)
OS_TYPE?=centos
OS_VERSION?=7
OS_BIN_ARCH?=amd64
OS_ARCH?=x86_64

# common FPM commands
FPMCOMMON=-a $(OS_ARCH) -n $(BINARY_NAME) -v $(NUMBER) --iteration $(GITHASH) --url "$(WEBSITE)" -m "$(MAINTAINER)" --config-files="/etc/gudgeon" --config-files="/etc/gudgeon/gudgeon.yml" --directories="/var/lib/$(BINARY_NAME)" --description "$(DESCRIPTION)" --before-install ./resources/before_install.sh --after-install ./resources/after_install.sh --after-upgrade ./resources/after_upgrade.sh --prefix / -C $(BUILD_DIR)/pkgtmp

all: test build minimize
.PHONY: all test build clean minimize rpm

build: ## Build Binary
		$(GODOWN)
		rm -f $(BUILD_DIR)/$(BINARY_NAME)*
		$(GOCMD) get github.com/mitchellh/gox
		$(GOBUILD) -cgo -tags netgo -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitHash=$(GITHASH) -extldflags \"static\"" -output "$(BUILD_DIR)/$(BINARY_NAME)_{{.OS}}_{{.Arch}}" -verbose

test: ## Do Unit Tests
		$(GODOWN)
		$(GOTEST) -v ./...

clean: ## Remove build artifacts
		$(GOCLEAN)
		rm -rf $(BUILD_DIR)

minimize: build ## Binimize build
		$(UPXCMD) -q $(BUILD_DIR)/$(BINARY_NAME)*
		rm -f $(BUILD_DIR)/*.upx

rpm: ## Build target linux/redhat RPM for $OS_BIN_ARCH/$OS_ARCH
		rm -rf $(BUILD_DIR)/pkgtmp
		rm -rf $(BUILD_DIR)/$(BINARY_NAME)*$(OS_ARCH).rpm
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)/lists
		mkdir -p $(BUILD_DIR)/pkgtmp/usr/bin/
		mkdir -p $(BUILD_DIR)/pkgtmp/var/lib/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/lib/systemd/system
		cp $(BUILD_DIR)/$(BINARY_NAME)_linux_$(OS_BIN_ARCH) $(BUILD_DIR)/pkgtmp/usr/bin/$(BINARY_NAME)
		cp ./resources/gudgeon.socket $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.socket
		cp ./resources/gudgeon.service $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.service
		cp ./resources/gudgeon.yml $(BUILD_DIR)/pkgtmp/etc/gudgeon/gudgeon.yml
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)-VERSION-$(GITHASH).ARCH.rpm" -t rpm $(FPMCOMMON)
		rm -rf $(BUILD_DIR)/pkgtmp

deb: ## Build deb file for $OS_BIN_ARCH/$OS_ARCH
		rm -rf $(BUILD_DIR)/pkgtmp
		rm -rf $(BUILD_DIR)/$(BINARY_NAME)*$(OS_BIN_ARCH).deb
		rm -rf $(BUILD_DIR)/$(BINARY_NAME)*$(OS_ARCH).deb
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)/lists
		mkdir -p $(BUILD_DIR)/pkgtmp/usr/bin/
		mkdir -p $(BUILD_DIR)/pkgtmp/var/lib/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/lib/systemd/system
		cp $(BUILD_DIR)/$(BINARY_NAME)_linux_$(OS_BIN_ARCH) $(BUILD_DIR)/pkgtmp/usr/bin/$(BINARY_NAME)
		cp ./resources/gudgeon.socket $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.socket
		cp ./resources/gudgeon.service $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.service
		cp ./resources/gudgeon.yml $(BUILD_DIR)/pkgtmp/etc/gudgeon/gudgeon.yml
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)_VERSION-$(GITHASH)_ARCH.deb" -t deb $(FPMCOMMON)
		rm -rf $(BUILD_DIR)/pkgtmp	