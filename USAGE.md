# How to use NFS Provisioner Operator
### Using local storage for NFS server

[Local Storage Operator Doc](https://docs.openshift.com/container-platform/4.5/storage/persistent_storage/persistent-storage-local.html#local-storage-install_persistent-storage-local)

You need to login in OpenShift with  `cluster-admin` user before you follow the below commands

*Deploy Local Storage*
**Create NS/OperatorGorup/Subscription**
~~~
export product-verion=4.5      # <=== Update
echo"
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-local-storage
---
apiVersion: operators.coreos.com/v1alpha2
kind: OperatorGroup
metadata:
  name: local-operator-group
  namespace: local-storage
spec:
  targetNamespaces:
    - openshift-local-storage
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: local-storage-operator
  namespace: openshift-local-storage
spec:
  channel: "${product-version}" 
  installPlanApproval: Automatic
  name: local-storage-operator
  source: redhat-operators
  sourceNamespace: openshift-marketplace" |oc create -f -
~~~

**Deploy LocalVolume**
~~~

oc project openshift-local-storage

echo "
apiVersion: "local.storage.openshift.io/v1"
kind: "LocalVolume"
metadata:
  name: "local-disks"
  namespace: "openshift-local-storage" 
spec:
  nodeSelector: 
    nodeSelectorTerms:
    - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - worker-2.bell.tamlab.brq.redhat.com                 # <==== Update
  storageClassDevices:
    - storageClassName: "local-sc"
      volumeMode: Filesystem 
      fsType: xfs 
      devicePaths: 
        - /dev/vdb" | oc create -f -                            #<===== Update
~~~

### NFS Provisioner
**Create `CatalogSource`**
~~~
  cat <<EOF | oc apply -f -
  apiVersion: operators.coreos.com/v1alpha1
  kind: CatalogSource
  metadata:
    name: nfsprovisioner-catalog
    namespace: openshift-marketplace
  spec:
    sourceType: grpc
    image: quay.io/jooholee/nfs-provisioner-operator-index:0.0.2 
EOF
~~~

**Create a `project`**
~~~
oc new-project nfsprovisioner-operator
~~~

**Create `operatorgroup`**
~~~
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  annotations:
    olm.providedAPIs: NFSProvisioner.v1alpha1.cache.jhouse.com
  name: nfsprovisioner-operator
  namespace: nfsprovisioner-operator
spec:
  targetNamespaces:
  - nfsprovisioner-operator
~~~

**Create a subscription**
~~~
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: nfs-provisioner-operator
spec:
  channel: alpha
  installPlanApproval: Automatic
  name: nfs-provisioner-operator
  source: nfsprovisioner-catalog
  sourceNamespace: openshift-marketplace
  startingCSV: nfs-provisioner-operator.v0.0.2
~~~

**Deploy NFSProvisioner Operand**
~~~
echo "
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfsprovisioner-sample
spec:
  storageSize: "1G"
  scForNFSPvc: local-sc
  SCForNFSProvisioner: nfs"|oc create -f -
~~~

### Test
**Create `PVC` with NFS `StorageClass`**
~~~
echo "
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: nfs-pvc-example
  namespace: nfsprovisioner-operator
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Mi
  storageClassName: nfs"|oc create -f -
~~~