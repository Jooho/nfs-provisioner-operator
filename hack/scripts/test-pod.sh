source ./env.sh

#Clean
oc delete -f ${TEMPLATE_DIR}/pod.yaml

oc get pvc test-pvc
if [[ $? != 0 ]] 
then
  oc create -f ${TEMPLATE_DIR}/pvc.yaml
fi

oc create -f ${TEMPLATE_DIR}/pod.yaml

sleep 10

if [[ "Succeeded" != $(oc get pod test-pod -ojsonpath='{ .status.phase}') ]]
then
  echo "1"
else
  echo "0"
fi
