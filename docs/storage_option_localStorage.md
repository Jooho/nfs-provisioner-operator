# LocalStorage for NFS Server

### Install local storage 
[Local Storage Operator Document](https://docs.openshift.com/container-platform/4.8/storage/persistent_storage/persistent-storage-local.html)

You need to login in OpenShift with  `cluster-admin` user before you follow the below commands

**Script to create NS/OperatorGorup/Subscription**
~~~
export product-verion=4.8      # <=== Update
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
          - worker-2.openshiftcluster.com                       # <==== Update
  storageClassDevices:
    - storageClassName: "local-sc"
      volumeMode: Filesystem 
      fsType: xfs 
      devicePaths: 
        - /dev/vdb" | oc create -f -                            #<===== Update
~~~
