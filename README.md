# Continues Deployment Target - Operator
Automate the configuration & lifecycle of Azure self-hosted pipelines agents and enable self-service for adding egress targets, without the need of delegating full network policy permissions to the namespace administrator. Event driven autoscaling is automatically enabled trough KEDA and Azure pipelines integrations.

## Operator Design

###  Describing the problem
For us as namespace administrators (cluster users) the CRUD functionality on network policy objects are unauthorized by security design and can only be changed by the cluster administrators. To enable end tot end automation, we need the ability to add target IPs ourselves to a specified set of allowed egress ports through a Custom Resource, the ports are specified by the cluster administrators from centralized configuration. An Operator should automatically create or update a network policy containing the specified IPs defined in the CustomResource. The operator should als configure and manage the lifecycle of the self-hosted pipeline agents, be able to inject proxy configurations and CA certificates trough Kubernetes secrets and simplify the enablement of event driven autoscaling.  

### Designing the API and a CRD

#### K8s Network Policy NetworkPolicyPeer API spec
```text
NetworkPolicyPeer describes a peer to allow traffic to/from. Only certain combinations of fields are allowed

   .  egress.to.ipBlock (IPBlock)

    IPBlock defines policy on a particular IPBlock. If this field is set then neither of the other fields can be.

    IPBlock describes a particular CIDR (Ex. "192.168.1.1/24","2001:db9::/64") that is allowed to the pods matched by a NetworkPolicySpec's podSelector. The except entry describes CIDRs that should not be included within this rule.

        . egress.to.ipBlock.cidr (string), required

        CIDR is a string representing the IP Block Valid examples are "192.168.1.1/24" or "2001:db9::/64"

        . egress.to.ipBlock.except ([]string)

        Except is a slice of CIDRs that should not be included within an IP Block Valid examples are "192.168.1.1/24" or "2001:db9::/64" Except values will be rejected if they are outside the CIDR range
```
#### IPBLock type
```Go
type IPBlock struct {
	// CIDR is a string representing the IP Block
	// Valid examples are "192.168.1.1/24" or "2001:db9::/64"
	CIDR string `json:"cidr" protobuf:"bytes,1,name=cidr"`
	// Except is a slice of CIDRs that should not be included within an IP Block
	// Valid examples are "192.168.1.1/24" or "2001:db9::/64"
	// Except values will be rejected if they are outside the CIDR range
	// +optional
	Except []string `json:"except,omitempty" protobuf:"bytes,2,rep,name=except"`
}
```

#### CDTarget types
```Go
// CDTargetSpec defines the desired state of CDTarget
type CDTargetSpec struct {
	// IP is a slice of string that contains all the CDTarget IPs
	IP []string `json:"ip,omitempty"`
	// specify the pod selector key value pair
	AdditionalSelector map[string]string `json:"additionalSelector"`
	// pipeline agent image
	AgentImage string `json:"agentImage,omitempty"`
	// +optional
	AgentResources corev1.ResourceRequirements `json:"agentResources,omitempty"`
	// image pull secrets
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// +optional
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// Inject additional environment variables to the deployment
	Env []corev1.EnvVar `json:"env,omitempty"`
	// reference to secret that contains the the Proxy settings
	ProxyRef string `json:"proxyRef,omitempty"`
	// reference to secret that contains the PAT
	TokenRef string `json:"tokenRef"`
	// reference to secret that contains the CA certificates
	CACertRef string `json:"caCertRef,omitempty"`
	// AzureDevPortal is configuring the Azure DevOps pool settings of the Agent
	// by using additional environment variables.
	Config AgentConfig `json:"config,omitempty"`
	// set to add or override the default metadata for the
	// scaled object trigger metadata
	TriggerMeta map[string]string `json:"triggerMeta,omitempty"`
}

// CDTargetStatus defines the observed state of CDTarget
type CDTargetStatus struct {
	// Conditions lists the most recent status condition updates
	Conditions []metav1.Condition `json:"conditions"`
}

// control the pool and agent work directory
type AgentConfig struct {
	URL       string `json:"url"`
	PoolName  string `json:"poolName"`
	AgentName string `json:"agentName,omitempty"`
	WorkDir   string `json:"workDir,omitempty"`
	// Allow specifying MTU value for networks used by container jobs
	// useful for docker-in-docker scenarios in k8s cluster
	MTUValue string `json:"mtuValue,omitempty"`
}
```

