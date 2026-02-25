/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApiGatewayRouteBackend defines the backend for a route
type ApiGatewayRouteBackend struct {
	// Type is the backend type (HTTP_BACKEND, ORACLE_FUNCTIONS_BACKEND, STOCK_RESPONSE_BACKEND)
	// +kubebuilder:validation:Enum=HTTP_BACKEND;ORACLE_FUNCTIONS_BACKEND;STOCK_RESPONSE_BACKEND
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Url is the backend URL (for HTTP_BACKEND type)
	Url string `json:"url,omitempty"`

	// FunctionId is the OCID of the Oracle Function (for ORACLE_FUNCTIONS_BACKEND type)
	FunctionId string `json:"functionId,omitempty"`

	// Status is the HTTP status code for stock response (for STOCK_RESPONSE_BACKEND type)
	Status int `json:"status,omitempty"`

	// Body is the response body for stock response (for STOCK_RESPONSE_BACKEND type)
	Body string `json:"body,omitempty"`
}

// ApiGatewayRoute defines a single route in a deployment specification
type ApiGatewayRoute struct {
	// Path is the route path (e.g., "/hello")
	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// Methods is the list of HTTP methods (GET, POST, PUT, DELETE, etc.)
	Methods []string `json:"methods,omitempty"`

	// Backend defines where the route sends traffic
	// +kubebuilder:validation:Required
	Backend ApiGatewayRouteBackend `json:"backend"`
}

// ApiGatewayDeploySpec defines the desired state of ApiGatewayDeployment
type ApiGatewayDeploySpec struct {
	// The OCID of an existing Deployment to bind to (optional; if omitted, a new deployment is created)
	DeploymentId OCID `json:"id,omitempty"`

	// GatewayId is the OCID of the API Gateway to deploy to
	// +kubebuilder:validation:Required
	GatewayId OCID `json:"gatewayId"`

	// CompartmentId is the OCID of the compartment in which to create the deployment
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the deployment
	DisplayName string `json:"displayName,omitempty"`

	// PathPrefix is the path prefix for all routes in this deployment
	// +kubebuilder:validation:Required
	PathPrefix string `json:"pathPrefix"`

	// Routes is the list of API routes in this deployment
	Routes []ApiGatewayRoute `json:"routes,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// ApiGatewayDeployStatus defines the observed state of ApiGatewayDeployment
type ApiGatewayDeployStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="PathPrefix",type="string",JSONPath=".spec.pathPrefix",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the ApiGatewayDeployment",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the ApiGatewayDeployment",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// ApiGatewayDeployment is the Schema for the apigatewaydeployments API
type ApiGatewayDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApiGatewayDeploySpec   `json:"spec,omitempty"`
	Status ApiGatewayDeployStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApiGatewayDeploymentList contains a list of ApiGatewayDeployment
type ApiGatewayDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApiGatewayDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApiGatewayDeployment{}, &ApiGatewayDeploymentList{})
}
