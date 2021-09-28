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

oc debug test-pod -- touch /mnt/a
oc debug test-pod -- ls /mnt/a

echo "$?"