#### Custom Resource schema
```yaml
apiVersion: cnad.gofound.nl/v1alpha1
kind: CDTarget
metadata:
  name: <<cdtarget-sample>>
  namespace: <<test>>
spec:
  agentImage: ghcr.io/bartvanbenthem/azagent-keda-22:latest
  imagePullSecrets:
  - name: <<cdtarget-regcred>>
  minReplicaCount: 1
  maxReplicaCount: 3
  agentResources:
    requests:
      cpu: 100m
    limits:
      cpu: 200m
  config:
    url: <<https://dev.azure.com/ORGANIZATION>>
    poolName: <<pool-name>>
    workDir: 
    mtuValue:
    agentName:
  tokenRef: <<cdtarget-token>>
  proxyRef: <<cdtarget-proxy>>
  caCertRef: <<cdtarget-ca>>
  dnsPolicy: <<None>>
  dnsConfig:
    nameservers: 
    - <<8.8.8.8>>
  triggerMeta:
    demands: "maven,docker"
  env:
  - name: INJECTED_ADDITIONAL_ENV
    value: example-env
  additionalSelector:
    app: cdtarget-agent 
  ip:
  - <<10.0.0.1>>
  - <<10.0.0.2>>
    ...
```

### Required Resources & Permissions
What other resources are required:
```Go
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operators.coreos.com,resources=operatorconditions,verbs=get;list;watch
//+kubebuilder:rbac:groups=keda.sh,resources=scaledobjects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keda.sh,resources=triggerauthentications,verbs=get;list;watch;create;update;patch;delete
```

### Target reconciliation loop design
```Go
func Reconcile:
// Get the Operator's CRD, if it doesn't exist then return
// an error so the user knows to create it:
operatorCrd, error = getMyCRD()
    if error != nil {
    return error
}
// Get the related resources for the Operator (networkpolicy)
// If they don't exist, create them:
resources, error = getRelatedResources()
if error == ResourcesNotFound {
    createRelatedResources()
}
// Check that the related resources relevant values match
// what is set in the Operator's CRD. If they don't match,
// update the resource with the specified values:
if resources.Spec != operatorCrd.Spec {
    updateRelatedResources(operatorCrd.Spec)
}
```

### Handling upgrades and downgrades
todo

### Failure reporting
Logs, Events + status updates

``` yaml
status:
  conditions:
  - lastTransitionTime: 2022-01-01T00:00:00Z
    message: reconciling message
    reason: event
    status: "False"/"True"
    type: ReconcileSuccess
```

# Pereqs

## Install KEDA
```bash
# Deploying using the deployment YAML files
kubectl apply --server-side -f \
  https://github.com/kedacore/keda/releases/download/v2.12.0/keda-2.12.0.yaml
```

# Scaffolding parameters
```bash
operator-sdk init --domain gofound.nl --repo github.com/bartvanbenthem/cdtarget-operator
operator-sdk create api --group cnad --version v1alpha1 --kind CDTarget --resource --controller
```

```bash
# always run make after changing *_types.go and *_controller.go
go mod tidy
make generate
make manifests
```

# Build Operator image
```bash
# docker and github repo username
export USERNAME='bartvanbenthem'
# image and bundle version
export VERSION=1.8.0
# operator repo and name
export OPERATOR_NAME='cdtarget-operator'

#######################################################
source ../00-ENV/env.sh # personal setup to inject PAT
# login to ghcr.io registry
echo $CR_PAT | docker login ghcr.io -u USERNAME --password-stdin
# Build the operator image
make docker-build docker-push IMG=ghcr.io/$USERNAME/$OPERATOR_NAME:v$VERSION
```

### Manual Operator Deployment
```bash
#######################################################
# test and deploy the operator
make deploy IMG=ghcr.io/$USERNAME/$OPERATOR_NAME:v$VERSION
```

