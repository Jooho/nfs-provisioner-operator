# Setup HostPath

## Steps

- Check a node that NFS server will be deployed
  ~~~
  oc get node
  ...
  ip-10-0-168-107.ec2.internal   Ready    worker         24h   v1.21.1+d8043e1
  ...
  ~~~

- Create a directory on the specific node
  ~~~ 
  export targetNode=$(oc get node -l node-role.kubernetes.io/worker -o name|cut -d/ -f2|head -n 1)
  
  oc debug node/${targetNode} 

  chroot /host
  mkdir /home/core/nfs
  chcon -Rvt svirt_sandbox_file_t /home/core/nfs
  ~~~

- Add a special label for the node (ODS does not work to update label)
  ~~~
  oc label node ${targetNode} app=nfs-provisioner
  ~~~

