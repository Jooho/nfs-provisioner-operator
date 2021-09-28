# Setup Environment for Go Operator Development

# Download Binary & Move them in the path
```
mkdir /tmp/operatorsdk
cd /tmp/operatorsdk

curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu
curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu
curl -L https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v3.8.4/kustomize_v3.8.4_linux_amd64.tar.gz | tar xz
curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"

curl -LO "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.1.0/kubebuilder_linux_amd64" 


chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mv operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk
chmod +x ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mv ansible-operator-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/ansible-operator
chmod +x helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu && sudo mv helm-operator-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/helm-operator
chmod +x kustomize  && sudo mv kustomize /usr/local/bin/kustomize
chmod +x kubectl && sudo mv kubectl /usr/local/bin/kubectl
chmod +x kubebuilder_linux_amd64 && mv kubebuilder /usr/local/bin/kubebuilder
```


## Test
~~~
make test
~~~
