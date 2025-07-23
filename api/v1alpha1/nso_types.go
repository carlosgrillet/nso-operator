/*
Copyright 2025.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NSOSpec defines the desired state of NSO.
type NSOSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// Container image name.
	Image string `json:"image"`

	// +kubebuilder:validation:Required
	// Name of the headless service for NSO.
	ServiceName string `json:"serviceName"`

	// +kubebuilder:validation:Required
	// Number of NSO replicas desired.
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Required
	// Labels for NSO resource.
	LabelSelector map[string]string `json:"labelSelector"`

	// +kubebuilder:validation:Required
	// Service ports.
	Ports []corev1.ServicePort `json:"ports"`

	// +kubebuilder:validation:Required
	// NSO configuration ConfigMap name.
	NsoConfigRef string `json:"nsoConfigRef"`

	// +kubebuilder:validation:Required
	// NSO admin credentials.
	AdminCredentials Credentials `json:"adminCredentials"`

	// +kubebuilder:validation:Required
	// NSO admin credentials.
	Env []corev1.EnvVar `json:"env"`
}

// Credentials for admin user.
type Credentials struct {
	// +kubebuilder:validation:Required
	// NSO admin username.
	Username string `json:"username"`

	// +kubebuilder:validation:Required
	// NSO admin password Secret name.
	PasswordSecretRef string `json:"passwordSecretRef"`
}

// NSOStatus defines the observed state of NSO.
type NSOStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NSO is the Schema for the nsoes API.
type NSO struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NSOSpec   `json:"spec,omitempty"`
	Status NSOStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NSOList contains a list of NSO.
type NSOList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NSO `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NSO{}, &NSOList{})
}
