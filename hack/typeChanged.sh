make generate
make manifests

make bundle bundle-build bundle-push
opm index add --bundles ${BUNDLE_IMG}  --tag ${INDEX_IMG}:${VERSION}
podman push ${INDEX_IMG}:${VERSION}


