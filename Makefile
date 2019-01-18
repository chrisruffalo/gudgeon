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

# downloading things
CURLCMD=curl

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

# patternfly artifact
PFVERSION=3.59.1
PFARTIFACT=v$(PFVERSION)
PFPATH=patternfly-$(PFVERSION)

# common FPM commands
FPMCOMMON=-a $(OS_ARCH) -n $(BINARY_NAME) -v $(NUMBER) --iteration $(GITHASH) --url "$(WEBSITE)" -m "$(MAINTAINER)" --config-files="/etc/gudgeon" --config-files="/etc/gudgeon/gudgeon.yml" --directories="/var/lib/$(BINARY_NAME)" --description "$(DESCRIPTION)" --before-install ./resources/before_install.sh --after-install ./resources/after_install.sh --prefix / -C $(BUILD_DIR)/pkgtmp

all: test build minimize
.PHONY: all prepare test build clean minimize package rpm deb

prepare: ## Get all go tools
		$(GOCMD) get -u github.com/mitchellh/gox
		$(GOCMD) get -u github.com/GeertJohan/go.rice/rice

download: ## Download newest supplementary assets (todo: maybe replace with webpack?)
		mkdir -p ./build/download/
		mkdir -p ./build/vendor
		$(CURLCMD) https://github.com/patternfly/patternfly/archive/$(PFARTIFACT).tar.gz -L -o ./build/download/$(PFARTIFACT).tar.gz
		tar xf ./build/download/$(PFARTIFACT).tar.gz -C ./build/vendor

		rm -rf ./web/assets/vendor/*

		mkdir -p ./web/assets/vendor/img
		mkdir -p ./web/assets/vendor/fonts
		cp -r ./build/vendor/$(PFPATH)/dist/img ./web/assets/vendor/img
		cp -r ./build/vendor/$(PFPATH)/dist/fonts ./web/assets/vendor/fonts

		mkdir -p ./web/assets/vendor/css
		cp ./build/vendor/$(PFPATH)/dist/css/patternfly.min.css ./web/assets/vendor/css/patternfly.min.css
		cp ./build/vendor/$(PFPATH)/dist/css/patternfly-additions.min.css ./web/assets/vendor/css/patternfly-additions.min.css

		mkdir -p ./web/assets/vendor/js
		$(CURLCMD) https://cdn.jsdelivr.net/npm/vue -L -o ./web/assets/vendor/js/vue.min.js
		$(CURLCMD) https://unpkg.com/axios/dist/axios.min.js -L -o ./web/assets/vendor/js/axios.min.js
		$(CURLCMD) https://cdnjs.cloudflare.com/ajax/libs/jquery/3.2.1/jquery.min.js -L -o ./web/assets/vendor/js/jquery.min.js
		$(CURLCMD) https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.7/js/bootstrap.min.js -L -o ./web/assets/vendor/js/bootstrap.min.js
		cp ./build/vendor/$(PFPATH)/dist/js/patternfly.min.js ./web/assets/vendor/js/patternfly.min.js
		rm -rf ./build/download
		rm -rf ./build/vendor

build: ## Build Binary
		$(GODOWN)
		rm -f $(BUILD_DIR)/$(BINARY_NAME)*
		$(GOBUILD) -cgo --tags netgo -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitHash=$(GITHASH) -extldflags \"static\"" -output "$(BUILD_DIR)/$(BINARY_NAME)_{{.OS}}_{{.Arch}}_bin" -verbose
		$(UPXCMD) -q $(BUILD_DIR)/$(BINARY_NAME)*
		rm -f $(BUILD_DIR)/*.upx
		find $(BUILD_DIR) -name "$(BINARY_NAME)*_bin" | xargs -n 1 rice append -i ./web/ --exec

test: ## Do Unit Tests
		$(GODOWN)
		$(GOTEST) -v ./...

clean: ## Remove build artifacts
		$(GOCLEAN)
		rm -rf $(BUILD_DIR)

package: # Build consistent package structure
		rm -rf $(BUILD_DIR)/pkgtmp
		rm -rf $(BUILD_DIR)/$(BINARY_NAME)*$(OS_BIN_ARCH).deb
		rm -rf $(BUILD_DIR)/$(BINARY_NAME)*$(OS_ARCH).deb
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)/lists
		mkdir -p $(BUILD_DIR)/pkgtmp/usr/bin/
		mkdir -p $(BUILD_DIR)/pkgtmp/var/lib/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/lib/systemd/system
		cp $(BUILD_DIR)/$(BINARY_NAME)_linux_$(OS_BIN_ARCH)_bin $(BUILD_DIR)/pkgtmp/usr/bin/$(BINARY_NAME)
		cp ./resources/gudgeon.socket $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.socket
		cp ./resources/gudgeon.service $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.service
		cp ./resources/gudgeon.yml $(BUILD_DIR)/pkgtmp/etc/gudgeon/gudgeon.yml	

rpm: package ## Build target linux/redhat RPM for $OS_BIN_ARCH/$OS_ARCH
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)-VERSION-$(GITHASH).ARCH.rpm" -t rpm $(FPMCOMMON)
		rm -rf $(BUILD_DIR)/pkgtmp

deb: package ## Build deb file for $OS_BIN_ARCH/$OS_ARCH
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)_VERSION-$(GITHASH)_ARCH.deb" -t deb $(FPMCOMMON)
		rm -rf $(BUILD_DIR)/pkgtmp	