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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=SCM;URL
type OriginType string

const (
	OriginTypeSCM OriginType = "SCM"
	OriginTypeURL OriginType = "URL"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PackageBundleSpec defines the desired state of PackageBundle
type PackageBundleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// +kubebuilder:validation:Required
	// Container image name.

	// +kubebuilder:validation:Required
	// Name of the NSO instance where the packages are going to be loaded.
	TargetName string `json:"targetName"`

	// +kubebuilder:validation:Optional
	// Size of the folder that will store the packages
	StorageSize string `json:"storageSize,omitempty"`

	// +kubebuilder:validation:Required
	// Origin of the packages. Must be either "scm" or "url".
	Origin OriginType `json:"origin"`

	// +kubebuilder:validation:Optional
	// If set to true, self-signed certificates will be accepted for downloading or pulling.
	InsecureTLS bool `json:"insecureTLS,omitempty"`

	// +kubebuilder:validation:Optional
	// Credentials used for WEB or SSH connections.
	Credentials AccessCredentials `json:"credentials,omitempty"`

	// +kubebuilder:validation:Required
	// Credentials used for WEB or SSH connections.
	Source PackageSource `json:"source"`
}

type AccessCredentials struct {
	// +kubebuilder:validation:Optional
	// SSH key Secret name.
	SshKeySecretRef string `json:"sshKeySecretRef,omitempty"`

	// +kubebuilder:validation:Optional
	// HTTP authentication Secret name.
	HttpAuthSecretRef string `json:"httpAuthSecretRef,omitempty"`
}

type PackageSource struct {
	// +kubebuilder:validation:Required
	// URL of the repository or WEB where the packages are.
	Url string `json:"url"`

	// +kubebuilder:validation:Optional
	// URL of the repository or WEB where the packages are.
	Branch string `json:"branch,omitempty"`

	// +kubebuilder:validation:Optional
	// Path in the repository where the packages are.
	Path string `json:"path,omitempty"`
}

// PackageBundleStatus defines the observed state of PackageBundle.
type PackageBundleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PackageBundle is the Schema for the packagebundles API
type PackageBundle struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of PackageBundle
	// +required
	Spec PackageBundleSpec `json:"spec"`

	// status defines the observed state of PackageBundle
	// +optional
	Status PackageBundleStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// PackageBundleList contains a list of PackageBundle
type PackageBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PackageBundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PackageBundle{}, &PackageBundleList{})
}
