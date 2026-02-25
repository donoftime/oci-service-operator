/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisClusterSpec defines the desired state of RedisCluster
type RedisClusterSpec struct {
	// RedisClusterId is the OCID of an existing Redis cluster to bind.
	// If empty, a new cluster will be created.
	RedisClusterId OCID `json:"Id,omitempty"`

	// CompartmentId is the OCID of the compartment that contains the Redis cluster.
	// +kubebuilder:validation:MaxLength=255
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the Redis cluster.
	DisplayName string `json:"displayName"`

	// NodeCount is the number of nodes in the Redis cluster.
	// +kubebuilder:validation:Minimum=1
	NodeCount int `json:"nodeCount"`

	// NodeMemoryInGBs is the amount of memory allocated to the Redis cluster's nodes, in gigabytes.
	// +kubebuilder:validation:Minimum=1
	NodeMemoryInGBs float32 `json:"nodeMemoryInGBs"`

	// SoftwareVersion is the Redis version that the cluster is running.
	// +kubebuilder:validation:Enum=V7_0_5
	SoftwareVersion string `json:"softwareVersion"`

	// SubnetId is the OCID of the Redis cluster's subnet.
	// +kubebuilder:validation:MaxLength=255
	SubnetId OCID `json:"subnetId"`

	TagResources `json:",inline"`
}

// RedisClusterStatus defines the observed state of RedisCluster
type RedisClusterStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the RedisCluster",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the RedisCluster",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// RedisCluster is the Schema for the redisclusters API
type RedisCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisClusterSpec   `json:"spec,omitempty"`
	Status RedisClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RedisClusterList contains a list of RedisCluster
type RedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisCluster{}, &RedisClusterList{})
}
