/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OciVcnSpec defines the desired state of OciVcn
type OciVcnSpec struct {
	// VcnId is the OCID of an existing VCN to bind to (optional; if omitted, a new VCN is created)
	VcnId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the VCN
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the VCN
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// CidrBlock is the CIDR block for the VCN
	// +kubebuilder:validation:Required
	CidrBlock string `json:"cidrBlock"`

	// DnsLabel is the DNS label for the VCN (optional)
	DnsLabel string `json:"dnsLabel,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OciVcnStatus defines the observed state of OciVcn
type OciVcnStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciVcn",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciVcn",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciVcn is the Schema for the ocivcns API
type OciVcn struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciVcnSpec   `json:"spec,omitempty"`
	Status OciVcnStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciVcnList contains a list of OciVcn
type OciVcnList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciVcn `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciVcn{}, &OciVcnList{})
}

// OciSubnetSpec defines the desired state of OciSubnet
type OciSubnetSpec struct {
	// SubnetId is the OCID of an existing Subnet to bind to (optional; if omitted, a new subnet is created)
	SubnetId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the Subnet
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the Subnet
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// VcnId is the OCID of the VCN that contains this subnet
	// +kubebuilder:validation:Required
	VcnId OCID `json:"vcnId"`

	// CidrBlock is the CIDR block for the subnet
	// +kubebuilder:validation:Required
	CidrBlock string `json:"cidrBlock"`

	// AvailabilityDomain is the availability domain for the subnet (omit for regional subnet)
	AvailabilityDomain string `json:"availabilityDomain,omitempty"`

	// DnsLabel is the DNS label for the subnet (optional)
	DnsLabel string `json:"dnsLabel,omitempty"`

	// ProhibitPublicIpOnVnic controls whether VNICs in this subnet can have public IPs
	ProhibitPublicIpOnVnic bool `json:"prohibitPublicIpOnVnic,omitempty"`

	// RouteTableId is the OCID of the route table the subnet uses (optional)
	RouteTableId OCID `json:"routeTableId,omitempty"`

	// SecurityListIds is the list of security list OCIDs associated with the subnet (optional)
	SecurityListIds []OCID `json:"securityListIds,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OciSubnetStatus defines the observed state of OciSubnet
type OciSubnetStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciSubnet",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciSubnet",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciSubnet is the Schema for the ocisubnets API
type OciSubnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciSubnetSpec   `json:"spec,omitempty"`
	Status OciSubnetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciSubnetList contains a list of OciSubnet
type OciSubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciSubnet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciSubnet{}, &OciSubnetList{})
}
