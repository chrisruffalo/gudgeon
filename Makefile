# information
MAINTAINER=Chris Ruffalo
WEBSITE=https://github.com/gudgeon
DESCRIPTION=Gudgeon is a flexible blocking DNS proxy/cache

# set relative paths
MKFILE_DIR:=$(abspath $(dir $(realpath $(firstword $(MAKEFILE_LIST)))))

# local arch (changed to standard names for build for gox/debian/travis)
LOCALARCH=$(shell uname -m | sed 's/x86_64/amd64/' | sed 's/i686/386/' | sed 's/686/386/' | sed 's/i386/386/' )

# use GOX to build certain architectures
GOOS_LIST?=linux
GOARCH_LIST?=$(LOCALARCH)
XGO_TARGETS?=linux/arm-5,linux/arm-6,linux/mips

# go commands and paths
GOPATH?=$(HOME)/go
GOBIN?=$(GOPATH)/bin/
GOCMD?=go
GODOWN=$(GOCMD) mod download
GOXCMD=$(abspath $(GOBIN)/gox)
XGOCMD=$(abspath $(GOBIN)/xgo)
GOBUILD=$(GOXCMD) -os "$(GOOS_LIST)" -arch "$(GOARCH_LIST)"
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# downloading things
CURLCMD=curl

# SQLite binaries
SQLITE_DEP?=https://sqlite.org/2019/sqlite-autoconf-3270100.tar.gz

# rice command
RICECMD=$(abspath $(GOBIN)/rice)
RICEPATHS=-i ./qlog/ -i ./web/

# upx command (minimizes/compresses binaries)
UPXCMD=upx

# fpm command (gem for creating packages)
FPMCMD=fpm

# the build targets
BUILD_DIR=$(MKFILE_DIR)/build
BINARY_NAME=gudgeon

# get version and hash from git if not passed in
VERSION?=$(shell git rev-parse --abbrev-ref HEAD)
GITHASH?=$(shell git rev-parse HEAD | head -c7)
NUMBER?=$(shell git tag | sort -V | tail -n 1 | cut --complement -b 1)

# docker stuff
DOCKER=$(shell which docker)
DOCKER_PATH?=gudgeon
DOCKER_NAME?=gudgeon
DOCKER_TAG?=$(NUMBER)
CONTAINER_PATH=$(DOCKER_PATH)/$(DOCKER_NAME):$(DOCKER_TAG)
DOCKERFILE?=Dockerfile

	# build targets for dockerized commands (build deb, build rpm)
OS_TYPE?=linux
OS_VERSION?=7
OS_BIN_ARCH?=amd64
OS_ARCH?=x86_64
BINARY_TARGET?=$(BINARY_NAME)-$(OS_TYPE)-$(OS_BIN_ARCH)

# build tags can change by target platform, only linux builds for now though
GO_BUILD_TAGS?=netgo linux sqlite3 
GO_LD_FLAGS?=-s -w -extldflags "-static" -X main.Version=$(VERSION) -X main.GitHash=$(GITHASH)

# patternfly artifact
PFVERSION=3.59.1
PFARTIFACT=v$(PFVERSION)
PFPATH=patternfly-$(PFVERSION)

# common FPM commands
FPMCOMMON=-a $(OS_ARCH) -n $(BINARY_NAME) -v $(NUMBER) --iteration $(GITHASH) --url "$(WEBSITE)" -m "$(MAINTAINER)" --config-files="/etc/gudgeon" --config-files="/etc/gudgeon/gudgeon.yml" --directories="/var/lib/$(BINARY_NAME)" --description "$(DESCRIPTION)" --before-install $(MKFILE_DIR)/resources/before_install.sh --after-install $(MKFILE_DIR)/resources/after_install.sh --prefix / -C $(BUILD_DIR)/pkgtmp

all: test build minimize
.PHONY: all prepare test build clean minimize package rpm deb docker

prepare: ## Get all go tools and required libraries
		$(GOCMD) get -u github.com/karalabe/xgo
		$(GOCMD) get -u github.com/mitchellh/gox
		$(GOCMD) get -u github.com/GeertJohan/go.rice/rice

