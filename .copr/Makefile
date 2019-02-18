# default output dir
mkfile_path:=$(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir:=$(abspath $(notdir $(patsubst %/,%,$(dir $(mkfile_path)))))
spec_path?=$(abspath $(mkfile_dir)/gudgeon.spec)
outdir?=$(abspath $(mkfile_dir)/../build/)

.PHONY: srpm
srpm:
	mkdir -p $(outdir)
	dnf install -y git python2-rpkg rpkg || sudo dnf install -y git python2-rpkg rpkg
	cat $(spec_path).template | sed 's/@VERSION_TOKEN@/$(shell git tag | tail -n 1 | cut --complement -b 1)/g' | sed 's/@HASH_TOKEN@/$(shell git rev-parse HEAD | head -c6)/g' > $(spec_path)
	rpkg srpm --spec $(spec_path) --outdir $(outdir)