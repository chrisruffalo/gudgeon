#!/bin/bash

# the path to the binary to copy in
BINARY=$1

# the commit name
COMMIT_TO=$2

echo "Adding ${BINARY} to ${COMMIT_TO}"

# start from scratch container
container=$(buildah from scratch)

# just need to copy the binary in and enable it
buildah ${container} copy ${BINARY} /gudgeon
buildah ${container} exec chmod +x /gudgeon

# set command

# commit changes to image
buildah commit ${COMMIT_TO}
