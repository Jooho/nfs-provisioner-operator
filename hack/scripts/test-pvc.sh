oc delete -f ${TEMPLATE_DIR}/pvc.yaml 
oc create -f  ${TEMPLATE_DIR}/pvc.yaml
sleep 10
oc get pvc test-pvc || echo "1"
