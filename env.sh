export OP_NAME=nfs-provisioner-operator
export NAMESPACE=${OP_NAME}
export VERSION=0.0.7
export TAG=${VERSION}
export NFS_SERVER_VERSION=v3.0.1
export RBAC_PROXY_VERSION=v0.5.0
export NFS_OPERATOR_PINNED_DIGESTS=sha256:ce226ddcda4fb81e1ba4ace5a9bdd6502facbbd4a3aef2279b1ae1e8cd294ca1
export NFS_SERVER_PINNED_DIGESTS=sha256:e943bb77c7df05ebdc8c7888b2db289b13bf9f012d6a3a5a74f14d4d5743d439
export RBAC_PROXY_PINNED_DIGESTS=sha256:e10d1d982dd653db74ca87a1d1ad017bc5ef1aeb651bdea089debf16485b080b

export HACK_DIR=./hack
export TEMPLATE_DIR=${HACK_DIR}/templates
export SCRIPTS_DIR=${HACK_DIR}/scripts
