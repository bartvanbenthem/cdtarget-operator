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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CDTargetSpec defines the desired state of CDTarget
type CDTargetSpec struct {
	// IP is a slice of string that contains all the CDTarget IPs
	IP []string `json:"ip,omitempty"`
	// specify the pod selector key value pair
	PodSelector map[string]string `json:"podSelector,omitempty"`
}

// CDTargetStatus defines the observed state of CDTarget
type CDTargetStatus struct {
	// Policy contains the name of the CDTarget network policy
	// this verrifies the CDTarget network policy creation
	Policy string `json:"policy,omitempty"`
	// Synced compares the IPs specified in the CR
	// with the IPs in the actual CDTarget network policy
	// when both are equal the Synced status is set to TRUE
	Synced bool `json:"synced,omitempty"`
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
