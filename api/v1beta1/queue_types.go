/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OciQueueSpec defines the desired state of OciQueue
type OciQueueSpec struct {
	// The OCID of an existing Queue to bind to (optional; if omitted, a new queue is created)
	QueueId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the Queue
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the Queue
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// RetentionInSeconds is the retention period of messages in the queue, in seconds
	// +kubebuilder:validation:Minimum:=10
	RetentionInSeconds int `json:"retentionInSeconds,omitempty"`

	// VisibilityInSeconds is the default visibility timeout of messages consumed from the queue, in seconds
	// +kubebuilder:validation:Minimum:=1
	VisibilityInSeconds int `json:"visibilityInSeconds,omitempty"`

	// TimeoutInSeconds is the default polling timeout of messages in the queue, in seconds
	// +kubebuilder:validation:Minimum:=1
	TimeoutInSeconds int `json:"timeoutInSeconds,omitempty"`

	// DeadLetterQueueDeliveryCount is the number of times a message can be delivered before being moved to the DLQ
	// A value of 0 disables the DLQ
	// +kubebuilder:validation:Minimum:=0
	DeadLetterQueueDeliveryCount int `json:"deadLetterQueueDeliveryCount,omitempty"`

	// CustomEncryptionKeyId is the OCID of the custom encryption key for message content (optional)
	CustomEncryptionKeyId OCID `json:"customEncryptionKeyId,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OciQueueStatus defines the observed state of OciQueue
type OciQueueStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciQueue",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciQueue",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciQueue is the Schema for the ociqueues API
type OciQueue struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciQueueSpec   `json:"spec,omitempty"`
	Status OciQueueStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciQueueList contains a list of OciQueue
type OciQueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciQueue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciQueue{}, &OciQueueList{})
}
