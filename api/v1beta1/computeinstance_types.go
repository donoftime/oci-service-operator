/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComputeInstanceShapeConfig defines the OCPU and memory configuration for a flexible compute shape.
type ComputeInstanceShapeConfig struct {
	// Ocpus is the number of OCPUs for the instance shape.
	// +kubebuilder:validation:Required
	Ocpus float32 `json:"ocpus"`

	// MemoryInGBs is the total amount of memory available to the instance in GBs.
	// +kubebuilder:validation:Required
	MemoryInGBs float32 `json:"memoryInGBs"`
}

// ComputeInstanceSpec defines the desired state of ComputeInstance
type ComputeInstanceSpec struct {
	// ComputeInstanceId is the OCID of an existing Compute Instance to bind to (optional).
	ComputeInstanceId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the instance.
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the instance.
	DisplayName *string `json:"displayName,omitempty"`

	// AvailabilityDomain is the availability domain where the instance runs.
	// +kubebuilder:validation:Required
	AvailabilityDomain string `json:"availabilityDomain"`

	// Shape is the OCI shape for the instance (e.g. "VM.Standard.E4.Flex").
	// +kubebuilder:validation:Required
	Shape string `json:"shape"`

	// ShapeConfig specifies the OCPUs and memory for flexible shapes.
	ShapeConfig *ComputeInstanceShapeConfig `json:"shapeConfig,omitempty"`

	// ImageId is the OCID of the image used to boot the instance.
	// +kubebuilder:validation:Required
	ImageId OCID `json:"imageId"`

	// SubnetId is the OCID of the subnet in which to create the instance's primary VNIC.
	// +kubebuilder:validation:Required
	SubnetId OCID `json:"subnetId"`

	TagResources `json:",inline,omitempty"`
}

// ComputeInstanceStatus defines the observed state of ComputeInstance
type ComputeInstanceStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the ComputeInstance",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the ComputeInstance",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// ComputeInstance is the Schema for the computeinstances API
type ComputeInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComputeInstanceSpec   `json:"spec,omitempty"`
	Status ComputeInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ComputeInstanceList contains a list of ComputeInstance
type ComputeInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComputeInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ComputeInstance{}, &ComputeInstanceList{})
}