download: ## Download newest supplementary assets (todo: maybe replace with webpack?)
		mkdir -p $(BUILD_DIR)/download/
		mkdir -p $(BUILD_DIR)/vendor
		$(CURLCMD) https://github.com/patternfly/patternfly/archive/$(PFARTIFACT).tar.gz -L -o $(BUILD_DIR)/download/$(PFARTIFACT).tar.gz
		tar xf $(BUILD_DIR)/download/$(PFARTIFACT).tar.gz -C $(BUILD_DIR)/vendor

		rm -rf $(MKFILE_DIR)/web/assets/vendor/*

		mkdir -p $(MKFILE_DIR)/web/assets/vendor/img
		mkdir -p $(MKFILE_DIR)/web/assets/vendor/fonts
		cp $(BUILD_DIR)/vendor/$(PFPATH)/dist/img/* $(MKFILE_DIR)/web/assets/vendor/img/.
		cp $(BUILD_DIR)/vendor/$(PFPATH)/dist/fonts/* $(MKFILE_DIR)/web/assets/vendor/fonts/.

		mkdir -p $(MKFILE_DIR)/web/assets/vendor/css
		cp $(BUILD_DIR)/vendor/$(PFPATH)/dist/css/patternfly.min.css $(MKFILE_DIR)/web/assets/vendor/css/patternfly.min.css
		cp $(BUILD_DIR)/vendor/$(PFPATH)/dist/css/patternfly-additions.min.css $(MKFILE_DIR)/web/assets/vendor/css/patternfly-additions.min.css
		$(CURLCMD) https://cdn.jsdelivr.net/npm/vuetify/dist/vuetify.min.css -L -o $(MKFILE_DIR)/web/assets/vendor/css/vuetify.min.css

		mkdir -p $(MKFILE_DIR)/web/assets/vendor/js
		$(CURLCMD) https://cdn.jsdelivr.net/npm/vue -L -o $(MKFILE_DIR)/web/assets/vendor/js/vue.min.js
		$(CURLCMD) https://unpkg.com/axios/dist/axios.min.js -L -o $(MKFILE_DIR)/web/assets/vendor/js/axios.min.js
		$(CURLCMD) https://cdnjs.cloudflare.com/ajax/libs/jquery/3.2.1/jquery.min.js -L -o $(MKFILE_DIR)/web/assets/vendor/js/jquery.min.js
		$(CURLCMD) https://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.3.7/js/bootstrap.min.js -L -o $(MKFILE_DIR)/web/assets/vendor/js/bootstrap.min.js
		$(CURLCMD) https://cdn.jsdelivr.net/npm/vuetify/dist/vuetify.js -L -o $(MKFILE_DIR)/web/assets/vendor/js/vuetify.js
		cp $(BUILD_DIR)/vendor/$(PFPATH)/dist/js/patternfly.min.js $(MKFILE_DIR)/web/assets/vendor/js/patternfly.min.js
		rm -rf $(BUILD_DIR)/download
		rm -rf $(BUILD_DIR)/vendor

build: ## Build Binary
		$(GODOWN)
		mkdir -p $(BUILD_DIR)
		$(RICECMD) embed-go $(RICEPATHS)
		$(GOBUILD) -verbose -cgo --tags "$(GO_BUILD_TAGS)" -ldflags "$(GO_LD_FLAGS)" -output "$(BUILD_DIR)/$(BINARY_NAME)-{{.OS}}-{{.Arch}}"

buildxgo: ## Use xgo to build arm targets with sqlite installed, this only works **from inside the go path** (until xgo gets module support, anyway)
		mkdir -p $(BUILD_DIR)
		$(RICECMD) embed-go $(RICEPATHS)
		$(XGOCMD) --dest $(BUILD_DIR) --tags "$(GO_BUILD_TAGS)" --ldflags="$(GO_LD_FLAGS)" --targets="$(XGO_TARGETS)" --deps "$(SQLITE_DEP)" .

compress: ## Attempt to compress binaries with UPX
		$(UPXCMD) -q $(BUILD_DIR)/$(BINARY_NAME)* || true
		rm -f $(BUILD_DIR)/*.upx || true

test: ## Do Unit Tests
		$(GODOWN)
		$(GOTEST) -v ./...

clean: ## Remove build artifacts
		# do go clean steps
		$(GOCLEAN)
		# remove rice artifacts
		$(RICECMD) clean $(RICEPATHS)
		# remove build dir
		rm -rf $(BUILD_DIR)

package: # Build consistent package structure
		rm -rf $(BUILD_DIR)/pkgtmp
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/etc/$(BINARY_NAME)/lists
		mkdir -p $(BUILD_DIR)/pkgtmp/usr/bin/
		mkdir -p $(BUILD_DIR)/pkgtmp/var/lib/$(BINARY_NAME)
		mkdir -p $(BUILD_DIR)/pkgtmp/lib/systemd/system
		cp $(BUILD_DIR)/$(BINARY_TARGET) $(BUILD_DIR)/pkgtmp/usr/bin/$(BINARY_NAME)
		cp $(MKFILE_DIR)/resources/gudgeon.socket $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.socket
		cp $(MKFILE_DIR)/resources/gudgeon.service $(BUILD_DIR)/pkgtmp/lib/systemd/system/gudgeon.service
		cp $(MKFILE_DIR)/resources/gudgeon.yml $(BUILD_DIR)/pkgtmp/etc/gudgeon/gudgeon.yml	

rpm: package ## Build target linux/redhat RPM for $OS_BIN_ARCH/$OS_ARCH
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)-VERSION-$(GITHASH).ARCH.rpm" -t rpm $(FPMCOMMON)
		rm -rf $(BUILD_DIR)/pkgtmp

deb: package ## Build deb file for $OS_BIN_ARCH/$OS_ARCH
		$(FPMCMD) -s dir -p "$(BUILD_DIR)/$(BINARY_NAME)_VERSION-$(GITHASH)_ARCH.deb" -t deb $(FPMCOMMON)
		rm -rf $(BUILD_DIR)/pkgtmp

docker: ## Create container and mark as latest as well
		$(DOCKER) build -f docker/$(DOCKERFILE) --build-arg BINARY_TARGET="$(BINARY_TARGET)" --rm -t $(CONTAINER_PATH) .
		$(DOCKER) tag $(CONTAINER_PATH) $(DOCKER_PATH)/$(DOCKER_NAME):latest

dockerpush: ## Push image at path to remote
		$(DOCKER) push $(CONTAINER_PATH) .		

install:
		mkdir -p $(DESTDIR)/bin
		install -m 0755 $(BUILD_DIR)/$(BINARY_NAME)-$(OS_TYPE)-$(LOCALARCH) $(DESTDIR)/bin/$(BINARY_NAME)
		mkdir -p $(DESTDIR)/etc/gudgeon
		install -m 0664 $(MKFILE_DIR)/resources/gudgeon.yml $(DESTDIR)/etc/gudgeon/gudgeon.yml
		mkdir -p $(DESTDIR)/var/lib/gudgeon
		mkdir -p $(DESTDIR)/lib/systemd/system
		install -m 0644 $(MKFILE_DIR)/resources/gudgeon.socket $(DESTDIR)/lib/systemd/system/gudgeon.socket
		install -m 0644 $(MKFILE_DIR)/resources/gudgeon.service $(DESTDIR)/lib/systemd/system/gudgeon.service