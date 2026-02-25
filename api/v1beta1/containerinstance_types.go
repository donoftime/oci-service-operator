/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ContainerSpec defines the configuration of a single container within a ContainerInstance.
type ContainerSpec struct {
	// ImageUrl is the URL of the container image (e.g. "docker.io/library/nginx:latest").
	// +kubebuilder:validation:Required
	ImageUrl string `json:"imageUrl"`

	// DisplayName is a user-friendly name for the container.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Command overrides the container image ENTRYPOINT.
	// +optional
	Command []string `json:"command,omitempty"`

	// Arguments are the ENTRYPOINT arguments.
	// +optional
	Arguments []string `json:"arguments,omitempty"`

	// WorkingDirectory is the container's working directory.
	// +optional
	WorkingDirectory string `json:"workingDirectory,omitempty"`

	// EnvironmentVariables are additional environment variables for the container.
	// +optional
	EnvironmentVariables map[string]string `json:"environmentVariables,omitempty"`
}

// ContainerVnicSpec defines the VNIC configuration for a ContainerInstance.
type ContainerVnicSpec struct {
	// SubnetId is the OCID of the subnet to attach this VNIC to.
	// +kubebuilder:validation:Required
	SubnetId OCID `json:"subnetId"`

	// DisplayName is a user-friendly name for the VNIC.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// IsPublicIpAssigned controls whether a public IP is assigned.
	// +optional
	IsPublicIpAssigned *bool `json:"isPublicIpAssigned,omitempty"`

	// NsgIds is the list of Network Security Group OCIDs to attach to this VNIC.
	// +optional
	NsgIds []string `json:"nsgIds,omitempty"`
}

// ContainerInstanceSpec defines the desired state of ContainerInstance.
type ContainerInstanceSpec struct {
	// ContainerInstanceId is the OCID of an existing container instance to bind to.
	// If omitted, a new container instance is created.
	// +optional
	ContainerInstanceId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the container instance.
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// AvailabilityDomain is the availability domain where the container instance runs.
	// +kubebuilder:validation:Required
	AvailabilityDomain string `json:"availabilityDomain"`

	// Shape is the name of the container instance shape (e.g. "CI.Standard.E4.Flex").
	// +kubebuilder:validation:Required
	Shape string `json:"shape"`

	// Ocpus is the number of OCPUs for the container instance.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	Ocpus float32 `json:"ocpus"`

	// MemoryInGBs is the amount of memory in gigabytes for the container instance.
	// +optional
	MemoryInGBs *float32 `json:"memoryInGBs,omitempty"`

	// Containers is the list of containers to run on this container instance.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Containers []ContainerSpec `json:"containers"`

	// Vnics is the list of VNICs for the container instance.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Vnics []ContainerVnicSpec `json:"vnics"`

	// DisplayName is a user-friendly name for the container instance.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// FaultDomain is the fault domain where the container instance runs.
	// +optional
	FaultDomain string `json:"faultDomain,omitempty"`

	// ContainerRestartPolicy controls the restart policy for containers (ALWAYS, NEVER, ON_FAILURE).
	// +optional
	ContainerRestartPolicy string `json:"containerRestartPolicy,omitempty"`

	// GracefulShutdownTimeoutInSeconds is the time processes have to gracefully shut down.
	// +optional
	GracefulShutdownTimeoutInSeconds *int64 `json:"gracefulShutdownTimeoutInSeconds,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// ContainerInstanceStatus defines the observed state of ContainerInstance.
type ContainerInstanceStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the ContainerInstance",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the ContainerInstance",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// ContainerInstance is the Schema for the containerinstances API
type ContainerInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerInstanceSpec   `json:"spec,omitempty"`
	Status ContainerInstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ContainerInstanceList contains a list of ContainerInstance
type ContainerInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ContainerInstance{}, &ContainerInstanceList{})
}
