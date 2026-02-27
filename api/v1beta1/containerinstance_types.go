/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ContainerInstanceShapeConfig defines the OCPU and memory config for the container instance shape.
type ContainerInstanceShapeConfig struct {
	// Ocpus is the number of OCPUs for the shape.
	// +kubebuilder:validation:Required
	Ocpus float32 `json:"ocpus"`

	// MemoryInGBs is the total amount of memory in GBs.
	// +kubebuilder:validation:Required
	MemoryInGBs float32 `json:"memoryInGBs"`
}

// ContainerResourceConfig defines CPU/memory resources reserved for a container.
type ContainerResourceConfig struct {
	// VcpusLimit is the maximum number of vCPUs the container can use.
	VcpusLimit *float32 `json:"vcpusLimit,omitempty"`

	// MemoryLimitInGBs is the maximum amount of memory (in GB) the container can use.
	MemoryLimitInGBs *float32 `json:"memoryLimitInGBs,omitempty"`
}

// ContainerVnicDetails defines the networking configuration for a container instance VNIC.
type ContainerVnicDetails struct {
	// SubnetId is the OCID of the subnet for the VNIC.
	// +kubebuilder:validation:Required
	SubnetId OCID `json:"subnetId"`

	// DisplayName is a user-friendly name for the VNIC.
	DisplayName *string `json:"displayName,omitempty"`

	// NsgIds is a list of NSG OCIDs to associate with this VNIC.
	NsgIds []OCID `json:"nsgIds,omitempty"`
}

// ContainerVolumeMount defines a volume mount for a container.
type ContainerVolumeMount struct {
	// MountPath is the path inside the container where the volume is mounted.
	// +kubebuilder:validation:Required
	MountPath string `json:"mountPath"`

	// VolumeName is the name of the volume to mount.
	// +kubebuilder:validation:Required
	VolumeName string `json:"volumeName"`

	// SubPath is an optional path within the volume to mount.
	SubPath *string `json:"subPath,omitempty"`

	// IsReadOnly mounts the volume as read-only when true.
	IsReadOnly *bool `json:"isReadOnly,omitempty"`
}

// ContainerImagePullSecret holds credentials for pulling images from a private registry.
type ContainerImagePullSecret struct {
	// RegistryEndpoint is the registry hostname (e.g. "registry.example.com").
	// +kubebuilder:validation:Required
	RegistryEndpoint string `json:"registryEndpoint"`

	// Username is the registry username.
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// Password is the registry password.
	// +kubebuilder:validation:Required
	Password string `json:"password"`
}

// ContainerDetails defines a single container in the instance.
type ContainerDetails struct {
	// ImageUrl is the container image URL (e.g. "busybox:latest").
	// +kubebuilder:validation:Required
	ImageUrl string `json:"imageUrl"`

	// DisplayName is a user-friendly name for the container.
	DisplayName *string `json:"displayName,omitempty"`

	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty"`

	// Arguments are command-line arguments for the container entrypoint.
	Arguments []string `json:"arguments,omitempty"`

	// WorkingDirectory sets the working directory inside the container.
	WorkingDirectory *string `json:"workingDirectory,omitempty"`

	// EnvironmentVariables are additional environment variables for the container.
	EnvironmentVariables map[string]string `json:"environmentVariables,omitempty"`

	// ResourceConfig sets per-container resource limits.
	ResourceConfig *ContainerResourceConfig `json:"resourceConfig,omitempty"`

	// VolumeMounts defines volume mounts for this container.
	VolumeMounts []ContainerVolumeMount `json:"volumeMounts,omitempty"`
}

// ContainerInstanceGCPolicy controls how many historical container instances
// OSOK retains for this CR. Instances beyond MaxInstances (oldest first) are
// deleted from OCI when the controller reconciles.
type ContainerInstanceGCPolicy struct {
	// MaxInstances is the maximum number of non-DELETED OCI Container Instances
	// to keep for this CR. Must be >= 1. Defaults to 3.
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	MaxInstances int32 `json:"maxInstances,omitempty"`
}

// ContainerInstanceSpec defines the desired state of ContainerInstance
type ContainerInstanceSpec struct {
	// ContainerInstanceId is the OCID of an existing ContainerInstance to bind to (optional).
	ContainerInstanceId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the container instance.
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// AvailabilityDomain is the availability domain where the container instance runs.
	// +kubebuilder:validation:Required
	AvailabilityDomain string `json:"availabilityDomain"`

	// Shape is the OCI shape for the container instance (e.g. "CI.Standard.E4.Flex").
	// +kubebuilder:validation:Required
	Shape string `json:"shape"`

	// ShapeConfig specifies the OCPUs and memory for the shape.
	// +kubebuilder:validation:Required
	ShapeConfig ContainerInstanceShapeConfig `json:"shapeConfig"`

	// Containers is the list of containers to run in this instance.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Containers []ContainerDetails `json:"containers"`

	// Vnics defines the networking configuration for the container instance.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Vnics []ContainerVnicDetails `json:"vnics"`

	// DisplayName is a user-friendly name for the container instance.
	DisplayName *string `json:"displayName,omitempty"`

	// FaultDomain places the container instance in a specific fault domain.
	FaultDomain *string `json:"faultDomain,omitempty"`

	// GracefulShutdownTimeoutInSeconds is the time in seconds for graceful shutdown.
	GracefulShutdownTimeoutInSeconds *int64 `json:"gracefulShutdownTimeoutInSeconds,omitempty"`

	// ContainerRestartPolicy controls container restart behaviour (ALWAYS, NEVER, ON_FAILURE).
	ContainerRestartPolicy *string `json:"containerRestartPolicy,omitempty"`

	// ImagePullSecrets provides credentials for pulling images from private registries.
	ImagePullSecrets []ContainerImagePullSecret `json:"imagePullSecrets,omitempty"`

	// GCPolicy controls garbage collection of old container instances.
	// Defaults to keeping the 3 most recent non-DELETED instances.
	GCPolicy *ContainerInstanceGCPolicy `json:"gcPolicy,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// ContainerInstanceStatus defines the observed state of ContainerInstance
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