### Test custom resource
```bash
#######################################################
# test cdtarget CR 
kubectl create ns test
# prestage the PAT (token) Secret for succesfull Azure AUTH
kubectl -n test create secret generic cdtarget-token --from-literal=AZP_TOKEN=$PAT
# apply cdtarget resource
# for scaling >1 replica don`t set the agentName field in the CR
kubectl -n test apply -f config/samples/cnad_cdtarget_sample.yaml
kubectl -n test describe cdtarget cdtarget-agent
# test CDTarget created objects
kubectl -n test describe secret cdtarget-token
kubectl -n test get configmaps
kubectl -n test get networkpolicies
kubectl -n test describe networkpolicies cdtarget-agent
kubectl -n test describe scaledobject cdtarget-agent-keda
kubectl -n test describe deployment cdtarget-agent

```

### Create pull secret
```bash
# create regcred secret
kubectl -n test create secret docker-registry cdtarget-regcred \
          --docker-server='https://ghcr.io' \
          --docker-username='bartvanbenthem' \
          --docker-password=$CR_PAT \
```
### Create & Update Proxy config
```bash
# update secret containing proxy settings
kubectl -n test create secret generic cdtarget-proxy --dry-run=client -o yaml \
                  --from-literal=PROXY_USER='' \
                  --from-literal=PROXY_PW='' \
                  --from-literal=PROXY_URL='' \
                  --from-literal=HTTP_PROXY='' \
                  --from-literal=HTTPS_PROXY='' \
                  --from-literal=FTP_PROXY='' \
                  --from-literal=NO_PROXY='10.0.0.0/8' | kubectl apply -f -
kubectl -n test scale deployment cdtarget-agent-keda --replicas=0  
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
EOF

kubectl -n test delete networkpolicies.networking.k8s.io cdtarget-agent-keda
```

### Update Personal Access Token
```bash
# update CDTarget PAT
kubectl -n test create secret generic cdtarget-token --dry-run=client -o yaml \
                  --from-literal=AZP_TOKEN=$PAT | kubectl apply -f -
kubectl -n test scale deployment cdtarget-agent-keda --replicas=0  
```

### Inject CA Certificates from file
* Best practise is to have the ca certificate prestaged as a kubernetes secret 
* from the custom resource a reference is made to the prestaged secret
```bash
# inject CA Certificates to CDTarget agents
# in /usr/local/share/ca-certificates/
# trust store: /etc/ssl/certs/ca-certificates.crt
kubectl -n test create secret generic cdtarget-ca --dry-run=client -o yaml \
                --from-file="config/samples/CERTIFICATE.crt" | kubectl apply -f -
kubectl -n test scale deployment cdtarget-agent-keda --replicas=0  
```

### Manual Remove Operator, CRD and CR
```bash
# cleanup test deployment
kubectl -n test delete -f config/samples/cnad_cdtarget_sample.yaml
kubectl delete ns test
# cleanup test deployment
make undeploy
```

## Operator lifecycle manager 
(instead of manual deployment)

### Operator lifecycle manager Installation
```bash
#######################################################
# install OLM (if not already present)
operator-sdk olm install
operator-sdk olm status
```

### Operator lifecycle manager Deployment
```bash
#######################################################
# Build the OLM bundle
make bundle IMG=ghcr.io/$USERNAME/$OPERATOR_NAME:v$VERSION   
make bundle-build bundle-push BUNDLE_IMG=ghcr.io/$USERNAME/$OPERATOR_NAME-bundle:v$VERSION
```

```bash
# Deploy OLM bundle
kubectl create ns 'cdtarget-operator'
operator-sdk run bundle ghcr.io/$USERNAME/$OPERATOR_NAME-bundle:v$VERSION --namespace='cdtarget-operator'
```

### Remove CR, CRD & Operator Bundle
```bash
# cleanup test deployment
kubectl -n test delete -f config/samples/cnad_cdtarget_sample.yaml
kubectl delete ns test
# cleanup OLM bundle & OLM installation
operator-sdk cleanup operator --delete-all --namespace='cdtarget-operator'
kubectl delete ns 'cdtarget-operator'
```

### Uninstall Operator Lifecycle Manager
```bash
# uninstall OLM
operator-sdk olm uninstall
```

