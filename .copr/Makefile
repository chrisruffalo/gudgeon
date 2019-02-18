# default output dir
mkfile_path:=$(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir:=$(abspath $(notdir $(patsubst %/,%,$(dir $(mkfile_path)))))
spec_path?=$(abspath $(mkfile_dir)/gudgeon.spec)
outdir?=$(abspath $(mkfile_dir)/../build/)

# get variables
VERSION?=$(shell git rev-parse --abbrev-ref HEAD)
GITHASH?=$(shell git rev-parse HEAD | head -c6)
NUMBER?=$(shell git tag | tail -n 1 | cut --complement -b 1)

.PHONY: srpm

srpm:
	env
	dnf install -y python2-rpkg rpkg || sudo dnf install -y python2-rpkg rpkg
	mkdir -p $(outdir)
	cat $(spec_path).template | sed 's/@VERSION_TOKEN@/$(NUMBER)/g' | sed 's/@HASH_TOKEN@/$(GITHASH)/g' > $(spec_path)
	rpkg srpm --spec $(spec_path) --outdir $(outdir)