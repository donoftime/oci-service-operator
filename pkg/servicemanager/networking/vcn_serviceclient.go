/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// VirtualNetworkClientInterface defines the OCI operations used by the VCN and Subnet service managers.
type VirtualNetworkClientInterface interface {
	CreateVcn(ctx context.Context, request ocicore.CreateVcnRequest) (ocicore.CreateVcnResponse, error)
	GetVcn(ctx context.Context, request ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error)
	ListVcns(ctx context.Context, request ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error)
	UpdateVcn(ctx context.Context, request ocicore.UpdateVcnRequest) (ocicore.UpdateVcnResponse, error)
	DeleteVcn(ctx context.Context, request ocicore.DeleteVcnRequest) (ocicore.DeleteVcnResponse, error)
	CreateSubnet(ctx context.Context, request ocicore.CreateSubnetRequest) (ocicore.CreateSubnetResponse, error)
	GetSubnet(ctx context.Context, request ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error)
	ListSubnets(ctx context.Context, request ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error)
	UpdateSubnet(ctx context.Context, request ocicore.UpdateSubnetRequest) (ocicore.UpdateSubnetResponse, error)
	DeleteSubnet(ctx context.Context, request ocicore.DeleteSubnetRequest) (ocicore.DeleteSubnetResponse, error)
	// Internet Gateway
	CreateInternetGateway(ctx context.Context, request ocicore.CreateInternetGatewayRequest) (ocicore.CreateInternetGatewayResponse, error)
	GetInternetGateway(ctx context.Context, request ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error)
	ListInternetGateways(ctx context.Context, request ocicore.ListInternetGatewaysRequest) (ocicore.ListInternetGatewaysResponse, error)
	UpdateInternetGateway(ctx context.Context, request ocicore.UpdateInternetGatewayRequest) (ocicore.UpdateInternetGatewayResponse, error)
	DeleteInternetGateway(ctx context.Context, request ocicore.DeleteInternetGatewayRequest) (ocicore.DeleteInternetGatewayResponse, error)
	// NAT Gateway
	CreateNatGateway(ctx context.Context, request ocicore.CreateNatGatewayRequest) (ocicore.CreateNatGatewayResponse, error)
	GetNatGateway(ctx context.Context, request ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error)
	ListNatGateways(ctx context.Context, request ocicore.ListNatGatewaysRequest) (ocicore.ListNatGatewaysResponse, error)
	UpdateNatGateway(ctx context.Context, request ocicore.UpdateNatGatewayRequest) (ocicore.UpdateNatGatewayResponse, error)
	DeleteNatGateway(ctx context.Context, request ocicore.DeleteNatGatewayRequest) (ocicore.DeleteNatGatewayResponse, error)
	// Service Gateway
	CreateServiceGateway(ctx context.Context, request ocicore.CreateServiceGatewayRequest) (ocicore.CreateServiceGatewayResponse, error)
	GetServiceGateway(ctx context.Context, request ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error)
	ListServiceGateways(ctx context.Context, request ocicore.ListServiceGatewaysRequest) (ocicore.ListServiceGatewaysResponse, error)
	UpdateServiceGateway(ctx context.Context, request ocicore.UpdateServiceGatewayRequest) (ocicore.UpdateServiceGatewayResponse, error)
	DeleteServiceGateway(ctx context.Context, request ocicore.DeleteServiceGatewayRequest) (ocicore.DeleteServiceGatewayResponse, error)
	// DRG
	CreateDrg(ctx context.Context, request ocicore.CreateDrgRequest) (ocicore.CreateDrgResponse, error)
	GetDrg(ctx context.Context, request ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error)
	ListDrgs(ctx context.Context, request ocicore.ListDrgsRequest) (ocicore.ListDrgsResponse, error)
	UpdateDrg(ctx context.Context, request ocicore.UpdateDrgRequest) (ocicore.UpdateDrgResponse, error)
	DeleteDrg(ctx context.Context, request ocicore.DeleteDrgRequest) (ocicore.DeleteDrgResponse, error)
	// Security List
	CreateSecurityList(ctx context.Context, request ocicore.CreateSecurityListRequest) (ocicore.CreateSecurityListResponse, error)
	GetSecurityList(ctx context.Context, request ocicore.GetSecurityListRequest) (ocicore.GetSecurityListResponse, error)
	ListSecurityLists(ctx context.Context, request ocicore.ListSecurityListsRequest) (ocicore.ListSecurityListsResponse, error)
	UpdateSecurityList(ctx context.Context, request ocicore.UpdateSecurityListRequest) (ocicore.UpdateSecurityListResponse, error)
	DeleteSecurityList(ctx context.Context, request ocicore.DeleteSecurityListRequest) (ocicore.DeleteSecurityListResponse, error)
	// Network Security Group
	CreateNetworkSecurityGroup(ctx context.Context, request ocicore.CreateNetworkSecurityGroupRequest) (ocicore.CreateNetworkSecurityGroupResponse, error)
	GetNetworkSecurityGroup(ctx context.Context, request ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error)
	ListNetworkSecurityGroups(ctx context.Context, request ocicore.ListNetworkSecurityGroupsRequest) (ocicore.ListNetworkSecurityGroupsResponse, error)
	UpdateNetworkSecurityGroup(ctx context.Context, request ocicore.UpdateNetworkSecurityGroupRequest) (ocicore.UpdateNetworkSecurityGroupResponse, error)
	DeleteNetworkSecurityGroup(ctx context.Context, request ocicore.DeleteNetworkSecurityGroupRequest) (ocicore.DeleteNetworkSecurityGroupResponse, error)
	// Route Table
	CreateRouteTable(ctx context.Context, request ocicore.CreateRouteTableRequest) (ocicore.CreateRouteTableResponse, error)
	GetRouteTable(ctx context.Context, request ocicore.GetRouteTableRequest) (ocicore.GetRouteTableResponse, error)
	ListRouteTables(ctx context.Context, request ocicore.ListRouteTablesRequest) (ocicore.ListRouteTablesResponse, error)
	UpdateRouteTable(ctx context.Context, request ocicore.UpdateRouteTableRequest) (ocicore.UpdateRouteTableResponse, error)
	DeleteRouteTable(ctx context.Context, request ocicore.DeleteRouteTableRequest) (ocicore.DeleteRouteTableResponse, error)
}

