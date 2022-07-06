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

oc exec test-pod -- touch /mnt/a
oc exec test-pod  -- ls /mnt/a

if [[ $? == '0' ]] 
then
  echo "Success. Test files are created"
else
  echo "Fail. Test files are not created"
fi
