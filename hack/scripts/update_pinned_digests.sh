#!/bin/bash
source ./env.sh

function contains {
  local target=$1
  shift

  printf '%s\n' "$@" | grep -x -q "$target"
  out=$?
  (( out = 1 - out ))
  return $out
}

target=$1
podman pull gcr.io/kubebuilder/kube-rbac-proxy:${RBAC_PROXY_VERSION}

if [[ $? == 0 ]];then
    NEW_RBAC_PROXY_PINNED_DIGESTS=$(podman images --digests| grep gcr.io/kubebuilder/kube-rbac-proxy|grep ${RBAC_PROXY_VERSION}| awk '{print $3}')
    if [[ ${RBAC_PROXY_PINNED_DIGESTS} != ${NEW_RBAC_PROXY_PINNED_DIGESTS} ]];then
      sed "s/export RBAC_PROXY_PINNED_DIGESTS=.*/export RBAC_PROXY_PINNED_DIGESTS=${NEW_RBAC_PROXY_PINNED_DIGESTS}/g" -i env.sh
      echo "env.sh is updated for RBAC_PROXY_PINNED_DIGESTS"
    fi
else 
  echo "[ERROR] gcr.io/kubebuilder/kube-rbac-proxy:${RBAC_PROXY_VERSION} does not exist"
  exit 1
fi

podman pull k8s.gcr.io/sig-storage/nfs-provisioner:${NFS_SERVER_VERSION}
if [[ $? == 0 ]];then
    NEW_NFS_SERVER_PINNED_DIGESTS=$(podman images --digests| grep k8s.gcr.io/sig-storage/nfs-provisioner|grep ${NFS_SERVER_VERSION}| awk '{print $3}')

    if [[ ${NFS_SERVER_PINNED_DIGESTS} != ${NEW_NFS_SERVER_PINNED_DIGESTS} ]];then
      sed "s/export NFS_SERVER_PINNED_DIGESTS=.*/export NFS_SERVER_PINNED_DIGESTS=${NEW_NFS_SERVER_PINNED_DIGESTS}/g" -i env.sh
      echo "env.sh is updated for NFS_SERVER_PINNED_DIGESTS"
    fi
else 
  echo "[ERROR] k8s.gcr.io/sig-storage/nfs-provisioner:${NFS_SERVER_VERSION} does not exist"
  exit 1
fi


NEW_NFS_OPERATOR_PINNED_DIGESTS=$(docker images --digests| grep quay.io/jooholee/nfs-provisioner-operator | grep ${TAG} |grep -E -v 'index|bundle'| awk '{print $3}'|head -n 1) 
NFS_OPERATOR_PINNED_TAG=$(docker images --digests|grep ${NEW_NFS_OPERATOR_PINNED_DIGESTS}| awk '{print $2}' )

contains $TAG "${NFS_OPERATOR_PINNED_TAG[@]}"
tag_exist=$?

if [[ ${NFS_OPERATOR_PINNED_TAG} == "" ]];then
  echo "[ERROR] quay.io/jooholee/nfs-provisioner-operator:${TAG} does not exist"
  exit 1
fi
echo "tag_exist: $tag_exist"
if [[ ${tag_exist}  == '1'  ]];then
  sed "s/export NFS_OPERATOR_PINNED_DIGESTS=.*/export NFS_OPERATOR_PINNED_DIGESTS=${NEW_NFS_OPERATOR_PINNED_DIGESTS}/g" -i env.sh
  echo "env.sh is updated for NFS_OPERATOR_PINNED_TAG"
fi

if [[ ${NFS_OPERATOR_PINNED_DIGESTS} == "" ]] || [[ ${NFS_SERVER_PINNED_DIGESTS} == "" ]] || [[ ${RBAC_PROXY_PINNED_DIGESTS} == ""  ]]; then 
  echo "Pinned images does not exist"
  exit 1
fi

