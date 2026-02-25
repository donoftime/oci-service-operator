/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NoSQLDatabaseTableLimits defines throughput and storage limits for a NoSQL table.
type NoSQLDatabaseTableLimits struct {
	// MaxReadUnits is the maximum sustained read throughput limit for the table.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	MaxReadUnits int `json:"maxReadUnits"`

	// MaxWriteUnits is the maximum sustained write throughput limit for the table.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	MaxWriteUnits int `json:"maxWriteUnits"`

	// MaxStorageInGBs is the maximum size of storage used by the table, in gigabytes.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	MaxStorageInGBs int `json:"maxStorageInGBs"`
}

// NoSQLDatabaseSpec defines the desired state of NoSQLDatabase
type NoSQLDatabaseSpec struct {
	// TableId is the OCID of an existing NoSQL table to bind to (optional; if omitted, a new table is created)
	TableId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the NoSQL table
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// Name is the name of the NoSQL table (human-friendly, immutable after creation)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// DdlStatement is the complete CREATE TABLE DDL statement for this table
	// +kubebuilder:validation:Required
	DdlStatement string `json:"ddlStatement"`

	// TableLimits defines throughput and storage limits for the table (required for provisioned capacity)
	// +kubebuilder:validation:Optional
	TableLimits *NoSQLDatabaseTableLimits `json:"tableLimits,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// NoSQLDatabaseStatus defines the observed state of NoSQLDatabase
type NoSQLDatabaseStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Name",type="string",JSONPath=".spec.name",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the NoSQLDatabase",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the NoSQLDatabase",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// NoSQLDatabase is the Schema for the nosqldatabases API
type NoSQLDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NoSQLDatabaseSpec   `json:"spec,omitempty"`
	Status NoSQLDatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NoSQLDatabaseList contains a list of NoSQLDatabase
type NoSQLDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NoSQLDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NoSQLDatabase{}, &NoSQLDatabaseList{})
}
