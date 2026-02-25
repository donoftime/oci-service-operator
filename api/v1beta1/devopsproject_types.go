/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DevopsProjectSpec defines the desired state of DevopsProject
type DevopsProjectSpec struct {
	// The OCID of an existing DevOps project to bind to (optional; if omitted, a new project is created)
	DevopsProjectId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the DevOps project
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// Name is the project name (case-sensitive)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description is a human-readable description for the project
	Description string `json:"description,omitempty"`

	// NotificationTopicId is the OCID of the ONS topic used for project notifications
	// +kubebuilder:validation:Required
	NotificationTopicId OCID `json:"notificationTopicId"`

	TagResources `json:",inline,omitempty"`
}

// DevopsProjectStatus defines the observed state of DevopsProject
type DevopsProjectStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Name",type="string",JSONPath=".spec.name",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the DevopsProject",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the DevopsProject",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// DevopsProject is the Schema for the devopsprojects API
type DevopsProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevopsProjectSpec   `json:"spec,omitempty"`
	Status DevopsProjectStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DevopsProjectList contains a list of DevopsProject
type DevopsProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevopsProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DevopsProject{}, &DevopsProjectList{})
}
