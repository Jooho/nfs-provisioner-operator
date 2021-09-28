source ./env.sh

# NFS Operator Namespace
oc project ${NAMESPACE}
if [[ $? != 0 ]] 
then
  oc new-project ${NAMESPACE}
fi

# Copy catalogSource yaml
cp ${TEMPLATE_DIR}/nfs-cs.yaml /tmp/nfs-cs.yaml
sed "s/0.0.1/${VERSION}/g" -i /tmp/nfs-cs.yaml
  
oc apply -f /tmp/nfs-cs.yaml