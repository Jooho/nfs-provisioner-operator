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
  oc debug node/ip-10-0-168-107.ec2.internal
  chroot /host
  mkdir /home/core/test
  ~~~

- Add a special label for the node (ODS does not work to update label)
  ~~~
  oc label node/ip-10-0-168-107.ec2.internal app=nfs-provisioner
  ~~~

