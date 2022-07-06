source ./env.sh

#Clean
oc delete -f ${TEMPLATE_DIR}/pod.yaml

oc get pvc test-pvc
if [[ $? != 0 ]] 
then
  oc create -f ${TEMPLATE_DIR}/pvc.yaml
fi

oc create -f ${TEMPLATE_DIR}/pod.yaml

if [[ FILE_CHECK != '' ]] 
then
  sleep 8
  oc exec test-pod  -- ls -al /mnt/
fi

sleep 15

if [[ "Succeeded" == $(oc get pod test-pod -ojsonpath='{ .status.phase}') ]]
then
  echo "Succeed. NFS is successful attached to POD"
else
  echo "Fail. NFS is NOT successful attached to POD"
fi
