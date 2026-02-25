/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FunctionsFunctionSpec defines the desired state of FunctionsFunction
type FunctionsFunctionSpec struct {
	// The OCID of an existing FunctionsFunction to bind to (optional; if omitted, a new function is created)
	FunctionsFunctionId OCID `json:"id,omitempty"`

	// ApplicationId is the OCID of the application this function belongs to
	// +kubebuilder:validation:Required
	ApplicationId OCID `json:"applicationId"`

	// DisplayName is a user-friendly name for the function
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Image is the qualified name of the Docker image to use in the function
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// MemoryInMBs is the maximum usable memory for the function in MiB
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=128
	MemoryInMBs int64 `json:"memoryInMBs"`

	// TimeoutInSeconds is the timeout for executions of the function in seconds
	TimeoutInSeconds int `json:"timeoutInSeconds,omitempty"`

	// Config is the function configuration, overrides application configuration
	Config map[string]string `json:"config,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// FunctionsFunctionStatus defines the observed state of FunctionsFunction
type FunctionsFunctionStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the FunctionsFunction",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the FunctionsFunction",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// FunctionsFunction is the Schema for the functionsfunctions API
type FunctionsFunction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionsFunctionSpec   `json:"spec,omitempty"`
	Status FunctionsFunctionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FunctionsFunctionList contains a list of FunctionsFunction
type FunctionsFunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FunctionsFunction `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FunctionsFunction{}, &FunctionsFunctionList{})
}
