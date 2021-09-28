# Manual installation
NFS Provisioner Operator is provided through operatorhub but if you are familiar with CLI, you can use this manual way to install the operator.

## OLM
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
    image: quay.io/jooholee/nfs-provisioner-operator-index:0.0.2    #<=== Version Update
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
  startingCSV: nfs-provisioner-operator.v0.0.2                 #<=== Version Update
~~~


## CR (NFSProvisioner)
**Deploy NFSProvisioner Operand(CR)**
- local storage (refer [this](storage_option_localStorage.md))
  ~~~
  echo "
  apiVersion: cache.jhouse.com/v1alpha1
  kind: NFSProvisioner
  metadata:
    name: nfsprovisioner-sample
  spec:
    storageSize: "1G"
    scForNFSPvc: local-sc
    scForNFS: nfs"|oc create -f -
  ~~~

- gp2 storageclass on AWS
  ~~~
  echo "
  apiVersion: cache.jhouse.com/v1alpha1
  kind: NFSProvisioner
  metadata:
    name: nfsprovisioner-sample
    namespace: nfsprovisioner-operator
  spec:
    storageSize: "1G"
    scForNFSPvc: gp2
    scForNFS: nfs"|oc create -f -
  ~~~

- hostPath on prem cluster
  ~~~
  echo "
  apiVersion: cache.jhouse.com/v1alpha1
  kind: NFSProvisioner
  metadata:
    name: nfsprovisioner-sample
    namespace: nfsprovisioner-operator
  spec:
    nodeSelector: 
      kubernetes.io/hostname: worker-0.bell.tamlab.brq.redhat.com
    hostPathDir: "/home/core/test"
  ~~~
