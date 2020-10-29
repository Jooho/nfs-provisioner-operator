
export PRE_VERSION=0.0.1
export VERSION=0.0.2

export NEW_OP_NAME=nfs-provisioner-operator

export IMG=quay.io/jooholee/${NEW_OP_NAME}:${VERSION}
export BUNDLE_IMG=quay.io/jooholee/${NEW_OP_NAME}-bundle:${VERSION}
export INDEX_IMG=quay.io/jooholee/${NEW_OP_NAME}-index
export CHANNELS=alpha
export DEFAULT_CHANNEL=alpha


make generate
make manifests

make bundle bundle-build bundle-push
opm index add --bundles ${BUNDLE_IMG} --from-index ${INDEX_IMG}:${PRE_VERSION} --tag ${INDEX_IMG}:${VERSION} -c podman
podman push ${INDEX_IMG}:${VERSION}


