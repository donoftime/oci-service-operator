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

// OciInternetGatewaySpec defines the desired state of OciInternetGateway
type OciInternetGatewaySpec struct {
	// InternetGatewayId is the OCID of an existing Internet Gateway to bind to (optional)
	InternetGatewayId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the Internet Gateway
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// VcnId is the OCID of the VCN that contains this Internet Gateway
	// +kubebuilder:validation:Required
	VcnId OCID `json:"vcnId"`

	// DisplayName is a user-friendly name for the Internet Gateway
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// IsEnabled controls whether the Internet Gateway is enabled (default true)
	IsEnabled bool `json:"isEnabled,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OciInternetGatewayStatus defines the observed state of OciInternetGateway
type OciInternetGatewayStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciInternetGateway",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciInternetGateway",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciInternetGateway is the Schema for the ociinternetgateways API
type OciInternetGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciInternetGatewaySpec   `json:"spec,omitempty"`
	Status OciInternetGatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciInternetGatewayList contains a list of OciInternetGateway
type OciInternetGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciInternetGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciInternetGateway{}, &OciInternetGatewayList{})
}

// OciNatGatewaySpec defines the desired state of OciNatGateway
type OciNatGatewaySpec struct {
	// NatGatewayId is the OCID of an existing NAT Gateway to bind to (optional)
	NatGatewayId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the NAT Gateway
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// VcnId is the OCID of the VCN that contains this NAT Gateway
	// +kubebuilder:validation:Required
	VcnId OCID `json:"vcnId"`

	// DisplayName is a user-friendly name for the NAT Gateway
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// BlockTraffic controls whether the NAT Gateway blocks traffic (default false)
	BlockTraffic bool `json:"blockTraffic,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OciNatGatewayStatus defines the observed state of OciNatGateway
type OciNatGatewayStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciNatGateway",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciNatGateway",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciNatGateway is the Schema for the ocinatgateways API
type OciNatGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciNatGatewaySpec   `json:"spec,omitempty"`
	Status OciNatGatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciNatGatewayList contains a list of OciNatGateway
type OciNatGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciNatGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciNatGateway{}, &OciNatGatewayList{})
}

// OciServiceGatewaySpec defines the desired state of OciServiceGateway
type OciServiceGatewaySpec struct {
	// ServiceGatewayId is the OCID of an existing Service Gateway to bind to (optional)
	ServiceGatewayId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the Service Gateway
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// VcnId is the OCID of the VCN that contains this Service Gateway
	// +kubebuilder:validation:Required
	VcnId OCID `json:"vcnId"`

	// DisplayName is a user-friendly name for the Service Gateway
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Services is the list of OCI service OCIDs to enable on this gateway
	// +kubebuilder:validation:Required
	Services []string `json:"services"`

	TagResources `json:",inline,omitempty"`
}

// OciServiceGatewayStatus defines the observed state of OciServiceGateway
type OciServiceGatewayStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciServiceGateway",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciServiceGateway",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciServiceGateway is the Schema for the ociservicegateways API
type OciServiceGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciServiceGatewaySpec   `json:"spec,omitempty"`
	Status OciServiceGatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciServiceGatewayList contains a list of OciServiceGateway
type OciServiceGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciServiceGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciServiceGateway{}, &OciServiceGatewayList{})
}

// OciDrgSpec defines the desired state of OciDrg
type OciDrgSpec struct {
	// DrgId is the OCID of an existing DRG to bind to (optional)
	DrgId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the DRG
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the DRG
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	TagResources `json:",inline,omitempty"`
}

// OciDrgStatus defines the observed state of OciDrg
type OciDrgStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciDrg",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciDrg",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciDrg is the Schema for the ocidrgs API
type OciDrg struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciDrgSpec   `json:"spec,omitempty"`
	Status OciDrgStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciDrgList contains a list of OciDrg
