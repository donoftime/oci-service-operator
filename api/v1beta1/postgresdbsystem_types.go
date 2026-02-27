/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresDbSystemSpec defines the desired state of PostgresDbSystem
type PostgresDbSystemSpec struct {
	// The OCID of an existing PostgresDbSystem to bind to (optional; if omitted, a new DB system is created)
	PostgresDbSystemId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the PostgreSQL DB system
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the PostgreSQL DB system
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// DbVersion is the PostgreSQL version (e.g. "14.10")
	// +kubebuilder:validation:Required
	DbVersion string `json:"dbVersion"`

	// Shape is the instance shape for the DB system nodes (e.g. "VM.Standard.E4.Flex")
	// +kubebuilder:validation:Required
	Shape string `json:"shape"`

	// SubnetId is the OCID of the subnet for the DB system
	// +kubebuilder:validation:Required
	SubnetId OCID `json:"subnetId"`

	// StorageType is an optional hint for storage selection; currently the OCI Optimized storage tier is always used
	StorageType string `json:"storageType,omitempty"`

	// Description is an optional user-provided description of the DB system
	Description string `json:"description,omitempty"`

	// InstanceCount is the number of database instance nodes (defaults to 1)
	// +kubebuilder:validation:Minimum:=1
	InstanceCount int `json:"instanceCount,omitempty"`

	// InstanceOcpuCount is the total OCPUs available to each instance node
	InstanceOcpuCount int `json:"instanceOcpuCount,omitempty"`

	// InstanceMemoryInGBs is the total memory available to each instance node, in gigabytes
	InstanceMemoryInGBs int `json:"instanceMemoryInGBs,omitempty"`

	// AdminUsername is the admin username for the PostgreSQL DB system, read from a Kubernetes secret
	AdminUsername UsernameSource `json:"adminUsername,omitempty"`

	// AdminPassword is the admin password for the PostgreSQL DB system, read from a Kubernetes secret
	AdminPassword PasswordSource `json:"adminPassword,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// PostgresDbSystemStatus defines the observed state of PostgresDbSystem
type PostgresDbSystemStatus struct {
	OsokStatus OSOKStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the PostgresDbSystem",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the PostgresDbSystem",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// PostgresDbSystem is the Schema for the postgresdbsystems API
type PostgresDbSystem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresDbSystemSpec   `json:"spec,omitempty"`
	Status PostgresDbSystemStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PostgresDbSystemList contains a list of PostgresDbSystem
type PostgresDbSystemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresDbSystem `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresDbSystem{}, &PostgresDbSystemList{})
}
