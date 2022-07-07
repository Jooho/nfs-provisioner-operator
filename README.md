# NFS Provisioner Go Operator 
![](https://img.shields.io/badge/openshift%204.8-tested-green)  ![](https://img.shields.io/badge/openshift%204.9-tested-green) ![](https://img.shields.io/badge/openshift%204.10-tested-green)

This operator deploy NFS server with serveral storage options and also provide provisioner for storageClass.

## Core Capabilities
* NFS Server: Deployed
* NFS Provisioner: Help customers to create PV using StorageClass
* StorageClass: Dynamically create PV for requested PVC
## NFS Provisioner Operator Features
* NFS Server can use localStorage PVC or HostPath on the node


Originally, this operator is created for sharing how to develop operator by Jooho Lee.
This is [the full tutorial page](https://github.com/Jooho/jhouse_openshift/blob/master/test_cases/operator/go-operator/nfs-provisioner-tutorial-docs/Tutorial-1-Go-Operator-without-logic.md)


## Documentation
- Storage Options
  - [localStorage](./docs/storage_option_localStorage.md)
  - [hostPath](./docs/storage_option_hostPath.md)
  - [storageClass](./config/samples/cache_v1alpha1_nfsprovisioner.yaml)
  - [PVC](./config/samples/cache_v1alpha1_nfsprovisioner_pvc.yaml)

- Development
  - [Makefile playbook](./docs/makefile_playbook.md)
  - [Build a new image](./docs/new_image.md)
  - [Test Scripts](./docs/test_script.md)
  - [Setup Dev Env](./docs/setup_development.md)

- [Manual Installation](./docs/manual_deploy.md)

