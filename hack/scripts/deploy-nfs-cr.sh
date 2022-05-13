source ./env.sh

# This is for local test by make run.


# Clean up
oc delete -f ${TEMPLATE_DIR}/nfs.yaml
sleep 5

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