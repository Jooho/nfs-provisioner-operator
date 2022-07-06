# Process how to create a new image with testing.

When you try to release a new version of NFS provisioner with some reasons, you should test the new image with 4 times before pushing it. This doc explains the 4 steps.

## Scenario
- Latest version: 0.0.4
- New version: 0.0.5

## Local Test
- Set variables
  ~~~
  export CUSTOM_OLD_VERSION=0.0.4


  vi env.sh
  NAMESPACE=${OP_NAME}-test
  VERSION=0.0.5-test
  ~~~

- Cmds
  ~~~
  # Deploy a new version of operator locally
  make deploy-op-local

  # You should open a new terminal
  # Create a CR
  make deploy-nfs-cr

  # Verify NFS StorageClass
  make test-pvc

  # Clear all objects
  make test-cleanup
  ~~~

## Cluster Test

- Build/Push a new operator
  ~~~
  make podman-build podman-push
  ~~~

- Set variable to avoid conflicting namespaces
  ~~~
  source env.sh
  cd config/default;kustomize edit set namespace ${NAMESPACE} ; cd ../..
  ~~~

- Cmds
  ~~~
  # Deploy the new operator on a cluster
  make deploy-nfs-cluster
  
  # Create a CR
  make deploy-nfs-cr
  
  # Verify NFS StorageClass
  make test-pvc

  # Clear all objects
  make test-cleanup
  ~~~

- Roll back the variable
  ~~~
  export NAMESPACE=${OP_NAME}
  cd config/default;kustomize edit set namespace ${NAMESPACE} ; cd ../..
  ~~~

## Cluster OLM Test
It uses a index image to deploy the new operator
- Cmds
  ~~~
  # Create a new operator/bundle/index images
  make deploy-nfs-cluster-olm

  # Verify NFS StorageClass
  make test-pvc

  # Clear all objects
  make test-cleanup
  ~~~


## Cluster OLM upgrade Test
It deploys old index image to deploy old operator first and then deploy the new index image to see upgrade.

- Deploy the latest NFS Provisioner from operator hub
  ~~~
  UPGRADE_TEST=TRUE make deploy-nfs-cluster-olm
  ~~~

- Cmds
  ~~~
  make deploy-nfs-cluster-olm-upgrade

  # Check the version of operator. It should be upgraded because it is using the same channel.

  # Verify NFS StorageClass
  make test-pvc

  # Clear all objects
  make test-cleanup
  ~~~


## Push new images(operator/bundle/index)

- Set variables
  ~~~
  vi env.sh
  NAMESPACE=${OP_NAME}
  VERSION=0.0.5

  export CUSTOM_OLD_VERSION=0.0.4
  ~~~

- Cmds
  ~~~
  make push-new-images
  ~~~

## Last Upgrade Test

- Cmds
  ~~~
  UPGRADE_TEST=TRUE  make deploy-nfs-cluster-olm
  
  ## Check NFS Provisioner 0.0.4 is runinng 
  make test-rw

  make deploy-nfs-cluster-olm-upgrade
  
  ## Check if NFS Provisioner 0.0.5 is running and existing PVC have no issues.
  oc debug test-pod -- ls -al /mnt/a
  make test-cleanup

  ## if the /mnt/a exist, upgrade is working fine.
  ~~~