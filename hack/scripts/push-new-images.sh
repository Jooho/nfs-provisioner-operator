source ./env.sh

# Build/Push test NFS Operator
make podman-build podman-push

# Build/Push test Bundle image
make bundle-build bundle-push

# Build/Push test Index image
make index-build index-push


str="\n\n\n *******NEXT STEPS*******\n\n## Check a old NFS Provisioner is runinng ##\n$ UPGRADE_TEST=TRUE make deploy-nfs-cluster-olm\n$ make test-rw\n ## Check a new NFS Provisioner is running \n $ make deploy-nfs-cluster-olm-upgrade\n## if the /mnt/a exist, upgrade is working fine.##\n$ FILE_CHECK=TRUE make test-pod\n"

echo -e $str