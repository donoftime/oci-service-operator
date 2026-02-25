/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FunctionsApplicationSpec defines the desired state of FunctionsApplication
type FunctionsApplicationSpec struct {
	// The OCID of an existing FunctionsApplication to bind to (optional; if omitted, a new application is created)
	FunctionsApplicationId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the application
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the application
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// SubnetIds is the list of subnet OCIDs in which to run functions in the application
	// +kubebuilder:validation:Required
	SubnetIds []string `json:"subnetIds"`

	// Config is the application configuration passed to functions as environment variables
	Config map[string]string `json:"config,omitempty"`

	// NetworkSecurityGroupIds is the list of NSG OCIDs to add the application to
	NetworkSecurityGroupIds []string `json:"networkSecurityGroupIds,omitempty"`

	// SyslogUrl is the syslog URL to which to send all function logs
	SyslogUrl string `json:"syslogUrl,omitempty"`

	// Shape is the processor shape for functions in the application (GENERIC_X86, GENERIC_ARM, GENERIC_X86_ARM)
	Shape string `json:"shape,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// FunctionsApplicationStatus defines the observed state of FunctionsApplication
type FunctionsApplicationStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the FunctionsApplication",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the FunctionsApplication",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// FunctionsApplication is the Schema for the functionsapplications API
type FunctionsApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionsApplicationSpec   `json:"spec,omitempty"`
	Status FunctionsApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FunctionsApplicationList contains a list of FunctionsApplication
type FunctionsApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FunctionsApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FunctionsApplication{}, &FunctionsApplicationList{})
}
