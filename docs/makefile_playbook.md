# Makefile targets

## Env
- **CUSTOM_OLD_VERSION**
  - Set old operator version for upgrading test
- **UPGRADE_TEST**
  - Set TRUE, if you want to deploy old index for upgrade test.


## Deploy
- **deploy-op-local**
  - Install a new operator locally
- **deploy-nfs-cr**
  - Create a NFSProvisioner CR to deploy NFS server
- **deploy-nfs-cluster**
  - Install a new operator on a openshift cluster using `operatorsdk deploy` cmd
- **deploy-nfs-cluster-olm**
  - Install a new operator on a openshift cluster using `OLM`
- **deploy-nfs-cluster-olm-upgrade**
  - Replace old index image tag to a new index image tag. It will upgrade old operator to new operator



## Test
- **test-pvc**
  - Create a PVC object by NFS StorageClass
- **test-pod**
  - Create a test Pod to attach the PVC that is created by StorageClass
- **test-rw**
  - Create a PVC and attach it to a test pod. Then creating a folder in the pvc and reading it.
- **test-cleanup**
  - Cleanup all related objects



## Images
- **podman-build**
  - Create a new operator image
- **podman-push**
  -  Push a new operator image
- **bundle-build**
  - Create a new bundle image
- **bundle-push**
  - Push a new bundle image
- **index-build**
  - Create a new index image
- **index-push**
  - Push a new index image
- **push-new-images**
  - All in one target(podman-build/podman-push/bundle-build/bundle-push/index-build/index-push)