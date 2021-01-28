# Day 1 - Find necessary data for debugging 

In order to classify data properly, 

~~~
# Namespaces level 
- NFS Provisioner Operator (oc adm inspect)
- NFS Provisioner          (oc adm inspect)

## PVC 
commands_desc+=("storagecluster")
- Test PVC (RW)
- Check SElinux in specific path
- Test new PVC with NFS server
   - Create a PVC  
   - Check if file can be created
   - Clean the PVC

## OLM  (oc adm inspect olm)
commands_get+=("subscription")
commands_get+=("csv")
commands_get+=("installplan")


# Cluster level

commands_get+=("pv")
commands_get+=("ob")
commands_get+=("sc")
commands_get+=("nodes -o wide --show-labels")
commands_get+=("clusterversion")
commands_get+=("infrastructures.config")
commands_get+=("clusterrole")
commands_get+=("clusterrolebinding")
commands_get+=("scc")

# Operand level
- NFS config file
- Check NFS conf file
- Mount information
- showmount -e Service_IP
showmount -e : Shows the available shares on your local machine
showmount -e <server-ip or hostname>: Lists the available shares at the remote server
showmount -d : Lists all the sub directories
exportfs -v : Displays a list of shares files and options on a server
exportfs -a : Exports all shares listed in /etc/exports, or given name
exportfs -u : Unexports all shares listed in /etc/exports, or given name
exportfs -r : Refresh the server’s list after modifying /etc/exports
~~~


# Day 2 - Extract Sharable Variable

~~~

# Folder where tar ball will be stored
BASE_COLLECTION_PATH="must-gather"

# Gather Data since this value
SINCE_TIME = 0

# Delimeter for Sed command
SED_DELIMITER=$(echo -en "\001");

# Operator Name
OPERATOR_NAME="nfs-provisioner"

# CR Name
CR_NAME="NFSProvisioner"

# Test PVC Name
TEST_PVC_NAME="must-gather-test-pvc"

~~~




# Day 3 - Develop each gather 

- Create additional gather files and update gather file
  - gather_namespaced_resources
  - gather_cluster_resources
  - gather_operand_resources
  ~~~

  ~~~
~~~
# Check if PVC has Read/Write permission
can_readwrite_on_pvc(){
  oc 
}

~~~