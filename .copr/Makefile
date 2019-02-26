# default output dir
mkfile_path:=$(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir:=$(abspath $(notdir $(patsubst %/,%,$(dir $(mkfile_path)))))
spec_path?=$(abspath $(mkfile_dir)/gudgeon.spec)
outdir?=$(abspath $(mkfile_dir)/../build/)

LONGVERSION?=$$(git describe)
VERSION?=$$(git describe --tags $$(git rev-list --tags --max-count=1))
GITHASH?=$$(git rev-parse HEAD | head -c7)
NUMBER?=$$(echo $(LONGVERSION) | sed -r -e 's/(.*)?v(.*)-(.*)?-(.*)?/\2/' )
RELEASE?=$$(echo $(LONGVERSION) | sed -r -e 's/(.*)?v(.*)-(.*)?-(.*)?/\3/' | sed 's/^$$/1/' )

.PHONY: srpm
srpm:
	mkdir -p $(outdir)
	dnf install -y git python2-rpkg rpkg || sudo dnf install -y git python2-rpkg rpkg
	cat $(spec_path).template | sed "s/@VERSION_TOKEN@/$(VERSION)/g" | sed "s/@HASH_TOKEN@/$(GIT_HASH)/g" | sed "s/@RELEASE_TOKEN@/$(RELEASE)/g" > $(spec_path)
	rpkg srpm --spec $(spec_path) --outdir $(outdir)
	rm -f $(spec_path)