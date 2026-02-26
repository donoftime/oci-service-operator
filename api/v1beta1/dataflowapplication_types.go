/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataFlowApplicationSpec defines the desired state of DataFlowApplication
type DataFlowApplicationSpec struct {
	// DataFlowApplicationId is the OCID of an existing Data Flow Application to bind to (optional)
	DataFlowApplicationId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the application
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the Data Flow Application
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Language is the Spark language for the application (PYTHON, SCALA, JAVA, SQL)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=PYTHON;SCALA;JAVA;SQL
	Language string `json:"language"`

	// DriverShape is the VM shape for the driver
	// +kubebuilder:validation:Required
	DriverShape string `json:"driverShape"`

	// ExecutorShape is the VM shape for the executors
	// +kubebuilder:validation:Required
	ExecutorShape string `json:"executorShape"`

	// NumExecutors is the number of executor VMs requested
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	NumExecutors int `json:"numExecutors"`

	// SparkVersion is the Spark version to use (e.g. "3.2.1")
	// +kubebuilder:validation:Required
	SparkVersion string `json:"sparkVersion"`

	// FileUri is the OCI URI for the application file (not required for SQL)
	FileUri string `json:"fileUri,omitempty"`

	// ClassName is the Java/Scala main class name
	ClassName string `json:"className,omitempty"`

	// Arguments are command-line arguments passed to the application
	Arguments []string `json:"arguments,omitempty"`

	// Configuration is the Spark configuration key-value pairs
	Configuration map[string]string `json:"configuration,omitempty"`

	// Description is a user-friendly description of the application
	Description string `json:"description,omitempty"`

	// LogsBucketUri is the OCI URI for the logs bucket
	LogsBucketUri string `json:"logsBucketUri,omitempty"`

	// WarehouseBucketUri is the OCI URI for the Hive warehouse bucket
	WarehouseBucketUri string `json:"warehouseBucketUri,omitempty"`

	// ArchiveUri is the OCI URI for an archive file with custom dependencies
	ArchiveUri string `json:"archiveUri,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// DataFlowApplicationStatus defines the observed state of DataFlowApplication
type DataFlowApplicationStatus struct {
	OsokStatus OSOKStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Language",type="string",JSONPath=".spec.language",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the DataFlowApplication",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the DataFlowApplication",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// DataFlowApplication is the Schema for the dataflowapplications API
type DataFlowApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataFlowApplicationSpec   `json:"spec,omitempty"`
	Status DataFlowApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataFlowApplicationList contains a list of DataFlowApplication
type DataFlowApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataFlowApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataFlowApplication{}, &DataFlowApplicationList{})
}
