source ./env.sh

# This is for local test by make run.


# Clean up
oc delete -f ${TEMPLATE_DIR}/nfs.yaml
sleep 5

# Update tag 
sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner*/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./hack/templates/nfs-hostpath.yaml
sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner*/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./hack/templates/nfs.yaml

# NFS Operator Namespace
oc project ${NAMESPACE}
if [[ $? != 0 ]] 
then
  oc new-project ${NAMESPACE}
fi

# Deploy NFS
# oc create -f ${TEMPLATE_DIR}/nfs.yaml
oc create -f ${TEMPLATE_DIR}/nfs-hostpath.yaml
sleep 5

${SCRIPTS_DIR}/check-nfs-ready.sh