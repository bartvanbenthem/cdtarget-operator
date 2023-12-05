/*
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
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	ReasonCRNotAvailable                     = "OperatorResourceNotAvailable"
	ReasonNetworkPolicyNotAvailable          = "OperandNetworkPolicyNotAvailable"
	ReasonOperandNetworkPolicyFailed         = "OperandNetworkPolicyFailed"
	ReasonConfigMapNotAvailable              = "ConfigMapNotAvailable"
	ReasonOperandConfigMapFailed             = "OperandConfigMapFailed"
	ReasonDeploymentNotAvailable             = "DeploymentNotAvailable"
	ReasonOperandDeploymentFailed            = "OperandDeploymentFailed"
	ReasonSecretNotAvailable                 = "SecretNotAvailable"
	ReasonOperandSecretFailed                = "OperandSecretFailed"
	ReasonScaledObjectNotAvailable           = "ScaledObjectNotAvailable"
	ReasonOperandScaledObjectFailed          = "OperandScaledObjectFailed"
	ReasonTriggerAuthenticationNotAvailable  = "TriggerAuthenticationNotAvailable"
	ReasonOperandTriggerAuthenticationFailed = "OperandTriggerAuthenticationFailed"
	ReasonSucceeded                          = "OperatorSucceeded"
)

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
	// set to override the default DNS config of the agent
	DNSConfig corev1.PodDNSConfig `json:"dnsConfig,omitempty"`
	DNSPolicy corev1.DNSPolicy    `json:"dnsPolicy,omitempty"`
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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CDTarget is the Schema for the cdtargets API
type CDTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CDTargetSpec   `json:"spec,omitempty"`
	Status CDTargetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CDTargetList contains a list of CDTarget
type CDTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CDTarget `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CDTarget{}, &CDTargetList{})
}
