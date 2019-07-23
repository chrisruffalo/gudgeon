# default output dir
mkfile_path:=$(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir:=$(abspath $(notdir $(patsubst %/,%,$(dir $(mkfile_path)))))
spec_path?=$(abspath $(mkfile_dir)/gudgeon.spec)
outdir?=$(abspath $(mkfile_dir)/../build/)

LONGVERSION?=$$(git describe --tags)
VERSION?=$$(git describe --tags $$(git rev-list --tags --max-count=1) | sed -r -e 's/([^0-9.-]*)?-?v?([0-9.]*)-?([^-]*)?-?([^-]*)?/v\2/')
GITHASH?=$$(git rev-parse HEAD | head -c7)
NUMBER?=$$(echo $(LONGVERSION) | sed -r -e 's/([^0-9.-]*)?-?v?([0-9.]*)-?([^-]*)?-?([^-]*)?/\2/' )
RELEASE?=$$(echo $(LONGVERSION) | sed -r -e 's/([^0-9.-]*)?-?v?([0-9.]*)-?([^-]*)?-?([^-]*)?/\3/' | sed 's/^$$/1/' )

.PHONY: srpm
srpm:
	mkdir -p $(outdir)
	dnf install -y epel-release || sudo dnf install -y epel-release
	dnf update -y || sudo dnf update -y
	dnf install -y git python2-rpkg rpkg || sudo dnf install -y git python2-rpkg rpkg
	@echo "=============================="
	@echo "=============================="
	@echo "longversion = $(LONGVERSION)"
	@echo "version = $(VERSION)"
	@echo "number = $(NUMBER)"
	@echo "hash = $(GITHASH)"
	@echo "descriptor = $(DESCRIPTOR)"
	@echo "=============================="
	@echo "=============================="	
	cat $(spec_path).template | sed "s/@VERSION_TOKEN@/$(VERSION)/g" | sed "s/@HASH_TOKEN@/$(GIT_HASH)/g" | sed "s/@RELEASE_TOKEN@/$(RELEASE)/g" > $(spec_path)
	rpkg srpm --spec $(spec_path) --outdir $(outdir)
	rm -f $(spec_path)