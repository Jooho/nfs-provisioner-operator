## Operator Update
~~~
export OP_NAME=nfs-provisioner-operator
export OP_HOME=${ROOT_HOME}/operator-projects/${OP_NAME}
export NAMESPACE=${OP_NAME}
export VERSION=0.0.1
export IMG=quay.io/jooholee/${OP_NAME}:${VERSION}
~~~
make generate
make manifests
make podman-build podman-push 

## Bundle Update
~~~
export OP_NAME=nfs-provisioner-operator
export VERSION=0.0.1
export IMG=quay.io/jooholee/${OP_NAME}:${VERSION}
export BUNDLE_IMG=quay.io/jooholee/${OP_NAME}-bundle:${VERSION}

make bundle
make bundle-build bundle-push 
~~~

## Index Add
~~~
export OP_NAME=nfs-provisioner-operator
export VERSION=0.0.1
export IMG=quay.io/jooholee/${OP_NAME}:${VERSION}
export INDEX_IMG=quay.io/jooholee/${OP_NAME}-index
export BUNDLE_IMG=quay.io/jooholee/${OP_NAME}-bundle:${VERSION}

opm index add --bundles ${BUNDLE_IMG}  --tag ${INDEX_IMG}:${VERSION}
podman push ${INDEX_IMG}:${VERSION}
~~~
