# NFS Provisioner Go Operator

This operator deploy NFS server with local storage and also provide provisioner for storageClass.

## Core Capabilities
* NFS Server: Deployed
* NFS Provisioner: Help customers to create PV using StorageClass
* StorageClass: Dynamically create PV for requested PVC
## NFS Provisioner Operator Features
* NFS Server can use localStorage PVC or HostPath on the node


Originally, this operator is created for sharing how to develop operator by Jooho Lee.
This is [the full tutorial page](https://github.com/Jooho/jhouse_openshift/blob/master/test_cases/operator/go-operator/nfs-provisioner-tutorial-docs/Tutorial-1-Go-Operator-without-logic.md)


## Usage
Please refer to the [USAGE.md](./USAGE.md) document for information

## Contributing
Please refer to the [CONTRIBUTING.md](./CONTRIBUTING.md) document for information.

