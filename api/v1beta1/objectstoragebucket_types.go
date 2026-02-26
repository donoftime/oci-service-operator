/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectStorageBucketSpec defines the desired state of ObjectStorageBucket
type ObjectStorageBucketSpec struct {
	// BucketId is the composite identifier "namespace/bucketName" of an existing bucket to bind to (optional)
	BucketId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the bucket
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// Name is the name of the bucket
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the OCI Object Storage namespace (auto-resolved from tenancy if empty)
	Namespace string `json:"namespace,omitempty"`

	// AccessType controls public access: NoPublicAccess, ObjectRead, ObjectReadWithoutList, ObjectWrite
	AccessType string `json:"accessType,omitempty"`

	// StorageType is the storage tier: Standard or Archive (default: Standard)
	StorageType string `json:"storageType,omitempty"`

	// Versioning controls object versioning: Enabled or Suspended
	Versioning string `json:"versioning,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// ObjectStorageBucketStatus defines the observed state of ObjectStorageBucket
type ObjectStorageBucketStatus struct {
	OsokStatus OSOKStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="BucketName",type="string",JSONPath=".spec.name",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the ObjectStorageBucket",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="namespace/name identifier of the ObjectStorageBucket",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// ObjectStorageBucket is the Schema for the objectstoragebuckets API
type ObjectStorageBucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObjectStorageBucketSpec   `json:"spec,omitempty"`
	Status ObjectStorageBucketStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ObjectStorageBucketList contains a list of ObjectStorageBucket
type ObjectStorageBucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ObjectStorageBucket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ObjectStorageBucket{}, &ObjectStorageBucketList{})
}
