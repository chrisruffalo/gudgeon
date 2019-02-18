# default output dir
mkfile_path:=$(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir:=$(abspath $(notdir $(patsubst %/,%,$(dir $(mkfile_path)))))
spec_path?=$(abspath $(mkfile_dir)/gudgeon.spec)
outdir?=$(abspath $(mkfile_dir)/../build/)

.PHONY: srpm

srpm:
	dnf install -y python2-rpkg rpkg || sudo dnf install -y python2-rpkg rpkg
	mkdir -p $(outdir)
	rpkg srpm --spec $(spec_path) --outdir $(outdir)