type OciDrgList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciDrg `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciDrg{}, &OciDrgList{})
}

// IngressSecurityRule defines an ingress rule for a security list
type IngressSecurityRule struct {
	Protocol    string      `json:"protocol"`
	Source      string      `json:"source"`
	IsStateless bool        `json:"isStateless,omitempty"`
	Description string      `json:"description,omitempty"`
	TcpOptions  *TcpOptions `json:"tcpOptions,omitempty"`
	UdpOptions  *UdpOptions `json:"udpOptions,omitempty"`
}

// EgressSecurityRule defines an egress rule
type EgressSecurityRule struct {
	Protocol    string      `json:"protocol"`
	Destination string      `json:"destination"`
	IsStateless bool        `json:"isStateless,omitempty"`
	Description string      `json:"description,omitempty"`
	TcpOptions  *TcpOptions `json:"tcpOptions,omitempty"`
	UdpOptions  *UdpOptions `json:"udpOptions,omitempty"`
}

// PortRange defines min/max port
type PortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// TcpOptions for TCP rules
type TcpOptions struct {
	DestinationPortRange *PortRange `json:"destinationPortRange,omitempty"`
	SourcePortRange      *PortRange `json:"sourcePortRange,omitempty"`
}

// UdpOptions for UDP rules
type UdpOptions struct {
	DestinationPortRange *PortRange `json:"destinationPortRange,omitempty"`
	SourcePortRange      *PortRange `json:"sourcePortRange,omitempty"`
}

// OciSecurityListSpec defines the desired state of OciSecurityList
type OciSecurityListSpec struct {
	// SecurityListId is the OCID of an existing Security List to bind to (optional)
	SecurityListId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// VcnId is the OCID of the VCN that contains this Security List
	// +kubebuilder:validation:Required
	VcnId OCID `json:"vcnId"`

	// DisplayName is a user-friendly name for the Security List
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// IngressSecurityRules are the ingress rules
	IngressSecurityRules []IngressSecurityRule `json:"ingressSecurityRules,omitempty"`

	// EgressSecurityRules are the egress rules
	EgressSecurityRules []EgressSecurityRule `json:"egressSecurityRules,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OciSecurityListStatus defines the observed state of OciSecurityList
type OciSecurityListStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciSecurityList",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciSecurityList",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciSecurityList is the Schema for the ocisecuritylists API
type OciSecurityList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciSecurityListSpec   `json:"spec,omitempty"`
	Status OciSecurityListStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciSecurityListList contains a list of OciSecurityList
type OciSecurityListList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciSecurityList `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciSecurityList{}, &OciSecurityListList{})
}

// OciNetworkSecurityGroupSpec defines the desired state of OciNetworkSecurityGroup
type OciNetworkSecurityGroupSpec struct {
	// NetworkSecurityGroupId is the OCID of an existing NSG to bind to (optional)
	NetworkSecurityGroupId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// VcnId is the OCID of the VCN that contains this NSG
	// +kubebuilder:validation:Required
	VcnId OCID `json:"vcnId"`

	// DisplayName is a user-friendly name for the NSG
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	TagResources `json:",inline,omitempty"`
}

// OciNetworkSecurityGroupStatus defines the observed state of OciNetworkSecurityGroup
type OciNetworkSecurityGroupStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciNetworkSecurityGroup",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciNetworkSecurityGroup",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciNetworkSecurityGroup is the Schema for the ocinetworksecuritygroups API
type OciNetworkSecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciNetworkSecurityGroupSpec   `json:"spec,omitempty"`
	Status OciNetworkSecurityGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciNetworkSecurityGroupList contains a list of OciNetworkSecurityGroup
type OciNetworkSecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciNetworkSecurityGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciNetworkSecurityGroup{}, &OciNetworkSecurityGroupList{})
}

// RouteRule defines a single route in a route table
type RouteRule struct {
	// NetworkEntityId is the OCID of the gateway (IGW, NGW, etc.)
	NetworkEntityId string `json:"networkEntityId"`

	// Destination is the CIDR, e.g. "0.0.0.0/0"
	Destination string `json:"destination"`

	// DestinationType is "CIDR_BLOCK" (default) or "SERVICE_CIDR_BLOCK"
	DestinationType string `json:"destinationType,omitempty"`

	// Description is an optional description
	Description string `json:"description,omitempty"`
}

// OciRouteTableSpec defines the desired state of OciRouteTable
type OciRouteTableSpec struct {
	// RouteTableId is the OCID of an existing Route Table to bind to (optional)
	RouteTableId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// VcnId is the OCID of the VCN that contains this Route Table
	// +kubebuilder:validation:Required
	VcnId OCID `json:"vcnId"`

	// DisplayName is a user-friendly name for the Route Table
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// RouteRules are the routing rules for this table
	RouteRules []RouteRule `json:"routeRules,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OciRouteTableStatus defines the observed state of OciRouteTable
type OciRouteTableStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciRouteTable",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciRouteTable",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciRouteTable is the Schema for the ociroutetables API
type OciRouteTable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciRouteTableSpec   `json:"spec,omitempty"`
	Status OciRouteTableStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciRouteTableList contains a list of OciRouteTable
type OciRouteTableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciRouteTable `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciRouteTable{}, &OciRouteTableList{})
}
