source ./env.sh

# Build/Push test NFS Operator
make podman-build podman-push

# Build/Push test Bundle image
make bundle-build bundle-push

# Build/Push test Index image
make index-build index-push


str="\n\n\n *******NEXT STEPS*******\n\n$ UPGRADE_TEST=TRUE make deploy-nfs-cluster-olm\n## Check a old NFS Provisioner is runinng ##\n$ make test-rw\nmake deploy-nfs-cluster-olm\n## Check a new NFS Provisioner is running \n$ make test-pod\noc debug test-pod -- ls -al /mnt/\n## if the /mnt/a exist, upgrade is working fine.##"

echo -e $str