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
