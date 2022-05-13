source ./env.sh


# Clean up
oc delete -f ${TEMPLATE_DIR}/nfs.yaml
sleep 5

# NFS Operator Namespace
oc project ${NAMESPACE}
if [[ $? != 0 ]] 
then
  oc new-project ${NAMESPACE}
fi

# Deploy Operator
make deploy 

# Deploy NFS
# oc create -f ${TEMPLATE_DIR}/nfs.yaml -n ${NAMESPACE}
oc create -f ${TEMPLATE_DIR}/nfs-hostpath.yaml -n ${NAMESPACE}
sleep 5


${SCRIPTS_DIR}/check-nfs-ready.sh