/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenSearchClusterSpec defines the desired state of OpenSearchCluster
type OpenSearchClusterSpec struct {
	// The OCID of an existing OpenSearch cluster to bind (optional; if set, cluster is adopted rather than created)
	OpenSearchClusterId OCID `json:"id,omitempty"`
	// The OCID of the compartment in which to create the cluster
	CompartmentId OCID `json:"compartmentId,omitempty"`
	// The display name of the cluster
	DisplayName string `json:"displayName,omitempty"`
	// The version of OpenSearch software the cluster will run
	SoftwareVersion string `json:"softwareVersion,omitempty"`

	// Master node configuration
	// +kubebuilder:validation:Minimum:=1
	MasterNodeCount int `json:"masterNodeCount,omitempty"`
	// +kubebuilder:validation:Enum:=FLEX;BM
	MasterNodeHostType string `json:"masterNodeHostType,omitempty"`
	// +kubebuilder:validation:Minimum:=1
	MasterNodeHostOcpuCount int `json:"masterNodeHostOcpuCount,omitempty"`
	// +kubebuilder:validation:Minimum:=1
	MasterNodeHostMemoryGB      int    `json:"masterNodeHostMemoryGB,omitempty"`
	MasterNodeHostBareMetalShape string `json:"masterNodeHostBareMetalShape,omitempty"`

	// Data node configuration
	// +kubebuilder:validation:Minimum:=1
	DataNodeCount int `json:"dataNodeCount,omitempty"`
	// +kubebuilder:validation:Enum:=FLEX;BM;DENSE_IO
	DataNodeHostType string `json:"dataNodeHostType,omitempty"`
	// +kubebuilder:validation:Minimum:=1
	DataNodeHostOcpuCount int `json:"dataNodeHostOcpuCount,omitempty"`
	// +kubebuilder:validation:Minimum:=1
	DataNodeHostMemoryGB int `json:"dataNodeHostMemoryGB,omitempty"`
	// +kubebuilder:validation:Minimum:=1
	DataNodeStorageGB        int    `json:"dataNodeStorageGB,omitempty"`
	DataNodeHostBareMetalShape string `json:"dataNodeHostBareMetalShape,omitempty"`

	// OpenSearch Dashboard node configuration
	// +kubebuilder:validation:Minimum:=1
	OpendashboardNodeCount int `json:"opendashboardNodeCount,omitempty"`
	// +kubebuilder:validation:Minimum:=1
	OpendashboardNodeHostOcpuCount int `json:"opendashboardNodeHostOcpuCount,omitempty"`
	// +kubebuilder:validation:Minimum:=1
	OpendashboardNodeHostMemoryGB int `json:"opendashboardNodeHostMemoryGB,omitempty"`

	// Networking
	VcnId               OCID `json:"vcnId,omitempty"`
	SubnetId            OCID `json:"subnetId,omitempty"`
	VcnCompartmentId    OCID `json:"vcnCompartmentId,omitempty"`
	SubnetCompartmentId OCID `json:"subnetCompartmentId,omitempty"`

	// Security
	// +kubebuilder:validation:Enum:=DISABLED;PERMISSIVE;ENFORCING
	SecurityMode                   string `json:"securityMode,omitempty"`
	SecurityMasterUserName         string `json:"securityMasterUserName,omitempty"`
	SecurityMasterUserPasswordHash string `json:"securityMasterUserPasswordHash,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OpenSearchClusterStatus defines the observed state of OpenSearchCluster
type OpenSearchClusterStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OpenSearchCluster",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OpenSearchCluster",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OpenSearchCluster is the Schema for the opensearchclusters API
type OpenSearchCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenSearchClusterSpec   `json:"spec,omitempty"`
	Status OpenSearchClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenSearchClusterList contains a list of OpenSearchCluster
type OpenSearchClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenSearchCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenSearchCluster{}, &OpenSearchClusterList{})
}
