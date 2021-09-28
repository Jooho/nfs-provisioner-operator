source ./env.sh

# Clean All Objects
oc delete -f ${TEMPLATE_DIR}/pod.yaml
oc delete -f ${TEMPLATE_DIR}/pvc.yaml
oc delete -f ${TEMPLATE_DIR}/nfs.yaml

manager_count=$(oc get pod -l control-plane=controller-manager -n ${NAMESPACE}|wc -l)
if [[ ${manager_count} != 0 ]]; then make undeploy; fi

oc get subscription nfs-provisioner-operator-subs -n ${NAMESPACE}
if [[ $? == 0 ]]; then oc delete -f /tmp/nfs-subs.yaml; fi

oc get operatorgroup nfs-provisioner-operator-og -n ${NAMESPACE}
if [[ $? == 0 ]]; then oc delete -f /tmp/nfs-og.yaml; fi

oc get catalogsource nfs-provisioner-operator-cs -n openshift-marketplace
if [[ $? == 0 ]]; 
then 
oc delete -f /tmp/nfs-cs.yaml
oc delete scc nfs-provisioner
oc delete clusterrolebinding nfs-provisioner-runner
for i in $(oc get clusterrole|grep nfs|awk '{print $1}')
do 
  oc delete clusterrole $i
done

fi
