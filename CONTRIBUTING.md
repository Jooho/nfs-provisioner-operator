# Setup Environment for Go Operator Development

## Set environmental variables & Git clone
```
export ROOT_HOME=/home/jooho/dev/git    #Update
export REPO_NAME=RedHat  #Update(For OpenSource, use "jhouse_openshift")
export REPO_HOME=${ROOT_HOME}/${REPO_NAME}    
export DEMO_HOME=${REPO_HOME}/test_cases/operator/go-operator
export UTIL_HOME=${DEMO_HOME}/utils    
export TEST_HOME=${REPO_HOME}/test_cases/operator/test

# Set the release version for operator sdk
export RELEASE_VERSION=v1.0.1  #Update

cd ${ROOT_HOME}

git clone git@github.com:Jooho/${REPO_NAME}.git 

cd ${REPO_HOME}
```

## Download Binary & Move them in the path
```
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu
curl -L https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v3.8.4/kustomize_v3.8.4_linux_amd64.tar.gz | tar xz
curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"

chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mv operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk
chmod +x ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mv ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/ansible-operator
chmod +x helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mv helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/helm-operator
chmod +x kustomize  && sudo mv kustomize /usr/local/bin/kustomize
chmod +x kubectl && sudo mv kubectl /usr/local/bin/kubectl

${UTIL_HOME}/kubebuilder-install.sh
```


## Test
~~~
make test
~~~


## Local Test
- Run NFS Provisioner Operator
  ~~~
  oc new-project ${NAMESPACE}
  make run ENABLE_WEBHOOKS=false
  ~~~

- Create CR(NFSProvisioner)
  ~~~
  oc apply -f config/samples/cache_v1alpha1_nfsprovisioner.yaml 

  oc get pod
  ~~~

## Cluster Test
- Build/Push the image
  ~~~
  export IMG=quay.io/jooholee/nfs-provisioner-operator:test
  make podman-build podman-push IMG=${IMG}
  ~~~

- Run NFS Provisioner Operator
  ~~~
  # Update namespace
  cd config/default/; kustomize edit set namespace "${NAMESPACE}" ;cd ../..

  # Deploy Operator
  make deploy IMG=${IMG}

  oc project ${NAMESPACE}

  oc get pod
  ~~~

- Create CR(NFSProvisioner)
  ~~~
  oc apply -f config/samples/cache_v1alpha1_nfsprovisioner.yaml 

  oc get pod
  ~~~

