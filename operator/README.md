# operator
Azure Pipelines Continues Deployment Target Operator

## Description
Automate the configuration & lifecycle of Azure self-hosted pipelines agents and enable self-service for adding egress targets, without the need of delegating full network policy permissions to the namespace administrator. Event driven autoscaling is automatically enabled trough standard KEDA and Azure pipelines integrations.

###  Describing the problem
For us as namespace administrators (cluster users) the CRUD functionality on network policy objects are unauthorized by security design and can only be changed by the cluster administrators. To enable end tot end automation, we need the abillity to add target IPs ourselves to a specified set of allowed egress ports trough a Custom Resource, the ports are specified by the the cluster administrators from one central config. An Operator should automatically create or update a network policy containing the specified IPs defined in the Custom resource. The operator should als configure and manage the lifecycle of the self-hosted pipeline agents and simplify the enablement of event driven autoscaling.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
#### Build Operator image
```bash
# docker and github repo username
export USERNAME='user'
# image and bundle version
export VERSION=0.0.1
# operator repo and name
export OPERATOR_NAME='cdtarget-operator'

#######################################################
# Build the operator image
make docker-build docker-push IMG=docker.io/$USERNAME/$OPERATOR_NAME:v$VERSION
```

## Operator lifecycle manager Deployment
```bash
#######################################################
# install OLM (if not already present)
operator-sdk olm install
operator-sdk olm status

#######################################################
# Build the OLM bundle
make bundle IMG=docker.io/$USERNAME/$OPERATOR_NAME:v$VERSION   
make bundle-build bundle-push BUNDLE_IMG=docker.io/$USERNAME/$OPERATOR_NAME-bundle:v$VERSION
```

```bash
# Deploy OLM bundle
kubectl create ns 'cdtarget-operator'
operator-sdk run bundle docker.io/$USERNAME/$OPERATOR_NAME-bundle:v$VERSION --namespace='cdtarget-operator'
```

```bash
# configmap to specify the ports
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: cdtarget-ports
  namespace: cdtarget-operator
data:
  ports: | 
    443
    22
    5986
    5432
EOF

#######################################################
# test cdtarget CR 
kubectl create ns test
# prestage the PAT (token) Secret for succesfull Azure AUTH
kubectl -n test create secret generic cdtarget-token \
                  --from-literal=AZP_TOKEN=$PAT
# apply cdtarget resource
# for scaling >1 replica don`t set the agentName field in the CR
kubectl -n test apply -f ../cnad_cdtarget_sample.yaml
kubectl -n test describe cdtarget cdtarget-agent
# test
kubectl -n test describe networkpolicies cdtarget-agent
kubectl -n test describe deployment cdtarget-agent
```

### Remove CR, CRD & Operator bundle
```bash
# cleanup test deployment
kubectl -n test delete -f ../cnad_cdtarget_sample.yaml
kubectl -n test delete secret cdtarget-proxy cdtarget-token
kubectl delete ns test
# cleanup OLM bundle & OLM installation
operator-sdk cleanup operator --delete-all --namespace='cdtarget-operator'
kubectl delete ns 'cdtarget-operator'
```

### Enable Proxy config
```bash
# update secret containing proxy settings
kubectl -n test create secret generic cdtarget-proxy --dry-run=client -o yaml \
                  --from-literal=PROXY_USER='username' \
                  --from-literal=PROXY_PW='password' \
                  --from-literal=PROXY_URL='http://user:password@proxy.gofound.nl:8080' \
                  --from-literal=HTTP_PROXY='http://proxy.gofound.nl:8080' \
                  --from-literal=HTTPS_PROXY='https://proxy.gofound.nl:8080' \
                  --from-literal=FTP_PROXY='' \
                  --from-literal=NO_PROXY='' | kubectl apply -f -
kubectl -n test scale deployment cdtarget-agent --replicas=0  
```

### Update Personal Access Token
```bash
# update CDTarget PAT
kubectl -n test create secret generic cdtarget-token --dry-run=client -o yaml \
                  --from-literal=AZP_TOKEN=$PAT | kubectl apply -f -
kubectl -n test scale deployment cdtarget-agent --replicas=0  
```

### Update allowed ports
```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: cdtarget-ports
  namespace: cdtarget-operator
data:
  ports: | 
    443
    22
    5986
    5432
    1433
EOF
```


### Uninstall Operator Lifecycle Manager
```bash
# uninstall OLM
operator-sdk olm uninstall
```

### Manual Operator Deployment (instead of OLM deployment)
```bash
#######################################################
# test and deploy the operator
make deploy IMG=docker.io/$USERNAME/$OPERATOR_NAME:v$VERSION
```

### Manual Remove Operator, CRD and CR
```bash
# cleanup test deployment
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 


**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