func getVirtualNetworkClient(provider common.ConfigurationProvider) (ocicore.VirtualNetworkClient, error) {
	return ocicore.NewVirtualNetworkClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciVcnServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciSubnetServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciInternetGatewayServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciNatGatewayServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciServiceGatewayServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciDrgServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// CreateVcn calls the OCI API to create a new VCN.
func (c *OciVcnServiceManager) CreateVcn(ctx context.Context, vcn ociv1beta1.OciVcn) (*ocicore.Vcn, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciVcn", "name", vcn.Spec.DisplayName)

	details := ocicore.CreateVcnDetails{
		CompartmentId: common.String(string(vcn.Spec.CompartmentId)),
		DisplayName:   common.String(vcn.Spec.DisplayName),
		CidrBlock:     common.String(vcn.Spec.CidrBlock),
		FreeformTags:  vcn.Spec.FreeFormTags,
	}
	if vcn.Spec.DnsLabel != "" {
		details.DnsLabel = common.String(vcn.Spec.DnsLabel)
	}
	if vcn.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&vcn.Spec.DefinedTags)
	}

	resp, err := client.CreateVcn(ctx, ocicore.CreateVcnRequest{CreateVcnDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.Vcn, nil
}

// GetVcn retrieves a VCN by OCID.
func (c *OciVcnServiceManager) GetVcn(ctx context.Context, vcnId ociv1beta1.OCID) (*ocicore.Vcn, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetVcn(ctx, ocicore.GetVcnRequest{VcnId: common.String(string(vcnId))})
	if err != nil {
		return nil, err
	}
	return &resp.Vcn, nil
}

// GetVcnOcid looks up an existing VCN by display name and returns its OCID if found.
func (c *OciVcnServiceManager) GetVcnOcid(ctx context.Context, vcn ociv1beta1.OciVcn) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListVcns(ctx, ocicore.ListVcnsRequest{
		CompartmentId: common.String(string(vcn.Spec.CompartmentId)),
		DisplayName:   common.String(vcn.Spec.DisplayName),
		Limit:         common.Int(1),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing VCNs")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "AVAILABLE" || state == "PROVISIONING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("OciVcn %s exists with OCID %s", vcn.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciVcn %s does not exist", vcn.Spec.DisplayName))
	return nil, nil
}

// UpdateVcn updates an existing VCN's display name and tags.
func (c *OciVcnServiceManager) UpdateVcn(ctx context.Context, vcn *ociv1beta1.OciVcn) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetVcn(ctx, vcn.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateVcnDetails{}
	updateNeeded := false

	if vcn.Spec.DisplayName != "" && (existing.DisplayName == nil || *existing.DisplayName != vcn.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(vcn.Spec.DisplayName)
		updateNeeded = true
	}
	if len(vcn.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = vcn.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateVcn(ctx, ocicore.UpdateVcnRequest{
		VcnId:            common.String(string(vcn.Status.OsokStatus.Ocid)),
		UpdateVcnDetails: updateDetails,
	})
	return err
}

// DeleteVcn deletes the VCN for the given OCID.
func (c *OciVcnServiceManager) DeleteVcn(ctx context.Context, vcnId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteVcn(ctx, ocicore.DeleteVcnRequest{VcnId: common.String(string(vcnId))})
	return err
}

// CreateSubnet calls the OCI API to create a new Subnet.
func (c *OciSubnetServiceManager) CreateSubnet(ctx context.Context, subnet ociv1beta1.OciSubnet) (*ocicore.Subnet, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciSubnet", "name", subnet.Spec.DisplayName)

	details := ocicore.CreateSubnetDetails{
		CompartmentId: common.String(string(subnet.Spec.CompartmentId)),
		VcnId:         common.String(string(subnet.Spec.VcnId)),
		CidrBlock:     common.String(subnet.Spec.CidrBlock),
		DisplayName:   common.String(subnet.Spec.DisplayName),
		FreeformTags:  subnet.Spec.FreeFormTags,
	}
	if subnet.Spec.AvailabilityDomain != "" {
		details.AvailabilityDomain = common.String(subnet.Spec.AvailabilityDomain)
	}
	if subnet.Spec.DnsLabel != "" {
		details.DnsLabel = common.String(subnet.Spec.DnsLabel)
	}
	if subnet.Spec.ProhibitPublicIpOnVnic {
		details.ProhibitPublicIpOnVnic = common.Bool(subnet.Spec.ProhibitPublicIpOnVnic)
	}
	if string(subnet.Spec.RouteTableId) != "" {
		details.RouteTableId = common.String(string(subnet.Spec.RouteTableId))
	}
	if len(subnet.Spec.SecurityListIds) > 0 {
		slIds := make([]string, len(subnet.Spec.SecurityListIds))
		for i, id := range subnet.Spec.SecurityListIds {
			slIds[i] = string(id)
		}
		details.SecurityListIds = slIds
	}
	if subnet.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&subnet.Spec.DefinedTags)
	}

	resp, err := client.CreateSubnet(ctx, ocicore.CreateSubnetRequest{CreateSubnetDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.Subnet, nil
}

// GetSubnet retrieves a Subnet by OCID.
func (c *OciSubnetServiceManager) GetSubnet(ctx context.Context, subnetId ociv1beta1.OCID) (*ocicore.Subnet, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetSubnet(ctx, ocicore.GetSubnetRequest{SubnetId: common.String(string(subnetId))})
	if err != nil {
		return nil, err
	}
	return &resp.Subnet, nil
}

// GetSubnetOcid looks up an existing Subnet by display name within a VCN and returns its OCID if found.
func (c *OciSubnetServiceManager) GetSubnetOcid(ctx context.Context, subnet ociv1beta1.OciSubnet) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListSubnets(ctx, ocicore.ListSubnetsRequest{
		CompartmentId: common.String(string(subnet.Spec.CompartmentId)),
		VcnId:         common.String(string(subnet.Spec.VcnId)),
		DisplayName:   common.String(subnet.Spec.DisplayName),
		Limit:         common.Int(1),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Subnets")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "AVAILABLE" || state == "PROVISIONING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("OciSubnet %s exists with OCID %s", subnet.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciSubnet %s does not exist", subnet.Spec.DisplayName))
	return nil, nil
}

// UpdateSubnet updates an existing Subnet's display name and tags.
func (c *OciSubnetServiceManager) UpdateSubnet(ctx context.Context, subnet *ociv1beta1.OciSubnet) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetSubnet(ctx, subnet.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateSubnetDetails{}
	updateNeeded := false

	if subnet.Spec.DisplayName != "" && (existing.DisplayName == nil || *existing.DisplayName != subnet.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(subnet.Spec.DisplayName)
		updateNeeded = true
	}
	if len(subnet.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = subnet.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateSubnet(ctx, ocicore.UpdateSubnetRequest{
		SubnetId:            common.String(string(subnet.Status.OsokStatus.Ocid)),
		UpdateSubnetDetails: updateDetails,
	})
	return err
}

// DeleteSubnet deletes the Subnet for the given OCID.
func (c *OciSubnetServiceManager) DeleteSubnet(ctx context.Context, subnetId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteSubnet(ctx, ocicore.DeleteSubnetRequest{SubnetId: common.String(string(subnetId))})
	return err
}

// --- Internet Gateway CRUD ---

// CreateInternetGateway calls the OCI API to create a new Internet Gateway.
func (c *OciInternetGatewayServiceManager) CreateInternetGateway(ctx context.Context, igw ociv1beta1.OciInternetGateway) (*ocicore.InternetGateway, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciInternetGateway", "name", igw.Spec.DisplayName)

	isEnabled := igw.Spec.IsEnabled
	details := ocicore.CreateInternetGatewayDetails{
		CompartmentId: common.String(string(igw.Spec.CompartmentId)),
		VcnId:         common.String(string(igw.Spec.VcnId)),
		DisplayName:   common.String(igw.Spec.DisplayName),
		IsEnabled:     common.Bool(isEnabled),
		FreeformTags:  igw.Spec.FreeFormTags,
	}
	if igw.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&igw.Spec.DefinedTags)
	}

	resp, err := client.CreateInternetGateway(ctx, ocicore.CreateInternetGatewayRequest{CreateInternetGatewayDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.InternetGateway, nil
}

// GetInternetGateway retrieves an Internet Gateway by OCID.
func (c *OciInternetGatewayServiceManager) GetInternetGateway(ctx context.Context, igwId ociv1beta1.OCID) (*ocicore.InternetGateway, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetInternetGateway(ctx, ocicore.GetInternetGatewayRequest{IgId: common.String(string(igwId))})
	if err != nil {
		return nil, err
	}
	return &resp.InternetGateway, nil
}

// GetInternetGatewayOcid looks up an existing Internet Gateway by display name and returns its OCID if found.
func (c *OciInternetGatewayServiceManager) GetInternetGatewayOcid(ctx context.Context, igw ociv1beta1.OciInternetGateway) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListInternetGateways(ctx, ocicore.ListInternetGatewaysRequest{
		CompartmentId: common.String(string(igw.Spec.CompartmentId)),
		VcnId:         common.String(string(igw.Spec.VcnId)),
		DisplayName:   common.String(igw.Spec.DisplayName),
		Limit:         common.Int(1),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Internet Gateways")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "AVAILABLE" || state == "PROVISIONING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("OciInternetGateway %s exists with OCID %s", igw.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciInternetGateway %s does not exist", igw.Spec.DisplayName))
	return nil, nil
}

// UpdateInternetGateway updates an existing Internet Gateway's display name and tags.
func (c *OciInternetGatewayServiceManager) UpdateInternetGateway(ctx context.Context, igw *ociv1beta1.OciInternetGateway) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetInternetGateway(ctx, igw.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateInternetGatewayDetails{}
	updateNeeded := false

	if igw.Spec.DisplayName != "" && (existing.DisplayName == nil || *existing.DisplayName != igw.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(igw.Spec.DisplayName)
		updateNeeded = true
	}
	if len(igw.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = igw.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateInternetGateway(ctx, ocicore.UpdateInternetGatewayRequest{
		IgId:                        common.String(string(igw.Status.OsokStatus.Ocid)),
		UpdateInternetGatewayDetails: updateDetails,
	})
	return err
}

// DeleteInternetGateway deletes the Internet Gateway for the given OCID.
func (c *OciInternetGatewayServiceManager) DeleteInternetGateway(ctx context.Context, igwId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteInternetGateway(ctx, ocicore.DeleteInternetGatewayRequest{IgId: common.String(string(igwId))})
	return err
}

// --- NAT Gateway CRUD ---

// CreateNatGateway calls the OCI API to create a new NAT Gateway.
func (c *OciNatGatewayServiceManager) CreateNatGateway(ctx context.Context, nat ociv1beta1.OciNatGateway) (*ocicore.NatGateway, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciNatGateway", "name", nat.Spec.DisplayName)

	details := ocicore.CreateNatGatewayDetails{
		CompartmentId: common.String(string(nat.Spec.CompartmentId)),
		VcnId:         common.String(string(nat.Spec.VcnId)),
		DisplayName:   common.String(nat.Spec.DisplayName),
		FreeformTags:  nat.Spec.FreeFormTags,
	}
	if nat.Spec.BlockTraffic {
		details.BlockTraffic = common.Bool(nat.Spec.BlockTraffic)
	}
	if nat.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&nat.Spec.DefinedTags)
	}

	resp, err := client.CreateNatGateway(ctx, ocicore.CreateNatGatewayRequest{CreateNatGatewayDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.NatGateway, nil
}

// GetNatGateway retrieves a NAT Gateway by OCID.
func (c *OciNatGatewayServiceManager) GetNatGateway(ctx context.Context, natId ociv1beta1.OCID) (*ocicore.NatGateway, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetNatGateway(ctx, ocicore.GetNatGatewayRequest{NatGatewayId: common.String(string(natId))})
	if err != nil {
		return nil, err
	}
	return &resp.NatGateway, nil
}

// GetNatGatewayOcid looks up an existing NAT Gateway by display name and returns its OCID if found.
func (c *OciNatGatewayServiceManager) GetNatGatewayOcid(ctx context.Context, nat ociv1beta1.OciNatGateway) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListNatGateways(ctx, ocicore.ListNatGatewaysRequest{
		CompartmentId: common.String(string(nat.Spec.CompartmentId)),
		VcnId:         common.String(string(nat.Spec.VcnId)),
		DisplayName:   common.String(nat.Spec.DisplayName),
		Limit:         common.Int(1),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing NAT Gateways")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "AVAILABLE" || state == "PROVISIONING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("OciNatGateway %s exists with OCID %s", nat.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciNatGateway %s does not exist", nat.Spec.DisplayName))
	return nil, nil
}

// UpdateNatGateway updates an existing NAT Gateway's display name and tags.
func (c *OciNatGatewayServiceManager) UpdateNatGateway(ctx context.Context, nat *ociv1beta1.OciNatGateway) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetNatGateway(ctx, nat.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateNatGatewayDetails{}
	updateNeeded := false

	if nat.Spec.DisplayName != "" && (existing.DisplayName == nil || *existing.DisplayName != nat.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(nat.Spec.DisplayName)
		updateNeeded = true
	}
	if len(nat.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = nat.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateNatGateway(ctx, ocicore.UpdateNatGatewayRequest{
		NatGatewayId:          common.String(string(nat.Status.OsokStatus.Ocid)),
		UpdateNatGatewayDetails: updateDetails,
	})
	return err
}

// DeleteNatGateway deletes the NAT Gateway for the given OCID.
func (c *OciNatGatewayServiceManager) DeleteNatGateway(ctx context.Context, natId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteNatGateway(ctx, ocicore.DeleteNatGatewayRequest{NatGatewayId: common.String(string(natId))})
	return err
}

// --- Service Gateway CRUD ---

// CreateServiceGateway calls the OCI API to create a new Service Gateway.
func (c *OciServiceGatewayServiceManager) CreateServiceGateway(ctx context.Context, sgw ociv1beta1.OciServiceGateway) (*ocicore.ServiceGateway, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciServiceGateway", "name", sgw.Spec.DisplayName)

	services := make([]ocicore.ServiceIdRequestDetails, len(sgw.Spec.Services))
	for i, svcId := range sgw.Spec.Services {
		services[i] = ocicore.ServiceIdRequestDetails{ServiceId: common.String(svcId)}
	}

	details := ocicore.CreateServiceGatewayDetails{
		CompartmentId: common.String(string(sgw.Spec.CompartmentId)),
		VcnId:         common.String(string(sgw.Spec.VcnId)),
		DisplayName:   common.String(sgw.Spec.DisplayName),
		Services:      services,
		FreeformTags:  sgw.Spec.FreeFormTags,
	}
	if sgw.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&sgw.Spec.DefinedTags)
	}

	resp, err := client.CreateServiceGateway(ctx, ocicore.CreateServiceGatewayRequest{CreateServiceGatewayDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.ServiceGateway, nil
}

// GetServiceGateway retrieves a Service Gateway by OCID.
func (c *OciServiceGatewayServiceManager) GetServiceGateway(ctx context.Context, sgwId ociv1beta1.OCID) (*ocicore.ServiceGateway, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetServiceGateway(ctx, ocicore.GetServiceGatewayRequest{ServiceGatewayId: common.String(string(sgwId))})
	if err != nil {
		return nil, err
	}
	return &resp.ServiceGateway, nil
}

// GetServiceGatewayOcid looks up an existing Service Gateway by display name and returns its OCID if found.
func (c *OciServiceGatewayServiceManager) GetServiceGatewayOcid(ctx context.Context, sgw ociv1beta1.OciServiceGateway) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListServiceGateways(ctx, ocicore.ListServiceGatewaysRequest{
		CompartmentId: common.String(string(sgw.Spec.CompartmentId)),
		VcnId:         common.String(string(sgw.Spec.VcnId)),
		Limit:         common.Int(1),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Service Gateways")
		return nil, err
	}

	for _, item := range resp.Items {
		if item.DisplayName != nil && *item.DisplayName == sgw.Spec.DisplayName {
			state := string(item.LifecycleState)
			if state == "AVAILABLE" || state == "PROVISIONING" || state == "UPDATING" {
				c.Log.DebugLog(fmt.Sprintf("OciServiceGateway %s exists with OCID %s", sgw.Spec.DisplayName, *item.Id))
				return (*ociv1beta1.OCID)(item.Id), nil
			}
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciServiceGateway %s does not exist", sgw.Spec.DisplayName))
	return nil, nil
}

// UpdateServiceGateway updates an existing Service Gateway's display name and tags.
func (c *OciServiceGatewayServiceManager) UpdateServiceGateway(ctx context.Context, sgw *ociv1beta1.OciServiceGateway) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetServiceGateway(ctx, sgw.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateServiceGatewayDetails{}
	updateNeeded := false

	if sgw.Spec.DisplayName != "" && (existing.DisplayName == nil || *existing.DisplayName != sgw.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(sgw.Spec.DisplayName)
		updateNeeded = true
	}
	if len(sgw.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = sgw.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateServiceGateway(ctx, ocicore.UpdateServiceGatewayRequest{
		ServiceGatewayId:            common.String(string(sgw.Status.OsokStatus.Ocid)),
		UpdateServiceGatewayDetails: updateDetails,
	})
	return err
}

// DeleteServiceGateway deletes the Service Gateway for the given OCID.
func (c *OciServiceGatewayServiceManager) DeleteServiceGateway(ctx context.Context, sgwId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteServiceGateway(ctx, ocicore.DeleteServiceGatewayRequest{ServiceGatewayId: common.String(string(sgwId))})
	return err
}

// --- DRG CRUD ---

// CreateDrg calls the OCI API to create a new DRG.
func (c *OciDrgServiceManager) CreateDrg(ctx context.Context, drg ociv1beta1.OciDrg) (*ocicore.Drg, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciDrg", "name", drg.Spec.DisplayName)

	details := ocicore.CreateDrgDetails{
		CompartmentId: common.String(string(drg.Spec.CompartmentId)),
		DisplayName:   common.String(drg.Spec.DisplayName),
		FreeformTags:  drg.Spec.FreeFormTags,
	}
	if drg.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&drg.Spec.DefinedTags)
	}

	resp, err := client.CreateDrg(ctx, ocicore.CreateDrgRequest{CreateDrgDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.Drg, nil
}

// GetDrg retrieves a DRG by OCID.
func (c *OciDrgServiceManager) GetDrg(ctx context.Context, drgId ociv1beta1.OCID) (*ocicore.Drg, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetDrg(ctx, ocicore.GetDrgRequest{DrgId: common.String(string(drgId))})
	if err != nil {
		return nil, err
	}
	return &resp.Drg, nil
}

// GetDrgOcid looks up an existing DRG by display name and returns its OCID if found.
func (c *OciDrgServiceManager) GetDrgOcid(ctx context.Context, drg ociv1beta1.OciDrg) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListDrgs(ctx, ocicore.ListDrgsRequest{
		CompartmentId: common.String(string(drg.Spec.CompartmentId)),
		Limit:         common.Int(50),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing DRGs")
		return nil, err
	}

	for _, item := range resp.Items {
		if item.DisplayName != nil && *item.DisplayName == drg.Spec.DisplayName {
			state := string(item.LifecycleState)
			if state == "AVAILABLE" || state == "PROVISIONING" || state == "UPDATING" {
				c.Log.DebugLog(fmt.Sprintf("OciDrg %s exists with OCID %s", drg.Spec.DisplayName, *item.Id))
				return (*ociv1beta1.OCID)(item.Id), nil
			}
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciDrg %s does not exist", drg.Spec.DisplayName))
	return nil, nil
}

// UpdateDrg updates an existing DRG's display name and tags.
func (c *OciDrgServiceManager) UpdateDrg(ctx context.Context, drg *ociv1beta1.OciDrg) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetDrg(ctx, drg.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateDrgDetails{}
	updateNeeded := false

	if drg.Spec.DisplayName != "" && (existing.DisplayName == nil || *existing.DisplayName != drg.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(drg.Spec.DisplayName)
		updateNeeded = true
	}
	if len(drg.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = drg.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateDrg(ctx, ocicore.UpdateDrgRequest{
		DrgId:            common.String(string(drg.Status.OsokStatus.Ocid)),
		UpdateDrgDetails: updateDetails,
	})
	return err
}

// DeleteDrg deletes the DRG for the given OCID.
func (c *OciDrgServiceManager) DeleteDrg(ctx context.Context, drgId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteDrg(ctx, ocicore.DeleteDrgRequest{DrgId: common.String(string(drgId))})
	return err
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciSecurityListServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciNetworkSecurityGroupServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciRouteTableServiceManager) getOCIClient() (VirtualNetworkClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getVirtualNetworkClient(c.Provider)
}

// --- Security List CRUD ---

func buildIngressRules(rules []ociv1beta1.IngressSecurityRule) []ocicore.IngressSecurityRule {
	result := make([]ocicore.IngressSecurityRule, len(rules))
	for i, r := range rules {
		rule := ocicore.IngressSecurityRule{
			Protocol:    common.String(r.Protocol),
			Source:      common.String(r.Source),
			IsStateless: common.Bool(r.IsStateless),
		}
		if r.Description != "" {
			rule.Description = common.String(r.Description)
		}
		if r.TcpOptions != nil {
			tcpOpts := &ocicore.TcpOptions{}
			if r.TcpOptions.DestinationPortRange != nil {
				tcpOpts.DestinationPortRange = &ocicore.PortRange{
					Min: common.Int(r.TcpOptions.DestinationPortRange.Min),
					Max: common.Int(r.TcpOptions.DestinationPortRange.Max),
				}
			}
			if r.TcpOptions.SourcePortRange != nil {
				tcpOpts.SourcePortRange = &ocicore.PortRange{
					Min: common.Int(r.TcpOptions.SourcePortRange.Min),
					Max: common.Int(r.TcpOptions.SourcePortRange.Max),
				}
			}
			rule.TcpOptions = tcpOpts
		}
		if r.UdpOptions != nil {
			udpOpts := &ocicore.UdpOptions{}
			if r.UdpOptions.DestinationPortRange != nil {
				udpOpts.DestinationPortRange = &ocicore.PortRange{
					Min: common.Int(r.UdpOptions.DestinationPortRange.Min),
					Max: common.Int(r.UdpOptions.DestinationPortRange.Max),
				}
			}
			if r.UdpOptions.SourcePortRange != nil {
				udpOpts.SourcePortRange = &ocicore.PortRange{
					Min: common.Int(r.UdpOptions.SourcePortRange.Min),
					Max: common.Int(r.UdpOptions.SourcePortRange.Max),
				}
			}
			rule.UdpOptions = udpOpts
		}
		result[i] = rule
	}
	return result
}

func buildEgressRules(rules []ociv1beta1.EgressSecurityRule) []ocicore.EgressSecurityRule {
	result := make([]ocicore.EgressSecurityRule, len(rules))
	for i, r := range rules {
		rule := ocicore.EgressSecurityRule{
			Protocol:    common.String(r.Protocol),
			Destination: common.String(r.Destination),
			IsStateless: common.Bool(r.IsStateless),
		}
		if r.Description != "" {
			rule.Description = common.String(r.Description)
		}
		if r.TcpOptions != nil {
			tcpOpts := &ocicore.TcpOptions{}
			if r.TcpOptions.DestinationPortRange != nil {
				tcpOpts.DestinationPortRange = &ocicore.PortRange{
					Min: common.Int(r.TcpOptions.DestinationPortRange.Min),
					Max: common.Int(r.TcpOptions.DestinationPortRange.Max),
				}
			}
			if r.TcpOptions.SourcePortRange != nil {
				tcpOpts.SourcePortRange = &ocicore.PortRange{
					Min: common.Int(r.TcpOptions.SourcePortRange.Min),
					Max: common.Int(r.TcpOptions.SourcePortRange.Max),
				}
			}
			rule.TcpOptions = tcpOpts
		}
		if r.UdpOptions != nil {
			udpOpts := &ocicore.UdpOptions{}
			if r.UdpOptions.DestinationPortRange != nil {
				udpOpts.DestinationPortRange = &ocicore.PortRange{
					Min: common.Int(r.UdpOptions.DestinationPortRange.Min),
					Max: common.Int(r.UdpOptions.DestinationPortRange.Max),
				}
			}
			if r.UdpOptions.SourcePortRange != nil {
				udpOpts.SourcePortRange = &ocicore.PortRange{
					Min: common.Int(r.UdpOptions.SourcePortRange.Min),
					Max: common.Int(r.UdpOptions.SourcePortRange.Max),
				}
			}
			rule.UdpOptions = udpOpts
		}
		result[i] = rule
	}
	return result
}

// CreateSecurityList calls the OCI API to create a new Security List.
func (c *OciSecurityListServiceManager) CreateSecurityList(ctx context.Context, sl ociv1beta1.OciSecurityList) (*ocicore.SecurityList, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciSecurityList", "name", sl.Spec.DisplayName)

	details := ocicore.CreateSecurityListDetails{
		CompartmentId:        common.String(string(sl.Spec.CompartmentId)),
		VcnId:                common.String(string(sl.Spec.VcnId)),
		DisplayName:          common.String(sl.Spec.DisplayName),
		IngressSecurityRules: buildIngressRules(sl.Spec.IngressSecurityRules),
		EgressSecurityRules:  buildEgressRules(sl.Spec.EgressSecurityRules),
		FreeformTags:         sl.Spec.FreeFormTags,
	}
	if sl.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&sl.Spec.DefinedTags)
	}

	resp, err := client.CreateSecurityList(ctx, ocicore.CreateSecurityListRequest{CreateSecurityListDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.SecurityList, nil
}

// GetSecurityList retrieves a Security List by OCID.
func (c *OciSecurityListServiceManager) GetSecurityList(ctx context.Context, slId ociv1beta1.OCID) (*ocicore.SecurityList, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetSecurityList(ctx, ocicore.GetSecurityListRequest{SecurityListId: common.String(string(slId))})
	if err != nil {
		return nil, err
	}
	return &resp.SecurityList, nil
}

// GetSecurityListOcid looks up an existing Security List by display name and returns its OCID if found.
func (c *OciSecurityListServiceManager) GetSecurityListOcid(ctx context.Context, sl ociv1beta1.OciSecurityList) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListSecurityLists(ctx, ocicore.ListSecurityListsRequest{
		CompartmentId: common.String(string(sl.Spec.CompartmentId)),
		VcnId:         common.String(string(sl.Spec.VcnId)),
		DisplayName:   common.String(sl.Spec.DisplayName),
		Limit:         common.Int(1),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Security Lists")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "AVAILABLE" || state == "PROVISIONING" {
			c.Log.DebugLog(fmt.Sprintf("OciSecurityList %s exists with OCID %s", sl.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciSecurityList %s does not exist", sl.Spec.DisplayName))
	return nil, nil
}

// UpdateSecurityList updates an existing Security List's display name, tags, and rules.
func (c *OciSecurityListServiceManager) UpdateSecurityList(ctx context.Context, sl *ociv1beta1.OciSecurityList) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateSecurityListDetails{}
	updateNeeded := false

	if sl.Spec.DisplayName != "" {
		updateDetails.DisplayName = common.String(sl.Spec.DisplayName)
		updateNeeded = true
	}
	if len(sl.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = sl.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateSecurityList(ctx, ocicore.UpdateSecurityListRequest{
		SecurityListId:            common.String(string(sl.Status.OsokStatus.Ocid)),
		UpdateSecurityListDetails: updateDetails,
	})
	return err
}

// DeleteSecurityList deletes the Security List for the given OCID.
func (c *OciSecurityListServiceManager) DeleteSecurityList(ctx context.Context, slId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteSecurityList(ctx, ocicore.DeleteSecurityListRequest{SecurityListId: common.String(string(slId))})
	return err
}

// --- Network Security Group CRUD ---

// CreateNetworkSecurityGroup calls the OCI API to create a new NSG.
func (c *OciNetworkSecurityGroupServiceManager) CreateNetworkSecurityGroup(ctx context.Context, nsg ociv1beta1.OciNetworkSecurityGroup) (*ocicore.NetworkSecurityGroup, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciNetworkSecurityGroup", "name", nsg.Spec.DisplayName)

	details := ocicore.CreateNetworkSecurityGroupDetails{
		CompartmentId: common.String(string(nsg.Spec.CompartmentId)),
		VcnId:         common.String(string(nsg.Spec.VcnId)),
		DisplayName:   common.String(nsg.Spec.DisplayName),
		FreeformTags:  nsg.Spec.FreeFormTags,
	}
	if nsg.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&nsg.Spec.DefinedTags)
	}

	resp, err := client.CreateNetworkSecurityGroup(ctx, ocicore.CreateNetworkSecurityGroupRequest{CreateNetworkSecurityGroupDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.NetworkSecurityGroup, nil
}

// GetNetworkSecurityGroup retrieves an NSG by OCID.
func (c *OciNetworkSecurityGroupServiceManager) GetNetworkSecurityGroup(ctx context.Context, nsgId ociv1beta1.OCID) (*ocicore.NetworkSecurityGroup, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetNetworkSecurityGroup(ctx, ocicore.GetNetworkSecurityGroupRequest{NetworkSecurityGroupId: common.String(string(nsgId))})
	if err != nil {
		return nil, err
	}
	return &resp.NetworkSecurityGroup, nil
}

// GetNetworkSecurityGroupOcid looks up an existing NSG by display name and returns its OCID if found.
func (c *OciNetworkSecurityGroupServiceManager) GetNetworkSecurityGroupOcid(ctx context.Context, nsg ociv1beta1.OciNetworkSecurityGroup) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListNetworkSecurityGroups(ctx, ocicore.ListNetworkSecurityGroupsRequest{
		CompartmentId: common.String(string(nsg.Spec.CompartmentId)),
		VcnId:         common.String(string(nsg.Spec.VcnId)),
		DisplayName:   common.String(nsg.Spec.DisplayName),
		Limit:         common.Int(1),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Network Security Groups")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "AVAILABLE" || state == "PROVISIONING" {
			c.Log.DebugLog(fmt.Sprintf("OciNetworkSecurityGroup %s exists with OCID %s", nsg.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciNetworkSecurityGroup %s does not exist", nsg.Spec.DisplayName))
	return nil, nil
}

// UpdateNetworkSecurityGroup updates an existing NSG's display name and tags.
func (c *OciNetworkSecurityGroupServiceManager) UpdateNetworkSecurityGroup(ctx context.Context, nsg *ociv1beta1.OciNetworkSecurityGroup) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetNetworkSecurityGroup(ctx, nsg.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateNetworkSecurityGroupDetails{}
	updateNeeded := false

	if nsg.Spec.DisplayName != "" && (existing.DisplayName == nil || *existing.DisplayName != nsg.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(nsg.Spec.DisplayName)
		updateNeeded = true
	}
	if len(nsg.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = nsg.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateNetworkSecurityGroup(ctx, ocicore.UpdateNetworkSecurityGroupRequest{
		NetworkSecurityGroupId:            common.String(string(nsg.Status.OsokStatus.Ocid)),
		UpdateNetworkSecurityGroupDetails: updateDetails,
	})
	return err
}

// DeleteNetworkSecurityGroup deletes the NSG for the given OCID.
func (c *OciNetworkSecurityGroupServiceManager) DeleteNetworkSecurityGroup(ctx context.Context, nsgId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteNetworkSecurityGroup(ctx, ocicore.DeleteNetworkSecurityGroupRequest{NetworkSecurityGroupId: common.String(string(nsgId))})
	return err
}

// --- Route Table CRUD ---

func buildRouteRules(rules []ociv1beta1.RouteRule) []ocicore.RouteRule {
	result := make([]ocicore.RouteRule, len(rules))
	for i, r := range rules {
		destType := r.DestinationType
		if destType == "" {
			destType = "CIDR_BLOCK"
		}
		rule := ocicore.RouteRule{
			NetworkEntityId: common.String(r.NetworkEntityId),
			Destination:     common.String(r.Destination),
			DestinationType: ocicore.RouteRuleDestinationTypeEnum(destType),
		}
		if r.Description != "" {
			rule.Description = common.String(r.Description)
		}
		result[i] = rule
	}
	return result
}

// CreateRouteTable calls the OCI API to create a new Route Table.
func (c *OciRouteTableServiceManager) CreateRouteTable(ctx context.Context, rt ociv1beta1.OciRouteTable) (*ocicore.RouteTable, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciRouteTable", "name", rt.Spec.DisplayName)

	details := ocicore.CreateRouteTableDetails{
		CompartmentId: common.String(string(rt.Spec.CompartmentId)),
		VcnId:         common.String(string(rt.Spec.VcnId)),
		DisplayName:   common.String(rt.Spec.DisplayName),
		RouteRules:    buildRouteRules(rt.Spec.RouteRules),
		FreeformTags:  rt.Spec.FreeFormTags,
	}
	if rt.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&rt.Spec.DefinedTags)
	}

	resp, err := client.CreateRouteTable(ctx, ocicore.CreateRouteTableRequest{CreateRouteTableDetails: details})
	if err != nil {
		return nil, err
	}
	return &resp.RouteTable, nil
}

// GetRouteTable retrieves a Route Table by OCID.
func (c *OciRouteTableServiceManager) GetRouteTable(ctx context.Context, rtId ociv1beta1.OCID) (*ocicore.RouteTable, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetRouteTable(ctx, ocicore.GetRouteTableRequest{RtId: common.String(string(rtId))})
	if err != nil {
		return nil, err
	}
	return &resp.RouteTable, nil
}

// GetRouteTableOcid looks up an existing Route Table by display name and returns its OCID if found.
func (c *OciRouteTableServiceManager) GetRouteTableOcid(ctx context.Context, rt ociv1beta1.OciRouteTable) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.ListRouteTables(ctx, ocicore.ListRouteTablesRequest{
		CompartmentId: common.String(string(rt.Spec.CompartmentId)),
		VcnId:         common.String(string(rt.Spec.VcnId)),
		DisplayName:   common.String(rt.Spec.DisplayName),
		Limit:         common.Int(1),
	})
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Route Tables")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "AVAILABLE" || state == "PROVISIONING" {
			c.Log.DebugLog(fmt.Sprintf("OciRouteTable %s exists with OCID %s", rt.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciRouteTable %s does not exist", rt.Spec.DisplayName))
	return nil, nil
}

// UpdateRouteTable updates an existing Route Table's display name, tags, and route rules.
func (c *OciRouteTableServiceManager) UpdateRouteTable(ctx context.Context, rt *ociv1beta1.OciRouteTable) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	updateDetails := ocicore.UpdateRouteTableDetails{}
	updateNeeded := false

	if rt.Spec.DisplayName != "" {
		updateDetails.DisplayName = common.String(rt.Spec.DisplayName)
		updateNeeded = true
	}
	if len(rt.Spec.FreeFormTags) > 0 {
		updateDetails.FreeformTags = rt.Spec.FreeFormTags
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateRouteTable(ctx, ocicore.UpdateRouteTableRequest{
		RtId:                    common.String(string(rt.Status.OsokStatus.Ocid)),
		UpdateRouteTableDetails: updateDetails,
	})
	return err
}

// DeleteRouteTable deletes the Route Table for the given OCID.
func (c *OciRouteTableServiceManager) DeleteRouteTable(ctx context.Context, rtId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteRouteTable(ctx, ocicore.DeleteRouteTableRequest{RtId: common.String(string(rtId))})
	return err
}
