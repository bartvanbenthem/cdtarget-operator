# Continues Deployment Target - Operator
Allow teams to add egress targets trough self-service, without delegating full network policy permissions within the team namespace.

## Operator Design

### Determine the core aspects
* Problem description
* Designing the API and CRD
* Required resources
* Target reconciliation loop design
* Upgrade and downgrade strategy
* Failure reporting

###  Describing the problem
For me as namespace administrator (cluster user) the CRUD functionality on network policy objects are unauthorized and can only be changed by the cluster administrators. We need to be able to add target IP blocks ourselves to a specified set (specified by the admins) of allowed egress ports trough a Custom Resource. An Operator should automatically update the specific network policy with all the IP blocks defined in the Custom resource.

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
type CDTargetSpec struct {
	// IP is a slice of string that contains all the CDTarget IPs
  // each IP is added as an /32 IPBlock CIDR value of the network policy 
  IP []string `json:"ip,omitempty"`
  // specify the pod selector key value pair
	PodSelector map[string]string `json:"podSelector,omitempty"`
}

type CDTargetStatus struct {
	// Conditions lists the most recent status condition updates
	Conditions []metav1.Condition `json:"conditions"`
}
```

#### Custom Resource schema
```yaml
apiVersion: v1alpha1
kind: CDTarget
metadata:
  name: example
  namespace: example
spec:
  podSelector:
    key: value 
  ip:
  - 10.0.0.1
  - 10.0.0.2
    ...
```

#### Custom Resource Definition schema
```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: cdtargets.cnad.gofound.nl
spec:
  group: cnad.gofound.nl
  names:
    kind: CDTarget
    listKind: CDTargetList
    plural: cdtargets
    singular: cdtarget
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: ''
        properties:
          apiVersion:
            description: ''
            type: string
          kind:
            description: ''
            type: string
          metadata:
            type: object
          spec:
            description: ''
            properties:
              ip:
                description: ''
                items:
                  type: string
                type: array
            type: object
          status:
            description: ''
            properties:
              policy:
                description: ''
              synced:
                description: ''
                type: boolean
            type: object
        type: object
    served: true
    storage: true
```

### Required Resources
What other resources are required:
```Go
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
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
    - type: Ready
      status: "True"
      lastProbeTime: null
      lastTransitionTime: 2022-01-01T00:00:00Z
```

## Scaffolding parameters
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

# Build & Deploy
```bash
# docker and github repo username
export USERNAME='bartvanbenthem'
# image and bundle version
export VERSION=0.1.9
# operator repo and name
export OPERATOR_NAME='cdtarget-operator'

#######################################################
# Build the operator
make docker-build docker-push IMG=docker.io/$USERNAME/$OPERATOR_NAME:v$VERSION

#######################################################
# test and deploy the operator
make deploy IMG=docker.io/$USERNAME/$OPERATOR_NAME:v$VERSION
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
kubectl -n test apply -f ../cnad_cdtarget_sample.yaml
kubectl -n test describe cdtarget cdtarget-sample
kubectl -n test describe networkpolicies cdtarget-sample
```

## Remove Operator, CRD and CR
```bash
# cleanup test deployment
kubectl -n test delete -f ../cnad_cdtarget_sample.yaml
kubectl delete ns test
make undeploy
```