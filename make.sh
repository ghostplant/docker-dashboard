#!/bin/bash -e

function compile {
	cd $(dirname $0)/ubuntu-docker
	sed -i 's/^ifeq.*nocheck.*$/ifeq (,0)/g' debian/rules
	sed -i 's/dockerd .* -H/dockerd -s btrfs --icc=false --userland-proxy=false -H/g' contrib/init/systemd/docker.service
	sed -i 's/nuke-graph-directory.sh/purge-graph-directory.sh/g' debian/docker.io.prerm

	dpkg-buildpackage -b || true
	dh_clean
	mv ../docker.io_*.deb ../.docker-next.deb
	rm -r ../*.deb ../docker.io_* ./.gopath
	mv ../.docker-next.deb ../docker-next.deb
}

time compile
