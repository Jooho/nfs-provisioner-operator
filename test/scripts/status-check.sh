#!/bin/bash
# Gather debug information after E2E test failures.
# Usage: ./test/scripts/status-check.sh

OPERATOR_NS="${OPERATOR_NAMESPACE:-nfs-provisioner-operator}"
E2E_NAMESPACES=("e2e-nfs-test" "e2e-nfs-sc-test")

echo "::group::Disk and Docker"
df -hT
docker image ls 2>/dev/null || true
echo "::endgroup::"

echo "::group::Nodes"
kubectl get nodes -o wide
echo "::endgroup::"

echo "::group::All Pods"
kubectl get pods -A
echo "::endgroup::"

echo "::group::Operator Pods in ${OPERATOR_NS}"
kubectl get pods -n ${OPERATOR_NS} -o wide
echo "::endgroup::"

echo "::group::Operator Controller Logs"
kubectl logs -l control-plane=controller-manager -n ${OPERATOR_NS} --all-containers=true --tail=200 2>&1 || true
echo "::endgroup::"

echo "::group::Describe Operator Pods"
for pod in $(kubectl get pods -n ${OPERATOR_NS} -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
  echo "--- Pod: $pod ---"
  kubectl describe pod -n ${OPERATOR_NS} $pod
  echo "--- End Pod: $pod ---"
done
echo "::endgroup::"

echo "::group::Events in ${OPERATOR_NS}"
kubectl get events -n ${OPERATOR_NS} --sort-by=.lastTimestamp
echo "::endgroup::"

echo "::group::NFSProvisioner CRs (all namespaces)"
kubectl get nfsprovisioner -A -o yaml 2>/dev/null || echo "No NFSProvisioner CRs found"
echo "::endgroup::"

echo "::group::NFS Server Pods"
for ns in ${OPERATOR_NS} "${E2E_NAMESPACES[@]}"; do
  if ! kubectl get namespace $ns &>/dev/null; then
    continue
  fi
  for pod in $(kubectl get pods -n $ns -l app=nfs-provisioner -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
    echo "--- NFS Pod: $ns/$pod ---"
    kubectl describe pod -n $ns $pod
    kubectl logs -n $ns $pod --tail=100 2>&1
    echo "--- End NFS Pod ---"
  done
done
echo "::endgroup::"

for ns in "${E2E_NAMESPACES[@]}"; do
  if ! kubectl get namespace $ns &>/dev/null; then
    continue
  fi

  echo "::group::Pods in ${ns}"
  kubectl get pods -n $ns -o wide
  echo "::endgroup::"

  echo "::group::Events in ${ns}"
  kubectl get events -n $ns --sort-by=.lastTimestamp
  echo "::endgroup::"

  echo "::group::Describe Pods in ${ns}"
  for pod in $(kubectl get pods -n $ns -o jsonpath='{.items[*].metadata.name}' 2>/dev/null); do
    echo "--- Pod: $pod ---"
    kubectl describe pod -n $ns $pod
    kubectl logs -n $ns $pod --all-containers=true --tail=200 2>&1
    echo "--- End Pod: $pod ---"
  done
  echo "::endgroup::"
done

echo "::group::Cluster-scoped Resources"
echo "--- StorageClasses ---"
kubectl get storageclass
echo "--- ClusterRoles (nfs) ---"
kubectl get clusterrole | grep nfs || echo "none"
echo "--- ClusterRoleBindings (nfs) ---"
kubectl get clusterrolebinding | grep nfs || echo "none"
echo "--- PVs ---"
kubectl get pv
echo "--- PVCs (all) ---"
kubectl get pvc -A
echo "::endgroup::"

echo "::group::SecurityContextConstraints (OpenShift)"
kubectl get scc nfs-provisioner -o yaml 2>/dev/null || echo "Not on OpenShift or SCC not found"
echo "::endgroup::"
