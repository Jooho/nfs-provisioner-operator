#!/bin/bash

# For Operator ISV
OPERATOR_NAME="nfs-provisioner-operator"

CUSTOM_RESOURCE_LIST=("NFSProvisioner")       #Syntax ("NFSProvisioenr" "NFSTest")

# Default values
# Folder where tar ball will be stored
BASE_COLLECTION_PATH="/opt/must-gather-root/must-gather"

# Current Namespace
INSTALL_NAMESPACE="${NAMESPACE}"

# Gather Data since this value
SINCE_TIME=0

# Delimeter for Sed command
SED_DELIMITER=$(echo -en "\001");

# Sharable methods
# Replace strings
safe_replace () {
    sed "s${SED_DELIMITER}${1}${SED_DELIMITER}${2}${SED_DELIMITER}g"
}
