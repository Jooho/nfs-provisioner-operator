# Setup Environment for Go Operator Development

# Download Binary & Move them in the path(2022.07.05)
```
mkdir /tmp/operatorsdk
cd /tmp/operatorsdk

RELEASE_VERSION=v1.22.0
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk_linux_amd64
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/ansible-operator_linux_amd64
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/helm-operator_linux_amd64
curl -L https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v4.5.5/kustomize_v4.5.5_linux_amd64.tar.gz | tar xz
curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
curl -LO https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.5.0/kubebuilder_linux_amd64

chmod +x operator-sdk_linux_amd64 && sudo mv operator-sdk_linux_amd64 /usr/local/bin/operator-sdk
chmod +x ansible-operator_linux_amd64 && sudo mv ansible-operator_linux_amd64 /usr/local/bin/ansible-operator
chmod +x helm-operator_linux_amd64 && sudo mv helm-operator_linux_amd64 /usr/local/bin/helm-operator
chmod +x kustomize  && sudo mv kustomize /usr/local/bin/kustomize
chmod +x kubectl && sudo mv kubectl /usr/local/bin/kubectl
chmod +x kubebuilder_linux_amd64 && mv kubebuilder_linux_amd64 /usr/local/bin/kubebuilder

operator-sdk version 
ansible-operator version
helm-operator version
kustomize version
kubectl version
kubebuilder version
```


## Test
~~~
make test
~~~
