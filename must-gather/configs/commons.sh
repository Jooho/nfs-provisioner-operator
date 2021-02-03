#!/bin/bash

# For Operator ISV
OPERATOR_NAME="nfs-provisioner-operator"

CUSTOM_RESOURCE_LIST=("NFSProvisioner")       #Syntax ("NFSProvisioenr" "NFSTest")

# Default values
# Folder where tar ball will be stored
BASE_COLLECTION_PATH="/opt/must-gather-root/must-gather"

# absolute path
BASE_DIR="$(cd "$(dirname ./ )" && pwd)"
SCRIPT_DIR="$(cd "${BASE_DIR}/collection-scripts" && pwd)"
CONFIG_DIR="$(cd "${BASE_DIR}/configs" && pwd)"

# necessary binaries will be stored.
ADDITIONAL_BINARY_PATH="${BASE_DIR}/exteranl_bin"

# Current Namespace
INSTALL_NAMESPACE="${NAMESPACE}"

# Set TOKEN
TOKEN_PATH="/var/run/secrets/kubernetes.io/serviceaccount/token"
if [ -f ${TOKEN_PATH} ] 
then 
    TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
else 
    TOKEN=""
fi

# Gather Data since this value
SINCE_TIME=0

# Delimeter for Sed command
SED_DELIMITER=$(echo -en "\001");

# Sharable methods
# Replace strings
safe_replace () {
    sed "s${SED_DELIMITER}${1}${SED_DELIMITER}${2}${SED_DELIMITER}g"
}

is_not_nothing () {
    resource=$1
    if [[ z$(oc get ${resource} --ignore-not-found) == 'z' ]]
    then
        return 1
    else 
        return 0
    fi
}