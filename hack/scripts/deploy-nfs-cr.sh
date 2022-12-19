source ./env.sh

# This is for local test by make run.


# Clean up
oc delete -f ${TEMPLATE_DIR}/nfs.yaml
sleep 5

# Update tag 
cp ./hack/templates/nfs-hostpath.yaml /tmp/nfs-hostpath.yaml
cp ./hack/templates/nfs.yaml /tmp/nfs.yaml
sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner*/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./tmp/nfs-hostpath.yaml
sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner*/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./tmp/templates/nfs.yaml

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