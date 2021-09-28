source ./env.sh

# Check required param CUSTOM_OLD_VERSION 
if [[ ${CUSTOM_OLD_VERSION} == '' ]]
then
  echo "Please set CUSTOM_OLD_VERSION param"
  exit 1
else
  echo "CUSTOM_OLD_VERSION=${CUSTOM_OLD_VERSION}"
fi


# Clean up
oc delete -f ${TEMPLATE_DIR}/nfs.yaml
sleep 5

# NFS Operator Namespace
oc project ${NAMESPACE}
if [[ $? != 0 ]] 
then
  oc new-project ${NAMESPACE}
fi

if [[ ${UPGRADE_TEST} == '' ]]
then
  # Build/Push test NFS Operator
  make podman-build podman-push

  # Build/Push test Bundle image
  make bundle-build bundle-push

  # Build/Push test Index image
  make index-build index-push
fi
# Copy catalogSource yaml
cp ${TEMPLATE_DIR}/nfs-cs.yaml /tmp/nfs-cs.yaml

# Copy/Update OperatorGroup yaml
cp ${TEMPLATE_DIR}/nfs-og.yaml /tmp/nfs-og.yaml
sed "s/nfs-provisioner-operator-ns/${NAMESPACE}/g" -i /tmp/nfs-og.yaml

# Copy Subscription yaml
cp ${TEMPLATE_DIR}/nfs-subs.yaml /tmp/nfs-subs.yaml
sed "s/nfs-provisioner-operator-ns/${NAMESPACE}/g" -i /tmp/nfs-subs.yaml

#Update CatalogSource/Subscription yaml
if [[ ${UPGRADE_TEST} == '' ]]
then
  sed "s/0.0.1/${VERSION}/g" -i /tmp/nfs-cs.yaml
  sed "s/0.0.1/${VERSION}/g" -i /tmp/nfs-subs.yaml
else
  sed "s/0.0.1/${CUSTOM_OLD_VERSION}/g" -i /tmp/nfs-cs.yaml
  sed "s/0.0.1/${CUSTOM_OLD_VERSION}/g" -i /tmp/nfs-subs.yaml
fi

# Setup OLM stuff
oc get catalogsource nfsprovisioner-catalogsource -n openshift-marketplace
if [[ $? != 0 ]]; then oc create -f /tmp/nfs-cs.yaml; fi

oc get operatorgroup nfsprovisioner-operator -n ${NAMESPACE}
if [[ $? != 0 ]]; then oc create -f /tmp/nfs-og.yaml; fi

oc get subscription nfs-provisioner-operator -n ${NAMESPACE}
if [[ $? != 0 ]]; then oc create -f/tmp/nfs-subs.yaml; fi

# Deploy NFS
oc create -f ${TEMPLATE_DIR}/nfs.yaml
sleep 5


${SCRIPTS_DIR}/check-nfs-ready.sh