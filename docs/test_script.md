# Test Scripts

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