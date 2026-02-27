/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/networking"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ---------------------------------------------------------------------------
// fakeVirtualNetworkClient — implements VirtualNetworkClientInterface for testing.
// ---------------------------------------------------------------------------

type fakeVirtualNetworkClient struct {
	createVcnFn    func(ctx context.Context, req ocicore.CreateVcnRequest) (ocicore.CreateVcnResponse, error)
	getVcnFn       func(ctx context.Context, req ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error)
	listVcnsFn     func(ctx context.Context, req ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error)
	updateVcnFn    func(ctx context.Context, req ocicore.UpdateVcnRequest) (ocicore.UpdateVcnResponse, error)
	deleteVcnFn    func(ctx context.Context, req ocicore.DeleteVcnRequest) (ocicore.DeleteVcnResponse, error)
	createSubnetFn func(ctx context.Context, req ocicore.CreateSubnetRequest) (ocicore.CreateSubnetResponse, error)
	getSubnetFn    func(ctx context.Context, req ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error)
	listSubnetsFn  func(ctx context.Context, req ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error)
	updateSubnetFn func(ctx context.Context, req ocicore.UpdateSubnetRequest) (ocicore.UpdateSubnetResponse, error)
	deleteSubnetFn func(ctx context.Context, req ocicore.DeleteSubnetRequest) (ocicore.DeleteSubnetResponse, error)
	// Internet Gateway
	createInternetGatewayFn func(ctx context.Context, req ocicore.CreateInternetGatewayRequest) (ocicore.CreateInternetGatewayResponse, error)
	getInternetGatewayFn    func(ctx context.Context, req ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error)
	listInternetGatewaysFn  func(ctx context.Context, req ocicore.ListInternetGatewaysRequest) (ocicore.ListInternetGatewaysResponse, error)
	updateInternetGatewayFn func(ctx context.Context, req ocicore.UpdateInternetGatewayRequest) (ocicore.UpdateInternetGatewayResponse, error)
	deleteInternetGatewayFn func(ctx context.Context, req ocicore.DeleteInternetGatewayRequest) (ocicore.DeleteInternetGatewayResponse, error)
	// NAT Gateway
	createNatGatewayFn func(ctx context.Context, req ocicore.CreateNatGatewayRequest) (ocicore.CreateNatGatewayResponse, error)
	getNatGatewayFn    func(ctx context.Context, req ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error)
	listNatGatewaysFn  func(ctx context.Context, req ocicore.ListNatGatewaysRequest) (ocicore.ListNatGatewaysResponse, error)
	updateNatGatewayFn func(ctx context.Context, req ocicore.UpdateNatGatewayRequest) (ocicore.UpdateNatGatewayResponse, error)
	deleteNatGatewayFn func(ctx context.Context, req ocicore.DeleteNatGatewayRequest) (ocicore.DeleteNatGatewayResponse, error)
	// Service Gateway
	createServiceGatewayFn func(ctx context.Context, req ocicore.CreateServiceGatewayRequest) (ocicore.CreateServiceGatewayResponse, error)
	getServiceGatewayFn    func(ctx context.Context, req ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error)
	listServiceGatewaysFn  func(ctx context.Context, req ocicore.ListServiceGatewaysRequest) (ocicore.ListServiceGatewaysResponse, error)
	updateServiceGatewayFn func(ctx context.Context, req ocicore.UpdateServiceGatewayRequest) (ocicore.UpdateServiceGatewayResponse, error)
	deleteServiceGatewayFn func(ctx context.Context, req ocicore.DeleteServiceGatewayRequest) (ocicore.DeleteServiceGatewayResponse, error)
	// DRG
	createDrgFn func(ctx context.Context, req ocicore.CreateDrgRequest) (ocicore.CreateDrgResponse, error)
	getDrgFn    func(ctx context.Context, req ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error)
	listDrgsFn  func(ctx context.Context, req ocicore.ListDrgsRequest) (ocicore.ListDrgsResponse, error)
	updateDrgFn func(ctx context.Context, req ocicore.UpdateDrgRequest) (ocicore.UpdateDrgResponse, error)
	deleteDrgFn func(ctx context.Context, req ocicore.DeleteDrgRequest) (ocicore.DeleteDrgResponse, error)
	// Security List
	createSecurityListFn func(ctx context.Context, req ocicore.CreateSecurityListRequest) (ocicore.CreateSecurityListResponse, error)
	getSecurityListFn    func(ctx context.Context, req ocicore.GetSecurityListRequest) (ocicore.GetSecurityListResponse, error)
	listSecurityListsFn  func(ctx context.Context, req ocicore.ListSecurityListsRequest) (ocicore.ListSecurityListsResponse, error)
	updateSecurityListFn func(ctx context.Context, req ocicore.UpdateSecurityListRequest) (ocicore.UpdateSecurityListResponse, error)
	deleteSecurityListFn func(ctx context.Context, req ocicore.DeleteSecurityListRequest) (ocicore.DeleteSecurityListResponse, error)
	// Network Security Group
	createNetworkSecurityGroupFn func(ctx context.Context, req ocicore.CreateNetworkSecurityGroupRequest) (ocicore.CreateNetworkSecurityGroupResponse, error)
	getNetworkSecurityGroupFn    func(ctx context.Context, req ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error)
	listNetworkSecurityGroupsFn  func(ctx context.Context, req ocicore.ListNetworkSecurityGroupsRequest) (ocicore.ListNetworkSecurityGroupsResponse, error)
	updateNetworkSecurityGroupFn func(ctx context.Context, req ocicore.UpdateNetworkSecurityGroupRequest) (ocicore.UpdateNetworkSecurityGroupResponse, error)
	deleteNetworkSecurityGroupFn func(ctx context.Context, req ocicore.DeleteNetworkSecurityGroupRequest) (ocicore.DeleteNetworkSecurityGroupResponse, error)
	// Route Table
	createRouteTableFn func(ctx context.Context, req ocicore.CreateRouteTableRequest) (ocicore.CreateRouteTableResponse, error)
	getRouteTableFn    func(ctx context.Context, req ocicore.GetRouteTableRequest) (ocicore.GetRouteTableResponse, error)
	listRouteTablesFn  func(ctx context.Context, req ocicore.ListRouteTablesRequest) (ocicore.ListRouteTablesResponse, error)
	updateRouteTableFn func(ctx context.Context, req ocicore.UpdateRouteTableRequest) (ocicore.UpdateRouteTableResponse, error)
	deleteRouteTableFn func(ctx context.Context, req ocicore.DeleteRouteTableRequest) (ocicore.DeleteRouteTableResponse, error)
}

func (f *fakeVirtualNetworkClient) CreateVcn(ctx context.Context, req ocicore.CreateVcnRequest) (ocicore.CreateVcnResponse, error) {
	if f.createVcnFn != nil {
		return f.createVcnFn(ctx, req)
	}
	return ocicore.CreateVcnResponse{Vcn: ocicore.Vcn{Id: common.String("ocid1.vcn.oc1..new"), LifecycleState: ocicore.VcnLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetVcn(ctx context.Context, req ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
	if f.getVcnFn != nil {
		return f.getVcnFn(ctx, req)
	}
	return ocicore.GetVcnResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListVcns(ctx context.Context, req ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
	if f.listVcnsFn != nil {
		return f.listVcnsFn(ctx, req)
	}
	return ocicore.ListVcnsResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateVcn(ctx context.Context, req ocicore.UpdateVcnRequest) (ocicore.UpdateVcnResponse, error) {
	if f.updateVcnFn != nil {
		return f.updateVcnFn(ctx, req)
	}
	return ocicore.UpdateVcnResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteVcn(ctx context.Context, req ocicore.DeleteVcnRequest) (ocicore.DeleteVcnResponse, error) {
	if f.deleteVcnFn != nil {
		return f.deleteVcnFn(ctx, req)
	}
	return ocicore.DeleteVcnResponse{}, nil
}

func (f *fakeVirtualNetworkClient) CreateSubnet(ctx context.Context, req ocicore.CreateSubnetRequest) (ocicore.CreateSubnetResponse, error) {
	if f.createSubnetFn != nil {
		return f.createSubnetFn(ctx, req)
	}
	return ocicore.CreateSubnetResponse{Subnet: ocicore.Subnet{Id: common.String("ocid1.subnet.oc1..new"), LifecycleState: ocicore.SubnetLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetSubnet(ctx context.Context, req ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
	if f.getSubnetFn != nil {
		return f.getSubnetFn(ctx, req)
	}
	return ocicore.GetSubnetResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListSubnets(ctx context.Context, req ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
	if f.listSubnetsFn != nil {
		return f.listSubnetsFn(ctx, req)
	}
	return ocicore.ListSubnetsResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateSubnet(ctx context.Context, req ocicore.UpdateSubnetRequest) (ocicore.UpdateSubnetResponse, error) {
	if f.updateSubnetFn != nil {
		return f.updateSubnetFn(ctx, req)
	}
	return ocicore.UpdateSubnetResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteSubnet(ctx context.Context, req ocicore.DeleteSubnetRequest) (ocicore.DeleteSubnetResponse, error) {
	if f.deleteSubnetFn != nil {
		return f.deleteSubnetFn(ctx, req)
	}
	return ocicore.DeleteSubnetResponse{}, nil
}

// Internet Gateway stubs

func (f *fakeVirtualNetworkClient) CreateInternetGateway(ctx context.Context, req ocicore.CreateInternetGatewayRequest) (ocicore.CreateInternetGatewayResponse, error) {
	if f.createInternetGatewayFn != nil {
		return f.createInternetGatewayFn(ctx, req)
	}
	return ocicore.CreateInternetGatewayResponse{InternetGateway: ocicore.InternetGateway{Id: common.String("ocid1.internetgateway.oc1..new"), LifecycleState: ocicore.InternetGatewayLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetInternetGateway(ctx context.Context, req ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error) {
	if f.getInternetGatewayFn != nil {
		return f.getInternetGatewayFn(ctx, req)
	}
	return ocicore.GetInternetGatewayResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListInternetGateways(ctx context.Context, req ocicore.ListInternetGatewaysRequest) (ocicore.ListInternetGatewaysResponse, error) {
	if f.listInternetGatewaysFn != nil {
		return f.listInternetGatewaysFn(ctx, req)
	}
	return ocicore.ListInternetGatewaysResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateInternetGateway(ctx context.Context, req ocicore.UpdateInternetGatewayRequest) (ocicore.UpdateInternetGatewayResponse, error) {
	if f.updateInternetGatewayFn != nil {
		return f.updateInternetGatewayFn(ctx, req)
	}
	return ocicore.UpdateInternetGatewayResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteInternetGateway(ctx context.Context, req ocicore.DeleteInternetGatewayRequest) (ocicore.DeleteInternetGatewayResponse, error) {
	if f.deleteInternetGatewayFn != nil {
		return f.deleteInternetGatewayFn(ctx, req)
	}
	return ocicore.DeleteInternetGatewayResponse{}, nil
}

// NAT Gateway stubs

func (f *fakeVirtualNetworkClient) CreateNatGateway(ctx context.Context, req ocicore.CreateNatGatewayRequest) (ocicore.CreateNatGatewayResponse, error) {
	if f.createNatGatewayFn != nil {
		return f.createNatGatewayFn(ctx, req)
	}
	return ocicore.CreateNatGatewayResponse{NatGateway: ocicore.NatGateway{Id: common.String("ocid1.natgateway.oc1..new"), LifecycleState: ocicore.NatGatewayLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetNatGateway(ctx context.Context, req ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error) {
	if f.getNatGatewayFn != nil {
		return f.getNatGatewayFn(ctx, req)
	}
	return ocicore.GetNatGatewayResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListNatGateways(ctx context.Context, req ocicore.ListNatGatewaysRequest) (ocicore.ListNatGatewaysResponse, error) {
	if f.listNatGatewaysFn != nil {
		return f.listNatGatewaysFn(ctx, req)
	}
	return ocicore.ListNatGatewaysResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateNatGateway(ctx context.Context, req ocicore.UpdateNatGatewayRequest) (ocicore.UpdateNatGatewayResponse, error) {
	if f.updateNatGatewayFn != nil {
		return f.updateNatGatewayFn(ctx, req)
	}
	return ocicore.UpdateNatGatewayResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteNatGateway(ctx context.Context, req ocicore.DeleteNatGatewayRequest) (ocicore.DeleteNatGatewayResponse, error) {
	if f.deleteNatGatewayFn != nil {
		return f.deleteNatGatewayFn(ctx, req)
	}
	return ocicore.DeleteNatGatewayResponse{}, nil
}

// Service Gateway stubs

func (f *fakeVirtualNetworkClient) CreateServiceGateway(ctx context.Context, req ocicore.CreateServiceGatewayRequest) (ocicore.CreateServiceGatewayResponse, error) {
	if f.createServiceGatewayFn != nil {
		return f.createServiceGatewayFn(ctx, req)
	}
	return ocicore.CreateServiceGatewayResponse{ServiceGateway: ocicore.ServiceGateway{Id: common.String("ocid1.servicegateway.oc1..new"), LifecycleState: ocicore.ServiceGatewayLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetServiceGateway(ctx context.Context, req ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error) {
	if f.getServiceGatewayFn != nil {
		return f.getServiceGatewayFn(ctx, req)
	}
	return ocicore.GetServiceGatewayResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListServiceGateways(ctx context.Context, req ocicore.ListServiceGatewaysRequest) (ocicore.ListServiceGatewaysResponse, error) {
	if f.listServiceGatewaysFn != nil {
		return f.listServiceGatewaysFn(ctx, req)
	}
	return ocicore.ListServiceGatewaysResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateServiceGateway(ctx context.Context, req ocicore.UpdateServiceGatewayRequest) (ocicore.UpdateServiceGatewayResponse, error) {
	if f.updateServiceGatewayFn != nil {
		return f.updateServiceGatewayFn(ctx, req)
	}
	return ocicore.UpdateServiceGatewayResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteServiceGateway(ctx context.Context, req ocicore.DeleteServiceGatewayRequest) (ocicore.DeleteServiceGatewayResponse, error) {
	if f.deleteServiceGatewayFn != nil {
		return f.deleteServiceGatewayFn(ctx, req)
	}
	return ocicore.DeleteServiceGatewayResponse{}, nil
}

// DRG stubs

func (f *fakeVirtualNetworkClient) CreateDrg(ctx context.Context, req ocicore.CreateDrgRequest) (ocicore.CreateDrgResponse, error) {
	if f.createDrgFn != nil {
		return f.createDrgFn(ctx, req)
	}
	return ocicore.CreateDrgResponse{Drg: ocicore.Drg{Id: common.String("ocid1.drg.oc1..new"), LifecycleState: ocicore.DrgLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetDrg(ctx context.Context, req ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error) {
	if f.getDrgFn != nil {
		return f.getDrgFn(ctx, req)
	}
	return ocicore.GetDrgResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListDrgs(ctx context.Context, req ocicore.ListDrgsRequest) (ocicore.ListDrgsResponse, error) {
	if f.listDrgsFn != nil {
		return f.listDrgsFn(ctx, req)
	}
	return ocicore.ListDrgsResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateDrg(ctx context.Context, req ocicore.UpdateDrgRequest) (ocicore.UpdateDrgResponse, error) {
	if f.updateDrgFn != nil {
		return f.updateDrgFn(ctx, req)
	}
	return ocicore.UpdateDrgResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteDrg(ctx context.Context, req ocicore.DeleteDrgRequest) (ocicore.DeleteDrgResponse, error) {
	if f.deleteDrgFn != nil {
		return f.deleteDrgFn(ctx, req)
	}
	return ocicore.DeleteDrgResponse{}, nil
}

// Security List stubs

func (f *fakeVirtualNetworkClient) CreateSecurityList(ctx context.Context, req ocicore.CreateSecurityListRequest) (ocicore.CreateSecurityListResponse, error) {
	if f.createSecurityListFn != nil {
		return f.createSecurityListFn(ctx, req)
	}
	return ocicore.CreateSecurityListResponse{SecurityList: ocicore.SecurityList{Id: common.String("ocid1.securitylist.oc1..new"), LifecycleState: ocicore.SecurityListLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetSecurityList(ctx context.Context, req ocicore.GetSecurityListRequest) (ocicore.GetSecurityListResponse, error) {
	if f.getSecurityListFn != nil {
		return f.getSecurityListFn(ctx, req)
	}
	return ocicore.GetSecurityListResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListSecurityLists(ctx context.Context, req ocicore.ListSecurityListsRequest) (ocicore.ListSecurityListsResponse, error) {
	if f.listSecurityListsFn != nil {
		return f.listSecurityListsFn(ctx, req)
	}
	return ocicore.ListSecurityListsResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateSecurityList(ctx context.Context, req ocicore.UpdateSecurityListRequest) (ocicore.UpdateSecurityListResponse, error) {
	if f.updateSecurityListFn != nil {
		return f.updateSecurityListFn(ctx, req)
	}
	return ocicore.UpdateSecurityListResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteSecurityList(ctx context.Context, req ocicore.DeleteSecurityListRequest) (ocicore.DeleteSecurityListResponse, error) {
	if f.deleteSecurityListFn != nil {
		return f.deleteSecurityListFn(ctx, req)
	}
	return ocicore.DeleteSecurityListResponse{}, nil
}

// Network Security Group stubs

func (f *fakeVirtualNetworkClient) CreateNetworkSecurityGroup(ctx context.Context, req ocicore.CreateNetworkSecurityGroupRequest) (ocicore.CreateNetworkSecurityGroupResponse, error) {
	if f.createNetworkSecurityGroupFn != nil {
		return f.createNetworkSecurityGroupFn(ctx, req)
	}
	return ocicore.CreateNetworkSecurityGroupResponse{NetworkSecurityGroup: ocicore.NetworkSecurityGroup{Id: common.String("ocid1.networksecuritygroup.oc1..new"), LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetNetworkSecurityGroup(ctx context.Context, req ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error) {
	if f.getNetworkSecurityGroupFn != nil {
		return f.getNetworkSecurityGroupFn(ctx, req)
	}
	return ocicore.GetNetworkSecurityGroupResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListNetworkSecurityGroups(ctx context.Context, req ocicore.ListNetworkSecurityGroupsRequest) (ocicore.ListNetworkSecurityGroupsResponse, error) {
	if f.listNetworkSecurityGroupsFn != nil {
		return f.listNetworkSecurityGroupsFn(ctx, req)
	}
	return ocicore.ListNetworkSecurityGroupsResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateNetworkSecurityGroup(ctx context.Context, req ocicore.UpdateNetworkSecurityGroupRequest) (ocicore.UpdateNetworkSecurityGroupResponse, error) {
	if f.updateNetworkSecurityGroupFn != nil {
		return f.updateNetworkSecurityGroupFn(ctx, req)
	}
	return ocicore.UpdateNetworkSecurityGroupResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteNetworkSecurityGroup(ctx context.Context, req ocicore.DeleteNetworkSecurityGroupRequest) (ocicore.DeleteNetworkSecurityGroupResponse, error) {
	if f.deleteNetworkSecurityGroupFn != nil {
		return f.deleteNetworkSecurityGroupFn(ctx, req)
	}
	return ocicore.DeleteNetworkSecurityGroupResponse{}, nil
}

// Route Table stubs

func (f *fakeVirtualNetworkClient) CreateRouteTable(ctx context.Context, req ocicore.CreateRouteTableRequest) (ocicore.CreateRouteTableResponse, error) {
	if f.createRouteTableFn != nil {
		return f.createRouteTableFn(ctx, req)
	}
	return ocicore.CreateRouteTableResponse{RouteTable: ocicore.RouteTable{Id: common.String("ocid1.routetable.oc1..new"), LifecycleState: ocicore.RouteTableLifecycleStateAvailable}}, nil
}

func (f *fakeVirtualNetworkClient) GetRouteTable(ctx context.Context, req ocicore.GetRouteTableRequest) (ocicore.GetRouteTableResponse, error) {
	if f.getRouteTableFn != nil {
		return f.getRouteTableFn(ctx, req)
	}
	return ocicore.GetRouteTableResponse{}, nil
}

func (f *fakeVirtualNetworkClient) ListRouteTables(ctx context.Context, req ocicore.ListRouteTablesRequest) (ocicore.ListRouteTablesResponse, error) {
	if f.listRouteTablesFn != nil {
		return f.listRouteTablesFn(ctx, req)
	}
	return ocicore.ListRouteTablesResponse{}, nil
}

func (f *fakeVirtualNetworkClient) UpdateRouteTable(ctx context.Context, req ocicore.UpdateRouteTableRequest) (ocicore.UpdateRouteTableResponse, error) {
	if f.updateRouteTableFn != nil {
		return f.updateRouteTableFn(ctx, req)
	}
	return ocicore.UpdateRouteTableResponse{}, nil
}

func (f *fakeVirtualNetworkClient) DeleteRouteTable(ctx context.Context, req ocicore.DeleteRouteTableRequest) (ocicore.DeleteRouteTableResponse, error) {
	if f.deleteRouteTableFn != nil {
		return f.deleteRouteTableFn(ctx, req)
	}
	return ocicore.DeleteRouteTableResponse{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func defaultLog() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
}

func emptyProvider() common.ConfigurationProvider {
	return common.NewRawConfigurationProvider("", "", "", "", "", nil)
}

func vcnMgrWithFake(fake *fakeVirtualNetworkClient) *OciVcnServiceManager {
	mgr := NewOciVcnServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetVcnClientForTest(mgr, fake)
	return mgr
}

func subnetMgrWithFake(fake *fakeVirtualNetworkClient) *OciSubnetServiceManager {
	mgr := NewOciSubnetServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetSubnetClientForTest(mgr, fake)
	return mgr
}

func makeAvailableVcn(id, displayName string) ocicore.Vcn {
	return ocicore.Vcn{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState: ocicore.VcnLifecycleStateAvailable,
		CidrBlock:      common.String("10.0.0.0/16"),
	}
}

func makeAvailableSubnet(id, displayName, vcnId string) ocicore.Subnet {
	return ocicore.Subnet{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
		VcnId:          common.String(vcnId),
		LifecycleState: ocicore.SubnetLifecycleStateAvailable,
		CidrBlock:      common.String("10.0.1.0/24"),
	}
}

// ---------------------------------------------------------------------------
// VCN: GetCrdStatus
// ---------------------------------------------------------------------------

func TestVcn_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciVcnServiceManager(emptyProvider(), nil, nil, defaultLog())

	v := &ociv1beta1.OciVcn{}
	v.Status.OsokStatus.Ocid = "ocid1.vcn.oc1..xxx"

	status, err := mgr.GetCrdStatus(v)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.vcn.oc1..xxx"), status.Ocid)
}

func TestVcn_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciVcnServiceManager(emptyProvider(), nil, nil, defaultLog())

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// ---------------------------------------------------------------------------
// VCN: CreateOrUpdate — type assertion
// ---------------------------------------------------------------------------

func TestVcn_CreateOrUpdate_BadType(t *testing.T) {
	mgr := NewOciVcnServiceManager(emptyProvider(), nil, nil, defaultLog())

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// VCN: CreateOrUpdate — create when not exists
// ---------------------------------------------------------------------------

func TestVcn_CreateOrUpdate_NoId_NotFound_CreatesAndActive(t *testing.T) {
	vcnID := "ocid1.vcn.oc1..created"
	fake := &fakeVirtualNetworkClient{
		listVcnsFn: func(_ context.Context, _ ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
			return ocicore.ListVcnsResponse{Items: []ocicore.Vcn{}}, nil
		},
		createVcnFn: func(_ context.Context, _ ocicore.CreateVcnRequest) (ocicore.CreateVcnResponse, error) {
			return ocicore.CreateVcnResponse{
				Vcn: makeAvailableVcn(vcnID, "new-vcn"),
			}, nil
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Name = "new-vcn"
	v.Namespace = "default"
	v.Spec.DisplayName = "new-vcn"
	v.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	v.Spec.CidrBlock = "10.0.0.0/16"

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

// TestVcn_CreateOrUpdate_NoId_NotFound_Provisioning verifies that a newly-created
// VCN in PROVISIONING state triggers a requeue (IsSuccessful=false, no error).
func TestVcn_CreateOrUpdate_NoId_NotFound_Provisioning(t *testing.T) {
	vcnID := "ocid1.vcn.oc1..provisioning"
	fake := &fakeVirtualNetworkClient{
		listVcnsFn: func(_ context.Context, _ ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
			return ocicore.ListVcnsResponse{Items: []ocicore.Vcn{}}, nil
		},
		createVcnFn: func(_ context.Context, _ ocicore.CreateVcnRequest) (ocicore.CreateVcnResponse, error) {
			return ocicore.CreateVcnResponse{
				Vcn: ocicore.Vcn{
					Id:             common.String(vcnID),
					DisplayName:    common.String("provisioning-vcn"),
					LifecycleState: ocicore.VcnLifecycleStateProvisioning,
				},
			}, nil
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Name = "provisioning-vcn"
	v.Namespace = "default"
	v.Spec.DisplayName = "provisioning-vcn"
	v.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	v.Spec.CidrBlock = "10.0.0.0/16"

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "PROVISIONING VCN should cause requeue")
}

// ---------------------------------------------------------------------------
// VCN: CreateOrUpdate — bind by display name
// ---------------------------------------------------------------------------

func TestVcn_CreateOrUpdate_NoId_FoundByDisplayName_Active(t *testing.T) {
	vcnID := "ocid1.vcn.oc1..existing"
	fake := &fakeVirtualNetworkClient{
		listVcnsFn: func(_ context.Context, _ ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
			return ocicore.ListVcnsResponse{
				Items: []ocicore.Vcn{
					{Id: common.String(vcnID), LifecycleState: ocicore.VcnLifecycleStateAvailable},
				},
			}, nil
		},
		getVcnFn: func(_ context.Context, _ ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
			return ocicore.GetVcnResponse{Vcn: makeAvailableVcn(vcnID, "existing-vcn")}, nil
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Name = "existing-vcn"
	v.Namespace = "default"
	v.Spec.DisplayName = "existing-vcn"
	v.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	v.Spec.CidrBlock = "10.0.0.0/16"

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(vcnID), v.Status.OsokStatus.Ocid)
}

// TestVcn_CreateOrUpdate_NoId_FoundByDisplayName_Provisioning verifies that a
// found-but-PROVISIONING VCN triggers a requeue.
func TestVcn_CreateOrUpdate_NoId_FoundByDisplayName_Provisioning(t *testing.T) {
	vcnID := "ocid1.vcn.oc1..prov"
	fake := &fakeVirtualNetworkClient{
		listVcnsFn: func(_ context.Context, _ ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
			return ocicore.ListVcnsResponse{
				Items: []ocicore.Vcn{
					// GetVcnOcid accepts AVAILABLE/PROVISIONING/UPDATING
					{Id: common.String(vcnID), LifecycleState: ocicore.VcnLifecycleStateAvailable},
				},
			}, nil
		},
		getVcnFn: func(_ context.Context, _ ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
			return ocicore.GetVcnResponse{
				Vcn: ocicore.Vcn{
					Id:             common.String(vcnID),
					DisplayName:    common.String("prov-vcn"),
					LifecycleState: ocicore.VcnLifecycleStateProvisioning,
				},
			}, nil
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Spec.DisplayName = "prov-vcn"
	v.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	v.Spec.CidrBlock = "10.0.0.0/16"

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "PROVISIONING VCN found by display name should requeue")
}

// ---------------------------------------------------------------------------
// VCN: CreateOrUpdate — bind by VcnId
// ---------------------------------------------------------------------------

func TestVcn_CreateOrUpdate_WithId_Binds(t *testing.T) {
	vcnID := "ocid1.vcn.oc1..bind"
	fake := &fakeVirtualNetworkClient{
		getVcnFn: func(_ context.Context, _ ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
			return ocicore.GetVcnResponse{Vcn: makeAvailableVcn(vcnID, "bind-vcn")}, nil
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Name = "bind-vcn"
	v.Namespace = "default"
	v.Spec.VcnId = ociv1beta1.OCID(vcnID)
	v.Spec.DisplayName = "bind-vcn"
	v.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	v.Spec.CidrBlock = "10.0.0.0/16"
	v.Status.OsokStatus.Ocid = ociv1beta1.OCID(vcnID)

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// VCN: CreateOrUpdate — error propagation
// ---------------------------------------------------------------------------

func TestVcn_CreateOrUpdate_ListError(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		listVcnsFn: func(_ context.Context, _ ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
			return ocicore.ListVcnsResponse{}, errors.New("list failed")
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Spec.DisplayName = "err-vcn"
	v.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestVcn_CreateOrUpdate_CreateError(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		listVcnsFn: func(_ context.Context, _ ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
			return ocicore.ListVcnsResponse{Items: []ocicore.Vcn{}}, nil
		},
		createVcnFn: func(_ context.Context, _ ocicore.CreateVcnRequest) (ocicore.CreateVcnResponse, error) {
			return ocicore.CreateVcnResponse{}, errors.New("create failed")
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Spec.DisplayName = "fail-vcn"
	v.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	v.Spec.CidrBlock = "10.0.0.0/16"

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// VCN: Delete
// ---------------------------------------------------------------------------

func TestVcn_Delete_NoOcid(t *testing.T) {
	mgr := NewOciVcnServiceManager(emptyProvider(), nil, nil, defaultLog())

	v := &ociv1beta1.OciVcn{}
	v.Name = "no-ocid-vcn"
	v.Namespace = "default"

	done, err := mgr.Delete(context.Background(), v)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestVcn_Delete_WithFakeClient(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteVcnFn: func(_ context.Context, _ ocicore.DeleteVcnRequest) (ocicore.DeleteVcnResponse, error) {
			deleteCalled = true
			return ocicore.DeleteVcnResponse{}, nil
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Name = "del-vcn"
	v.Namespace = "default"
	v.Status.OsokStatus.Ocid = "ocid1.vcn.oc1..del"

	done, err := mgr.Delete(context.Background(), v)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

func TestVcn_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteVcnFn: func(_ context.Context, _ ocicore.DeleteVcnRequest) (ocicore.DeleteVcnResponse, error) {
			return ocicore.DeleteVcnResponse{}, errors.New("delete failed")
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Status.OsokStatus.Ocid = "ocid1.vcn.oc1..del"

	done, err := mgr.Delete(context.Background(), v)
	assert.Error(t, err)
	assert.False(t, done)
}

// ---------------------------------------------------------------------------
// Subnet: GetCrdStatus
// ---------------------------------------------------------------------------

func TestSubnet_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciSubnetServiceManager(emptyProvider(), nil, nil, defaultLog())

	s := &ociv1beta1.OciSubnet{}
	s.Status.OsokStatus.Ocid = "ocid1.subnet.oc1..xxx"

	status, err := mgr.GetCrdStatus(s)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.subnet.oc1..xxx"), status.Ocid)
}

func TestSubnet_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciSubnetServiceManager(emptyProvider(), nil, nil, defaultLog())

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// ---------------------------------------------------------------------------
// Subnet: CreateOrUpdate — type assertion
// ---------------------------------------------------------------------------

func TestSubnet_CreateOrUpdate_BadType(t *testing.T) {
	mgr := NewOciSubnetServiceManager(emptyProvider(), nil, nil, defaultLog())

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// Subnet: CreateOrUpdate — create with VcnId
// ---------------------------------------------------------------------------

func TestSubnet_CreateOrUpdate_NoId_NotFound_CreatesWithVcnId(t *testing.T) {
	subnetID := "ocid1.subnet.oc1..created"
	vcnID := "ocid1.vcn.oc1..parent"

	var capturedReq ocicore.CreateSubnetRequest
	fake := &fakeVirtualNetworkClient{
		listSubnetsFn: func(_ context.Context, _ ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
			return ocicore.ListSubnetsResponse{Items: []ocicore.Subnet{}}, nil
		},
		createSubnetFn: func(_ context.Context, req ocicore.CreateSubnetRequest) (ocicore.CreateSubnetResponse, error) {
			capturedReq = req
			return ocicore.CreateSubnetResponse{
				Subnet: makeAvailableSubnet(subnetID, "new-subnet", vcnID),
			}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Name = "new-subnet"
	s.Namespace = "default"
	s.Spec.DisplayName = "new-subnet"
	s.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	s.Spec.VcnId = ociv1beta1.OCID(vcnID)
	s.Spec.CidrBlock = "10.0.1.0/24"

	resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, vcnID, *capturedReq.CreateSubnetDetails.VcnId, "VcnId must be passed to OCI")
}

// TestSubnet_CreateOrUpdate_NoId_NotFound_Provisioning verifies newly-created PROVISIONING subnet
// triggers a requeue.
func TestSubnet_CreateOrUpdate_NoId_NotFound_Provisioning(t *testing.T) {
	subnetID := "ocid1.subnet.oc1..prov"
	fake := &fakeVirtualNetworkClient{
		listSubnetsFn: func(_ context.Context, _ ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
			return ocicore.ListSubnetsResponse{Items: []ocicore.Subnet{}}, nil
		},
		createSubnetFn: func(_ context.Context, _ ocicore.CreateSubnetRequest) (ocicore.CreateSubnetResponse, error) {
			return ocicore.CreateSubnetResponse{
				Subnet: ocicore.Subnet{
					Id:             common.String(subnetID),
					DisplayName:    common.String("prov-subnet"),
					LifecycleState: ocicore.SubnetLifecycleStateProvisioning,
				},
			}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Spec.DisplayName = "prov-subnet"
	s.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	s.Spec.VcnId = "ocid1.vcn.oc1..parent"
	s.Spec.CidrBlock = "10.0.1.0/24"

	resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "PROVISIONING subnet should cause requeue")
}

// ---------------------------------------------------------------------------
// Subnet: CreateOrUpdate — AVAILABLE state success
// ---------------------------------------------------------------------------

func TestSubnet_CreateOrUpdate_NoId_FoundByDisplayName_Available(t *testing.T) {
	subnetID := "ocid1.subnet.oc1..existing"
	vcnID := "ocid1.vcn.oc1..parent"
	fake := &fakeVirtualNetworkClient{
		listSubnetsFn: func(_ context.Context, _ ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
			return ocicore.ListSubnetsResponse{
				Items: []ocicore.Subnet{
					{Id: common.String(subnetID), LifecycleState: ocicore.SubnetLifecycleStateAvailable},
				},
			}, nil
		},
		getSubnetFn: func(_ context.Context, _ ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
			return ocicore.GetSubnetResponse{Subnet: makeAvailableSubnet(subnetID, "existing-subnet", vcnID)}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Name = "existing-subnet"
	s.Namespace = "default"
	s.Spec.DisplayName = "existing-subnet"
	s.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	s.Spec.VcnId = ociv1beta1.OCID(vcnID)
	s.Spec.CidrBlock = "10.0.1.0/24"

	resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(subnetID), s.Status.OsokStatus.Ocid)
}

// TestSubnet_CreateOrUpdate_NoId_FoundByDisplayName_Provisioning verifies a found-but-PROVISIONING
// subnet triggers a requeue.
func TestSubnet_CreateOrUpdate_NoId_FoundByDisplayName_Provisioning(t *testing.T) {
	subnetID := "ocid1.subnet.oc1..provfound"
	vcnID := "ocid1.vcn.oc1..parent"
	fake := &fakeVirtualNetworkClient{
		listSubnetsFn: func(_ context.Context, _ ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
			return ocicore.ListSubnetsResponse{
				Items: []ocicore.Subnet{
					// GetSubnetOcid accepts AVAILABLE/PROVISIONING/UPDATING
					{Id: common.String(subnetID), LifecycleState: ocicore.SubnetLifecycleStateAvailable},
				},
			}, nil
		},
		getSubnetFn: func(_ context.Context, _ ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
			return ocicore.GetSubnetResponse{
				Subnet: ocicore.Subnet{
					Id:             common.String(subnetID),
					DisplayName:    common.String("prov-found-subnet"),
					LifecycleState: ocicore.SubnetLifecycleStateProvisioning,
				},
			}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Spec.DisplayName = "prov-found-subnet"
	s.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	s.Spec.VcnId = ociv1beta1.OCID(vcnID)
	s.Spec.CidrBlock = "10.0.1.0/24"

	resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "PROVISIONING subnet found by display name should requeue")
}

// ---------------------------------------------------------------------------
// Subnet: CreateOrUpdate — bind by SubnetId
// ---------------------------------------------------------------------------

func TestSubnet_CreateOrUpdate_WithId_Binds(t *testing.T) {
	subnetID := "ocid1.subnet.oc1..bind"
	vcnID := "ocid1.vcn.oc1..parent"
	fake := &fakeVirtualNetworkClient{
		getSubnetFn: func(_ context.Context, _ ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
			return ocicore.GetSubnetResponse{Subnet: makeAvailableSubnet(subnetID, "bind-subnet", vcnID)}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Name = "bind-subnet"
	s.Namespace = "default"
	s.Spec.SubnetId = ociv1beta1.OCID(subnetID)
	s.Spec.DisplayName = "bind-subnet"
	s.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	s.Spec.VcnId = ociv1beta1.OCID(vcnID)
	s.Spec.CidrBlock = "10.0.1.0/24"
	s.Status.OsokStatus.Ocid = ociv1beta1.OCID(subnetID)

	resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// Subnet: CreateOrUpdate — error propagation
// ---------------------------------------------------------------------------

func TestSubnet_CreateOrUpdate_ListError(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		listSubnetsFn: func(_ context.Context, _ ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
			return ocicore.ListSubnetsResponse{}, errors.New("list failed")
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Spec.DisplayName = "err-subnet"
	s.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	s.Spec.VcnId = "ocid1.vcn.oc1..parent"

	resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestSubnet_CreateOrUpdate_CreateError(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		listSubnetsFn: func(_ context.Context, _ ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
			return ocicore.ListSubnetsResponse{Items: []ocicore.Subnet{}}, nil
		},
		createSubnetFn: func(_ context.Context, _ ocicore.CreateSubnetRequest) (ocicore.CreateSubnetResponse, error) {
			return ocicore.CreateSubnetResponse{}, errors.New("create failed")
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Spec.DisplayName = "fail-subnet"
	s.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	s.Spec.VcnId = "ocid1.vcn.oc1..parent"
	s.Spec.CidrBlock = "10.0.1.0/24"

	resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// Subnet: Delete
// ---------------------------------------------------------------------------

func TestSubnet_Delete_NoOcid(t *testing.T) {
	mgr := NewOciSubnetServiceManager(emptyProvider(), nil, nil, defaultLog())

	s := &ociv1beta1.OciSubnet{}
	s.Name = "no-ocid-subnet"
	s.Namespace = "default"

	done, err := mgr.Delete(context.Background(), s)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestSubnet_Delete_WithFakeClient(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteSubnetFn: func(_ context.Context, _ ocicore.DeleteSubnetRequest) (ocicore.DeleteSubnetResponse, error) {
			deleteCalled = true
			return ocicore.DeleteSubnetResponse{}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Name = "del-subnet"
	s.Namespace = "default"
	s.Status.OsokStatus.Ocid = "ocid1.subnet.oc1..del"

	done, err := mgr.Delete(context.Background(), s)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

func TestSubnet_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteSubnetFn: func(_ context.Context, _ ocicore.DeleteSubnetRequest) (ocicore.DeleteSubnetResponse, error) {
			return ocicore.DeleteSubnetResponse{}, errors.New("delete failed")
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Status.OsokStatus.Ocid = "ocid1.subnet.oc1..del"

	done, err := mgr.Delete(context.Background(), s)
	assert.Error(t, err)
	assert.False(t, done)
}

// ---------------------------------------------------------------------------
// Helper factories for new gateway managers
// ---------------------------------------------------------------------------

func igwMgrWithFake(fake *fakeVirtualNetworkClient) *OciInternetGatewayServiceManager {
	mgr := NewOciInternetGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetInternetGatewayClientForTest(mgr, fake)
	return mgr
}

func natMgrWithFake(fake *fakeVirtualNetworkClient) *OciNatGatewayServiceManager {
	mgr := NewOciNatGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetNatGatewayClientForTest(mgr, fake)
	return mgr
}

func sgwMgrWithFake(fake *fakeVirtualNetworkClient) *OciServiceGatewayServiceManager {
	mgr := NewOciServiceGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetServiceGatewayClientForTest(mgr, fake)
	return mgr
}

func drgMgrWithFake(fake *fakeVirtualNetworkClient) *OciDrgServiceManager {
	mgr := NewOciDrgServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetDrgClientForTest(mgr, fake)
	return mgr
}

// ---------------------------------------------------------------------------
// InternetGateway tests
// ---------------------------------------------------------------------------

func TestInternetGateway_CreateOrUpdate_CreatesNew(t *testing.T) {
	igwID := "ocid1.internetgateway.oc1..created"
	fake := &fakeVirtualNetworkClient{
		listInternetGatewaysFn: func(_ context.Context, _ ocicore.ListInternetGatewaysRequest) (ocicore.ListInternetGatewaysResponse, error) {
			return ocicore.ListInternetGatewaysResponse{Items: []ocicore.InternetGateway{}}, nil
		},
		createInternetGatewayFn: func(_ context.Context, _ ocicore.CreateInternetGatewayRequest) (ocicore.CreateInternetGatewayResponse, error) {
			return ocicore.CreateInternetGatewayResponse{
				InternetGateway: ocicore.InternetGateway{
					Id:             common.String(igwID),
					DisplayName:    common.String("new-igw"),
					LifecycleState: ocicore.InternetGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := igwMgrWithFake(fake)

	igw := &ociv1beta1.OciInternetGateway{}
	igw.Name = "new-igw"
	igw.Namespace = "default"
	igw.Spec.DisplayName = "new-igw"
	igw.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	igw.Spec.VcnId = "ocid1.vcn.oc1..parent"
	igw.Spec.IsEnabled = true

	resp, err := mgr.CreateOrUpdate(context.Background(), igw, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(igwID), igw.Status.OsokStatus.Ocid)
}

func TestInternetGateway_CreateOrUpdate_FindsExisting(t *testing.T) {
	igwID := "ocid1.internetgateway.oc1..existing"
	fake := &fakeVirtualNetworkClient{
		listInternetGatewaysFn: func(_ context.Context, _ ocicore.ListInternetGatewaysRequest) (ocicore.ListInternetGatewaysResponse, error) {
			return ocicore.ListInternetGatewaysResponse{
				Items: []ocicore.InternetGateway{
					{Id: common.String(igwID), DisplayName: common.String("existing-igw"), LifecycleState: ocicore.InternetGatewayLifecycleStateAvailable},
				},
			}, nil
		},
		getInternetGatewayFn: func(_ context.Context, _ ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error) {
			return ocicore.GetInternetGatewayResponse{
				InternetGateway: ocicore.InternetGateway{
					Id:             common.String(igwID),
					DisplayName:    common.String("existing-igw"),
					LifecycleState: ocicore.InternetGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := igwMgrWithFake(fake)

	igw := &ociv1beta1.OciInternetGateway{}
	igw.Spec.DisplayName = "existing-igw"
	igw.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	igw.Spec.VcnId = "ocid1.vcn.oc1..parent"

	resp, err := mgr.CreateOrUpdate(context.Background(), igw, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(igwID), igw.Status.OsokStatus.Ocid)
}

func TestInternetGateway_Delete_Succeeds(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteInternetGatewayFn: func(_ context.Context, _ ocicore.DeleteInternetGatewayRequest) (ocicore.DeleteInternetGatewayResponse, error) {
			deleteCalled = true
			return ocicore.DeleteInternetGatewayResponse{}, nil
		},
	}
	mgr := igwMgrWithFake(fake)

	igw := &ociv1beta1.OciInternetGateway{}
	igw.Status.OsokStatus.Ocid = "ocid1.internetgateway.oc1..del"

	done, err := mgr.Delete(context.Background(), igw)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// ---------------------------------------------------------------------------
// NatGateway tests
// ---------------------------------------------------------------------------

func TestNatGateway_CreateOrUpdate_CreatesNew(t *testing.T) {
	natID := "ocid1.natgateway.oc1..created"
	fake := &fakeVirtualNetworkClient{
		listNatGatewaysFn: func(_ context.Context, _ ocicore.ListNatGatewaysRequest) (ocicore.ListNatGatewaysResponse, error) {
			return ocicore.ListNatGatewaysResponse{Items: []ocicore.NatGateway{}}, nil
		},
		createNatGatewayFn: func(_ context.Context, _ ocicore.CreateNatGatewayRequest) (ocicore.CreateNatGatewayResponse, error) {
			return ocicore.CreateNatGatewayResponse{
				NatGateway: ocicore.NatGateway{
					Id:             common.String(natID),
					DisplayName:    common.String("new-nat"),
					LifecycleState: ocicore.NatGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := natMgrWithFake(fake)

	nat := &ociv1beta1.OciNatGateway{}
	nat.Name = "new-nat"
	nat.Namespace = "default"
	nat.Spec.DisplayName = "new-nat"
	nat.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	nat.Spec.VcnId = "ocid1.vcn.oc1..parent"

	resp, err := mgr.CreateOrUpdate(context.Background(), nat, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(natID), nat.Status.OsokStatus.Ocid)
}

func TestNatGateway_CreateOrUpdate_FindsExisting(t *testing.T) {
	natID := "ocid1.natgateway.oc1..existing"
	fake := &fakeVirtualNetworkClient{
		listNatGatewaysFn: func(_ context.Context, _ ocicore.ListNatGatewaysRequest) (ocicore.ListNatGatewaysResponse, error) {
			return ocicore.ListNatGatewaysResponse{
				Items: []ocicore.NatGateway{
					{Id: common.String(natID), DisplayName: common.String("existing-nat"), LifecycleState: ocicore.NatGatewayLifecycleStateAvailable},
				},
			}, nil
		},
		getNatGatewayFn: func(_ context.Context, _ ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error) {
			return ocicore.GetNatGatewayResponse{
				NatGateway: ocicore.NatGateway{
					Id:             common.String(natID),
					DisplayName:    common.String("existing-nat"),
					LifecycleState: ocicore.NatGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := natMgrWithFake(fake)

	nat := &ociv1beta1.OciNatGateway{}
	nat.Spec.DisplayName = "existing-nat"
	nat.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	nat.Spec.VcnId = "ocid1.vcn.oc1..parent"

	resp, err := mgr.CreateOrUpdate(context.Background(), nat, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(natID), nat.Status.OsokStatus.Ocid)
}

func TestNatGateway_Delete_Succeeds(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteNatGatewayFn: func(_ context.Context, _ ocicore.DeleteNatGatewayRequest) (ocicore.DeleteNatGatewayResponse, error) {
			deleteCalled = true
			return ocicore.DeleteNatGatewayResponse{}, nil
		},
	}
	mgr := natMgrWithFake(fake)

	nat := &ociv1beta1.OciNatGateway{}
	nat.Status.OsokStatus.Ocid = "ocid1.natgateway.oc1..del"

	done, err := mgr.Delete(context.Background(), nat)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// ---------------------------------------------------------------------------
// ServiceGateway tests
// ---------------------------------------------------------------------------

func TestServiceGateway_CreateOrUpdate_CreatesNew(t *testing.T) {
	sgwID := "ocid1.servicegateway.oc1..created"
	fake := &fakeVirtualNetworkClient{
		listServiceGatewaysFn: func(_ context.Context, _ ocicore.ListServiceGatewaysRequest) (ocicore.ListServiceGatewaysResponse, error) {
			return ocicore.ListServiceGatewaysResponse{Items: []ocicore.ServiceGateway{}}, nil
		},
		createServiceGatewayFn: func(_ context.Context, req ocicore.CreateServiceGatewayRequest) (ocicore.CreateServiceGatewayResponse, error) {
			return ocicore.CreateServiceGatewayResponse{
				ServiceGateway: ocicore.ServiceGateway{
					Id:             common.String(sgwID),
					DisplayName:    common.String("new-sgw"),
					LifecycleState: ocicore.ServiceGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := sgwMgrWithFake(fake)

	sgw := &ociv1beta1.OciServiceGateway{}
	sgw.Name = "new-sgw"
	sgw.Namespace = "default"
	sgw.Spec.DisplayName = "new-sgw"
	sgw.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	sgw.Spec.VcnId = "ocid1.vcn.oc1..parent"
	sgw.Spec.Services = []string{"ocid1.service.oc1..svc"}

	resp, err := mgr.CreateOrUpdate(context.Background(), sgw, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(sgwID), sgw.Status.OsokStatus.Ocid)
}

func TestServiceGateway_CreateOrUpdate_FindsExisting(t *testing.T) {
	sgwID := "ocid1.servicegateway.oc1..existing"
	fake := &fakeVirtualNetworkClient{
		listServiceGatewaysFn: func(_ context.Context, _ ocicore.ListServiceGatewaysRequest) (ocicore.ListServiceGatewaysResponse, error) {
			return ocicore.ListServiceGatewaysResponse{
				Items: []ocicore.ServiceGateway{
					{Id: common.String(sgwID), DisplayName: common.String("existing-sgw"), LifecycleState: ocicore.ServiceGatewayLifecycleStateAvailable},
				},
			}, nil
		},
		getServiceGatewayFn: func(_ context.Context, _ ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error) {
			return ocicore.GetServiceGatewayResponse{
				ServiceGateway: ocicore.ServiceGateway{
					Id:             common.String(sgwID),
					DisplayName:    common.String("existing-sgw"),
					LifecycleState: ocicore.ServiceGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := sgwMgrWithFake(fake)

	sgw := &ociv1beta1.OciServiceGateway{}
	sgw.Spec.DisplayName = "existing-sgw"
	sgw.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	sgw.Spec.VcnId = "ocid1.vcn.oc1..parent"
	sgw.Spec.Services = []string{"ocid1.service.oc1..svc"}

	resp, err := mgr.CreateOrUpdate(context.Background(), sgw, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(sgwID), sgw.Status.OsokStatus.Ocid)
}

func TestServiceGateway_Delete_Succeeds(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteServiceGatewayFn: func(_ context.Context, _ ocicore.DeleteServiceGatewayRequest) (ocicore.DeleteServiceGatewayResponse, error) {
			deleteCalled = true
			return ocicore.DeleteServiceGatewayResponse{}, nil
		},
	}
	mgr := sgwMgrWithFake(fake)

	sgw := &ociv1beta1.OciServiceGateway{}
	sgw.Status.OsokStatus.Ocid = "ocid1.servicegateway.oc1..del"

	done, err := mgr.Delete(context.Background(), sgw)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// ---------------------------------------------------------------------------
// DRG tests
// ---------------------------------------------------------------------------

func TestDrg_CreateOrUpdate_CreatesNew(t *testing.T) {
	drgID := "ocid1.drg.oc1..created"
	fake := &fakeVirtualNetworkClient{
		listDrgsFn: func(_ context.Context, _ ocicore.ListDrgsRequest) (ocicore.ListDrgsResponse, error) {
			return ocicore.ListDrgsResponse{Items: []ocicore.Drg{}}, nil
		},
		createDrgFn: func(_ context.Context, _ ocicore.CreateDrgRequest) (ocicore.CreateDrgResponse, error) {
			return ocicore.CreateDrgResponse{
				Drg: ocicore.Drg{
					Id:             common.String(drgID),
					DisplayName:    common.String("new-drg"),
					LifecycleState: ocicore.DrgLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := drgMgrWithFake(fake)

	drg := &ociv1beta1.OciDrg{}
	drg.Name = "new-drg"
	drg.Namespace = "default"
	drg.Spec.DisplayName = "new-drg"
	drg.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), drg, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(drgID), drg.Status.OsokStatus.Ocid)
}

func TestDrg_CreateOrUpdate_FindsExisting(t *testing.T) {
	drgID := "ocid1.drg.oc1..existing"
	fake := &fakeVirtualNetworkClient{
		listDrgsFn: func(_ context.Context, _ ocicore.ListDrgsRequest) (ocicore.ListDrgsResponse, error) {
			return ocicore.ListDrgsResponse{
				Items: []ocicore.Drg{
					{Id: common.String(drgID), DisplayName: common.String("existing-drg"), LifecycleState: ocicore.DrgLifecycleStateAvailable},
				},
			}, nil
		},
		getDrgFn: func(_ context.Context, _ ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error) {
			return ocicore.GetDrgResponse{
				Drg: ocicore.Drg{
					Id:             common.String(drgID),
					DisplayName:    common.String("existing-drg"),
					LifecycleState: ocicore.DrgLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := drgMgrWithFake(fake)

	drg := &ociv1beta1.OciDrg{}
	drg.Spec.DisplayName = "existing-drg"
	drg.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), drg, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(drgID), drg.Status.OsokStatus.Ocid)
}

func TestDrg_Delete_Succeeds(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteDrgFn: func(_ context.Context, _ ocicore.DeleteDrgRequest) (ocicore.DeleteDrgResponse, error) {
			deleteCalled = true
			return ocicore.DeleteDrgResponse{}, nil
		},
	}
	mgr := drgMgrWithFake(fake)

	drg := &ociv1beta1.OciDrg{}
	drg.Status.OsokStatus.Ocid = "ocid1.drg.oc1..del"

	done, err := mgr.Delete(context.Background(), drg)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// ---------------------------------------------------------------------------
// Helper constructors for new service managers
// ---------------------------------------------------------------------------

func securityListMgrWithFake(fake *fakeVirtualNetworkClient) *OciSecurityListServiceManager {
	mgr := NewOciSecurityListServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetSecurityListClientForTest(mgr, fake)
	return mgr
}

func nsgMgrWithFake(fake *fakeVirtualNetworkClient) *OciNetworkSecurityGroupServiceManager {
	mgr := NewOciNetworkSecurityGroupServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetNSGClientForTest(mgr, fake)
	return mgr
}

func routeTableMgrWithFake(fake *fakeVirtualNetworkClient) *OciRouteTableServiceManager {
	mgr := NewOciRouteTableServiceManager(emptyProvider(), nil, nil, defaultLog())
	ExportSetRouteTableClientForTest(mgr, fake)
	return mgr
}

// ---------------------------------------------------------------------------
// SecurityList tests
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_SecurityList_CreatesNew(t *testing.T) {
	slID := "ocid1.securitylist.oc1..created"
	fake := &fakeVirtualNetworkClient{
		listSecurityListsFn: func(_ context.Context, _ ocicore.ListSecurityListsRequest) (ocicore.ListSecurityListsResponse, error) {
			return ocicore.ListSecurityListsResponse{Items: []ocicore.SecurityList{}}, nil
		},
		createSecurityListFn: func(_ context.Context, _ ocicore.CreateSecurityListRequest) (ocicore.CreateSecurityListResponse, error) {
			return ocicore.CreateSecurityListResponse{
				SecurityList: ocicore.SecurityList{
					Id:             common.String(slID),
					DisplayName:    common.String("new-sl"),
					LifecycleState: ocicore.SecurityListLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := securityListMgrWithFake(fake)

	sl := &ociv1beta1.OciSecurityList{}
	sl.Name = "new-sl"
	sl.Namespace = "default"
	sl.Spec.DisplayName = "new-sl"
	sl.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	sl.Spec.VcnId = "ocid1.vcn.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), sl, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(slID), sl.Status.OsokStatus.Ocid)
}

func TestCreateOrUpdate_SecurityList_FindsExisting(t *testing.T) {
	slID := "ocid1.securitylist.oc1..existing"
	fake := &fakeVirtualNetworkClient{
		listSecurityListsFn: func(_ context.Context, _ ocicore.ListSecurityListsRequest) (ocicore.ListSecurityListsResponse, error) {
			return ocicore.ListSecurityListsResponse{
				Items: []ocicore.SecurityList{
					{Id: common.String(slID), DisplayName: common.String("existing-sl"), LifecycleState: ocicore.SecurityListLifecycleStateAvailable},
				},
			}, nil
		},
		getSecurityListFn: func(_ context.Context, _ ocicore.GetSecurityListRequest) (ocicore.GetSecurityListResponse, error) {
			return ocicore.GetSecurityListResponse{
				SecurityList: ocicore.SecurityList{
					Id:             common.String(slID),
					DisplayName:    common.String("existing-sl"),
					LifecycleState: ocicore.SecurityListLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := securityListMgrWithFake(fake)

	sl := &ociv1beta1.OciSecurityList{}
	sl.Spec.DisplayName = "existing-sl"
	sl.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	sl.Spec.VcnId = "ocid1.vcn.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), sl, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(slID), sl.Status.OsokStatus.Ocid)
}

func TestDelete_SecurityList_Succeeds(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteSecurityListFn: func(_ context.Context, _ ocicore.DeleteSecurityListRequest) (ocicore.DeleteSecurityListResponse, error) {
			deleteCalled = true
			return ocicore.DeleteSecurityListResponse{}, nil
		},
	}
	mgr := securityListMgrWithFake(fake)

	sl := &ociv1beta1.OciSecurityList{}
	sl.Status.OsokStatus.Ocid = "ocid1.securitylist.oc1..del"

	done, err := mgr.Delete(context.Background(), sl)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// ---------------------------------------------------------------------------
// NetworkSecurityGroup tests
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_NSG_CreatesNew(t *testing.T) {
	nsgID := "ocid1.networksecuritygroup.oc1..created"
	fake := &fakeVirtualNetworkClient{
		listNetworkSecurityGroupsFn: func(_ context.Context, _ ocicore.ListNetworkSecurityGroupsRequest) (ocicore.ListNetworkSecurityGroupsResponse, error) {
			return ocicore.ListNetworkSecurityGroupsResponse{Items: []ocicore.NetworkSecurityGroup{}}, nil
		},
		createNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.CreateNetworkSecurityGroupRequest) (ocicore.CreateNetworkSecurityGroupResponse, error) {
			return ocicore.CreateNetworkSecurityGroupResponse{
				NetworkSecurityGroup: ocicore.NetworkSecurityGroup{
					Id:             common.String(nsgID),
					DisplayName:    common.String("new-nsg"),
					LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := nsgMgrWithFake(fake)

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	nsg.Name = "new-nsg"
	nsg.Namespace = "default"
	nsg.Spec.DisplayName = "new-nsg"
	nsg.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	nsg.Spec.VcnId = "ocid1.vcn.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), nsg, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(nsgID), nsg.Status.OsokStatus.Ocid)
}

func TestCreateOrUpdate_NSG_FindsExisting(t *testing.T) {
	nsgID := "ocid1.networksecuritygroup.oc1..existing"
	fake := &fakeVirtualNetworkClient{
		listNetworkSecurityGroupsFn: func(_ context.Context, _ ocicore.ListNetworkSecurityGroupsRequest) (ocicore.ListNetworkSecurityGroupsResponse, error) {
			return ocicore.ListNetworkSecurityGroupsResponse{
				Items: []ocicore.NetworkSecurityGroup{
					{Id: common.String(nsgID), DisplayName: common.String("existing-nsg"), LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateAvailable},
				},
			}, nil
		},
		getNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error) {
			return ocicore.GetNetworkSecurityGroupResponse{
				NetworkSecurityGroup: ocicore.NetworkSecurityGroup{
					Id:             common.String(nsgID),
					DisplayName:    common.String("existing-nsg"),
					LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := nsgMgrWithFake(fake)

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	nsg.Spec.DisplayName = "existing-nsg"
	nsg.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	nsg.Spec.VcnId = "ocid1.vcn.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), nsg, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(nsgID), nsg.Status.OsokStatus.Ocid)
}

func TestDelete_NSG_Succeeds(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.DeleteNetworkSecurityGroupRequest) (ocicore.DeleteNetworkSecurityGroupResponse, error) {
			deleteCalled = true
			return ocicore.DeleteNetworkSecurityGroupResponse{}, nil
		},
	}
	mgr := nsgMgrWithFake(fake)

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	nsg.Status.OsokStatus.Ocid = "ocid1.networksecuritygroup.oc1..del"

	done, err := mgr.Delete(context.Background(), nsg)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// ---------------------------------------------------------------------------
// RouteTable tests
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_RouteTable_CreatesNew(t *testing.T) {
	rtID := "ocid1.routetable.oc1..created"
	fake := &fakeVirtualNetworkClient{
		listRouteTablesFn: func(_ context.Context, _ ocicore.ListRouteTablesRequest) (ocicore.ListRouteTablesResponse, error) {
			return ocicore.ListRouteTablesResponse{Items: []ocicore.RouteTable{}}, nil
		},
		createRouteTableFn: func(_ context.Context, _ ocicore.CreateRouteTableRequest) (ocicore.CreateRouteTableResponse, error) {
			return ocicore.CreateRouteTableResponse{
				RouteTable: ocicore.RouteTable{
					Id:             common.String(rtID),
					DisplayName:    common.String("new-rt"),
					LifecycleState: ocicore.RouteTableLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := routeTableMgrWithFake(fake)

	rt := &ociv1beta1.OciRouteTable{}
	rt.Name = "new-rt"
	rt.Namespace = "default"
	rt.Spec.DisplayName = "new-rt"
	rt.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	rt.Spec.VcnId = "ocid1.vcn.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), rt, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(rtID), rt.Status.OsokStatus.Ocid)
}

func TestCreateOrUpdate_RouteTable_FindsExisting(t *testing.T) {
	rtID := "ocid1.routetable.oc1..existing"
	fake := &fakeVirtualNetworkClient{
		listRouteTablesFn: func(_ context.Context, _ ocicore.ListRouteTablesRequest) (ocicore.ListRouteTablesResponse, error) {
			return ocicore.ListRouteTablesResponse{
				Items: []ocicore.RouteTable{
					{Id: common.String(rtID), DisplayName: common.String("existing-rt"), LifecycleState: ocicore.RouteTableLifecycleStateAvailable},
				},
			}, nil
		},
		getRouteTableFn: func(_ context.Context, _ ocicore.GetRouteTableRequest) (ocicore.GetRouteTableResponse, error) {
			return ocicore.GetRouteTableResponse{
				RouteTable: ocicore.RouteTable{
					Id:             common.String(rtID),
					DisplayName:    common.String("existing-rt"),
					LifecycleState: ocicore.RouteTableLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := routeTableMgrWithFake(fake)

	rt := &ociv1beta1.OciRouteTable{}
	rt.Spec.DisplayName = "existing-rt"
	rt.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	rt.Spec.VcnId = "ocid1.vcn.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), rt, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(rtID), rt.Status.OsokStatus.Ocid)
}

func TestDelete_RouteTable_Succeeds(t *testing.T) {
	var deleteCalled bool
	fake := &fakeVirtualNetworkClient{
		deleteRouteTableFn: func(_ context.Context, _ ocicore.DeleteRouteTableRequest) (ocicore.DeleteRouteTableResponse, error) {
			deleteCalled = true
			return ocicore.DeleteRouteTableResponse{}, nil
		},
	}
	mgr := routeTableMgrWithFake(fake)

	rt := &ociv1beta1.OciRouteTable{}
	rt.Status.OsokStatus.Ocid = "ocid1.routetable.oc1..del"

	done, err := mgr.Delete(context.Background(), rt)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// ---------------------------------------------------------------------------
// UpdateRouteTable reconciliation tests
// ---------------------------------------------------------------------------

func TestUpdateRouteTable_IncludesRouteRulesInRequest(t *testing.T) {
	var capturedReq ocicore.UpdateRouteTableRequest
	fake := &fakeVirtualNetworkClient{
		updateRouteTableFn: func(_ context.Context, req ocicore.UpdateRouteTableRequest) (ocicore.UpdateRouteTableResponse, error) {
			capturedReq = req
			return ocicore.UpdateRouteTableResponse{}, nil
		},
	}
	mgr := routeTableMgrWithFake(fake)

	rt := &ociv1beta1.OciRouteTable{}
	rt.Status.OsokStatus.Ocid = "ocid1.routetable.oc1..test"
	rt.Spec.DisplayName = "my-rt"
	rt.Spec.RouteRules = []ociv1beta1.RouteRule{
		{NetworkEntityId: "ocid1.internetgateway.oc1..igw", Destination: "0.0.0.0/0", DestinationType: "CIDR_BLOCK"},
	}

	err := mgr.UpdateRouteTable(context.Background(), rt)
	assert.NoError(t, err)
	assert.Equal(t, "ocid1.routetable.oc1..test", *capturedReq.RtId)
	assert.Len(t, capturedReq.UpdateRouteTableDetails.RouteRules, 1)
	assert.Equal(t, "ocid1.internetgateway.oc1..igw", *capturedReq.UpdateRouteTableDetails.RouteRules[0].NetworkEntityId)
	assert.Equal(t, "0.0.0.0/0", *capturedReq.UpdateRouteTableDetails.RouteRules[0].Destination)
}

func TestUpdateRouteTable_EmptyRulesClearsRules(t *testing.T) {
	var capturedReq ocicore.UpdateRouteTableRequest
	fake := &fakeVirtualNetworkClient{
		updateRouteTableFn: func(_ context.Context, req ocicore.UpdateRouteTableRequest) (ocicore.UpdateRouteTableResponse, error) {
			capturedReq = req
			return ocicore.UpdateRouteTableResponse{}, nil
		},
	}
	mgr := routeTableMgrWithFake(fake)

	rt := &ociv1beta1.OciRouteTable{}
	rt.Status.OsokStatus.Ocid = "ocid1.routetable.oc1..test"
	rt.Spec.RouteRules = nil

	err := mgr.UpdateRouteTable(context.Background(), rt)
	assert.NoError(t, err)
	// Update is always sent even with no rules (clears existing rules to match spec).
	assert.NotNil(t, capturedReq.UpdateRouteTableDetails)
	assert.Empty(t, capturedReq.UpdateRouteTableDetails.RouteRules)
}

// ---------------------------------------------------------------------------
// UpdateSecurityList reconciliation tests
// ---------------------------------------------------------------------------

func TestUpdateSecurityList_IncludesRulesInRequest(t *testing.T) {
	var capturedReq ocicore.UpdateSecurityListRequest
	fake := &fakeVirtualNetworkClient{
		updateSecurityListFn: func(_ context.Context, req ocicore.UpdateSecurityListRequest) (ocicore.UpdateSecurityListResponse, error) {
			capturedReq = req
			return ocicore.UpdateSecurityListResponse{}, nil
		},
	}
	mgr := securityListMgrWithFake(fake)

	sl := &ociv1beta1.OciSecurityList{}
	sl.Status.OsokStatus.Ocid = "ocid1.securitylist.oc1..test"
	sl.Spec.DisplayName = "my-sl"
	sl.Spec.EgressSecurityRules = []ociv1beta1.EgressSecurityRule{
		{Protocol: "all", Destination: "0.0.0.0/0", IsStateless: false},
	}
	sl.Spec.IngressSecurityRules = []ociv1beta1.IngressSecurityRule{
		{Protocol: "6", Source: "10.0.0.0/8", IsStateless: false},
	}

	err := mgr.UpdateSecurityList(context.Background(), sl)
	assert.NoError(t, err)
	assert.Equal(t, "ocid1.securitylist.oc1..test", *capturedReq.SecurityListId)
	assert.Len(t, capturedReq.UpdateSecurityListDetails.EgressSecurityRules, 1)
	assert.Equal(t, "0.0.0.0/0", *capturedReq.UpdateSecurityListDetails.EgressSecurityRules[0].Destination)
	assert.Len(t, capturedReq.UpdateSecurityListDetails.IngressSecurityRules, 1)
	assert.Equal(t, "10.0.0.0/8", *capturedReq.UpdateSecurityListDetails.IngressSecurityRules[0].Source)
}

func TestUpdateSecurityList_EmptyRulesClearsRules(t *testing.T) {
	var capturedReq ocicore.UpdateSecurityListRequest
	fake := &fakeVirtualNetworkClient{
		updateSecurityListFn: func(_ context.Context, req ocicore.UpdateSecurityListRequest) (ocicore.UpdateSecurityListResponse, error) {
			capturedReq = req
			return ocicore.UpdateSecurityListResponse{}, nil
		},
	}
	mgr := securityListMgrWithFake(fake)

	sl := &ociv1beta1.OciSecurityList{}
	sl.Status.OsokStatus.Ocid = "ocid1.securitylist.oc1..test"
	sl.Spec.EgressSecurityRules = nil
	sl.Spec.IngressSecurityRules = nil

	err := mgr.UpdateSecurityList(context.Background(), sl)
	assert.NoError(t, err)
	// Update is always sent (clears rules to match empty spec).
	assert.NotNil(t, capturedReq.UpdateSecurityListDetails)
	assert.Empty(t, capturedReq.UpdateSecurityListDetails.EgressSecurityRules)
	assert.Empty(t, capturedReq.UpdateSecurityListDetails.IngressSecurityRules)
}

// ---------------------------------------------------------------------------
// GetCrdStatus tests for all remaining resource types
// ---------------------------------------------------------------------------

func TestIGW_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciInternetGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	igw := &ociv1beta1.OciInternetGateway{}
	igw.Status.OsokStatus.Ocid = "ocid1.internetgateway.oc1..xxx"

	status, err := mgr.GetCrdStatus(igw)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.internetgateway.oc1..xxx"), status.Ocid)
}

func TestIGW_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciInternetGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	_, err := mgr.GetCrdStatus(&ociv1beta1.Stream{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

func TestNAT_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciNatGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	nat := &ociv1beta1.OciNatGateway{}
	nat.Status.OsokStatus.Ocid = "ocid1.natgateway.oc1..xxx"

	status, err := mgr.GetCrdStatus(nat)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.natgateway.oc1..xxx"), status.Ocid)
}

func TestNAT_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciNatGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	_, err := mgr.GetCrdStatus(&ociv1beta1.Stream{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

func TestSGW_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciServiceGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	sgw := &ociv1beta1.OciServiceGateway{}
	sgw.Status.OsokStatus.Ocid = "ocid1.servicegateway.oc1..xxx"

	status, err := mgr.GetCrdStatus(sgw)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.servicegateway.oc1..xxx"), status.Ocid)
}

func TestSGW_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciServiceGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	_, err := mgr.GetCrdStatus(&ociv1beta1.Stream{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

func TestDRG_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciDrgServiceManager(emptyProvider(), nil, nil, defaultLog())

	drg := &ociv1beta1.OciDrg{}
	drg.Status.OsokStatus.Ocid = "ocid1.drg.oc1..xxx"

	status, err := mgr.GetCrdStatus(drg)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.drg.oc1..xxx"), status.Ocid)
}

func TestDRG_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciDrgServiceManager(emptyProvider(), nil, nil, defaultLog())

	_, err := mgr.GetCrdStatus(&ociv1beta1.Stream{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

func TestSecurityList_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciSecurityListServiceManager(emptyProvider(), nil, nil, defaultLog())

	sl := &ociv1beta1.OciSecurityList{}
	sl.Status.OsokStatus.Ocid = "ocid1.securitylist.oc1..xxx"

	status, err := mgr.GetCrdStatus(sl)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.securitylist.oc1..xxx"), status.Ocid)
}

func TestSecurityList_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciSecurityListServiceManager(emptyProvider(), nil, nil, defaultLog())

	_, err := mgr.GetCrdStatus(&ociv1beta1.Stream{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

func TestNSG_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciNetworkSecurityGroupServiceManager(emptyProvider(), nil, nil, defaultLog())

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	nsg.Status.OsokStatus.Ocid = "ocid1.networksecuritygroup.oc1..xxx"

	status, err := mgr.GetCrdStatus(nsg)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.networksecuritygroup.oc1..xxx"), status.Ocid)
}

func TestNSG_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciNetworkSecurityGroupServiceManager(emptyProvider(), nil, nil, defaultLog())

	_, err := mgr.GetCrdStatus(&ociv1beta1.Stream{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

func TestRouteTable_GetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciRouteTableServiceManager(emptyProvider(), nil, nil, defaultLog())

	rt := &ociv1beta1.OciRouteTable{}
	rt.Status.OsokStatus.Ocid = "ocid1.routetable.oc1..xxx"

	status, err := mgr.GetCrdStatus(rt)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.routetable.oc1..xxx"), status.Ocid)
}

func TestRouteTable_GetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciRouteTableServiceManager(emptyProvider(), nil, nil, defaultLog())

	_, err := mgr.GetCrdStatus(&ociv1beta1.Stream{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// ---------------------------------------------------------------------------
// Update* tests: 0% coverage functions
// ---------------------------------------------------------------------------

func TestUpdateInternetGateway_SendsDisplayName(t *testing.T) {
	var capturedReq ocicore.UpdateInternetGatewayRequest
	igwID := "ocid1.internetgateway.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getInternetGatewayFn: func(_ context.Context, _ ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error) {
			return ocicore.GetInternetGatewayResponse{
				InternetGateway: ocicore.InternetGateway{
					Id:          common.String(igwID),
					DisplayName: common.String("old-name"),
				},
			}, nil
		},
		updateInternetGatewayFn: func(_ context.Context, req ocicore.UpdateInternetGatewayRequest) (ocicore.UpdateInternetGatewayResponse, error) {
			capturedReq = req
			return ocicore.UpdateInternetGatewayResponse{}, nil
		},
	}
	mgr := igwMgrWithFake(fake)

	igw := &ociv1beta1.OciInternetGateway{}
	igw.Status.OsokStatus.Ocid = ociv1beta1.OCID(igwID)
	igw.Spec.DisplayName = "new-name"

	err := mgr.UpdateInternetGateway(context.Background(), igw)
	assert.NoError(t, err)
	assert.Equal(t, igwID, *capturedReq.IgId)
	assert.Equal(t, "new-name", *capturedReq.UpdateInternetGatewayDetails.DisplayName)
}

func TestUpdateInternetGateway_NoUpdateNeeded(t *testing.T) {
	var updateCalled bool
	igwID := "ocid1.internetgateway.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getInternetGatewayFn: func(_ context.Context, _ ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error) {
			return ocicore.GetInternetGatewayResponse{
				InternetGateway: ocicore.InternetGateway{
					Id:          common.String(igwID),
					DisplayName: common.String("same-name"),
				},
			}, nil
		},
		updateInternetGatewayFn: func(_ context.Context, _ ocicore.UpdateInternetGatewayRequest) (ocicore.UpdateInternetGatewayResponse, error) {
			updateCalled = true
			return ocicore.UpdateInternetGatewayResponse{}, nil
		},
	}
	mgr := igwMgrWithFake(fake)

	igw := &ociv1beta1.OciInternetGateway{}
	igw.Status.OsokStatus.Ocid = ociv1beta1.OCID(igwID)
	igw.Spec.DisplayName = "same-name"

	err := mgr.UpdateInternetGateway(context.Background(), igw)
	assert.NoError(t, err)
	assert.False(t, updateCalled, "no update should be called when nothing changed")
}

func TestUpdateNatGateway_SendsDisplayName(t *testing.T) {
	var capturedReq ocicore.UpdateNatGatewayRequest
	natID := "ocid1.natgateway.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getNatGatewayFn: func(_ context.Context, _ ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error) {
			return ocicore.GetNatGatewayResponse{
				NatGateway: ocicore.NatGateway{
					Id:          common.String(natID),
					DisplayName: common.String("old-name"),
				},
			}, nil
		},
		updateNatGatewayFn: func(_ context.Context, req ocicore.UpdateNatGatewayRequest) (ocicore.UpdateNatGatewayResponse, error) {
			capturedReq = req
			return ocicore.UpdateNatGatewayResponse{}, nil
		},
	}
	mgr := natMgrWithFake(fake)

	nat := &ociv1beta1.OciNatGateway{}
	nat.Status.OsokStatus.Ocid = ociv1beta1.OCID(natID)
	nat.Spec.DisplayName = "new-name"

	err := mgr.UpdateNatGateway(context.Background(), nat)
	assert.NoError(t, err)
	assert.Equal(t, natID, *capturedReq.NatGatewayId)
	assert.Equal(t, "new-name", *capturedReq.UpdateNatGatewayDetails.DisplayName)
}

func TestUpdateNatGateway_NoUpdateNeeded(t *testing.T) {
	var updateCalled bool
	natID := "ocid1.natgateway.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getNatGatewayFn: func(_ context.Context, _ ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error) {
			return ocicore.GetNatGatewayResponse{
				NatGateway: ocicore.NatGateway{
					Id:          common.String(natID),
					DisplayName: common.String("same-name"),
				},
			}, nil
		},
		updateNatGatewayFn: func(_ context.Context, _ ocicore.UpdateNatGatewayRequest) (ocicore.UpdateNatGatewayResponse, error) {
			updateCalled = true
			return ocicore.UpdateNatGatewayResponse{}, nil
		},
	}
	mgr := natMgrWithFake(fake)

	nat := &ociv1beta1.OciNatGateway{}
	nat.Status.OsokStatus.Ocid = ociv1beta1.OCID(natID)
	nat.Spec.DisplayName = "same-name"

	err := mgr.UpdateNatGateway(context.Background(), nat)
	assert.NoError(t, err)
	assert.False(t, updateCalled)
}

func TestUpdateServiceGateway_SendsDisplayName(t *testing.T) {
	var capturedReq ocicore.UpdateServiceGatewayRequest
	sgwID := "ocid1.servicegateway.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getServiceGatewayFn: func(_ context.Context, _ ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error) {
			return ocicore.GetServiceGatewayResponse{
				ServiceGateway: ocicore.ServiceGateway{
					Id:          common.String(sgwID),
					DisplayName: common.String("old-name"),
				},
			}, nil
		},
		updateServiceGatewayFn: func(_ context.Context, req ocicore.UpdateServiceGatewayRequest) (ocicore.UpdateServiceGatewayResponse, error) {
			capturedReq = req
			return ocicore.UpdateServiceGatewayResponse{}, nil
		},
	}
	mgr := sgwMgrWithFake(fake)

	sgw := &ociv1beta1.OciServiceGateway{}
	sgw.Status.OsokStatus.Ocid = ociv1beta1.OCID(sgwID)
	sgw.Spec.DisplayName = "new-name"

	err := mgr.UpdateServiceGateway(context.Background(), sgw)
	assert.NoError(t, err)
	assert.Equal(t, sgwID, *capturedReq.ServiceGatewayId)
	assert.Equal(t, "new-name", *capturedReq.UpdateServiceGatewayDetails.DisplayName)
}

func TestUpdateServiceGateway_NoUpdateNeeded(t *testing.T) {
	var updateCalled bool
	sgwID := "ocid1.servicegateway.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getServiceGatewayFn: func(_ context.Context, _ ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error) {
			return ocicore.GetServiceGatewayResponse{
				ServiceGateway: ocicore.ServiceGateway{
					Id:          common.String(sgwID),
					DisplayName: common.String("same-name"),
				},
			}, nil
		},
		updateServiceGatewayFn: func(_ context.Context, _ ocicore.UpdateServiceGatewayRequest) (ocicore.UpdateServiceGatewayResponse, error) {
			updateCalled = true
			return ocicore.UpdateServiceGatewayResponse{}, nil
		},
	}
	mgr := sgwMgrWithFake(fake)

	sgw := &ociv1beta1.OciServiceGateway{}
	sgw.Status.OsokStatus.Ocid = ociv1beta1.OCID(sgwID)
	sgw.Spec.DisplayName = "same-name"

	err := mgr.UpdateServiceGateway(context.Background(), sgw)
	assert.NoError(t, err)
	assert.False(t, updateCalled)
}

func TestUpdateDrg_SendsDisplayName(t *testing.T) {
	var capturedReq ocicore.UpdateDrgRequest
	drgID := "ocid1.drg.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getDrgFn: func(_ context.Context, _ ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error) {
			return ocicore.GetDrgResponse{
				Drg: ocicore.Drg{
					Id:          common.String(drgID),
					DisplayName: common.String("old-name"),
				},
			}, nil
		},
		updateDrgFn: func(_ context.Context, req ocicore.UpdateDrgRequest) (ocicore.UpdateDrgResponse, error) {
			capturedReq = req
			return ocicore.UpdateDrgResponse{}, nil
		},
	}
	mgr := drgMgrWithFake(fake)

	drg := &ociv1beta1.OciDrg{}
	drg.Status.OsokStatus.Ocid = ociv1beta1.OCID(drgID)
	drg.Spec.DisplayName = "new-name"

	err := mgr.UpdateDrg(context.Background(), drg)
	assert.NoError(t, err)
	assert.Equal(t, drgID, *capturedReq.DrgId)
	assert.Equal(t, "new-name", *capturedReq.UpdateDrgDetails.DisplayName)
}

func TestUpdateDrg_NoUpdateNeeded(t *testing.T) {
	var updateCalled bool
	drgID := "ocid1.drg.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getDrgFn: func(_ context.Context, _ ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error) {
			return ocicore.GetDrgResponse{
				Drg: ocicore.Drg{
					Id:          common.String(drgID),
					DisplayName: common.String("same-name"),
				},
			}, nil
		},
		updateDrgFn: func(_ context.Context, _ ocicore.UpdateDrgRequest) (ocicore.UpdateDrgResponse, error) {
			updateCalled = true
			return ocicore.UpdateDrgResponse{}, nil
		},
	}
	mgr := drgMgrWithFake(fake)

	drg := &ociv1beta1.OciDrg{}
	drg.Status.OsokStatus.Ocid = ociv1beta1.OCID(drgID)
	drg.Spec.DisplayName = "same-name"

	err := mgr.UpdateDrg(context.Background(), drg)
	assert.NoError(t, err)
	assert.False(t, updateCalled)
}

func TestUpdateNetworkSecurityGroup_SendsDisplayName(t *testing.T) {
	var capturedReq ocicore.UpdateNetworkSecurityGroupRequest
	nsgID := "ocid1.networksecuritygroup.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error) {
			return ocicore.GetNetworkSecurityGroupResponse{
				NetworkSecurityGroup: ocicore.NetworkSecurityGroup{
					Id:          common.String(nsgID),
					DisplayName: common.String("old-name"),
				},
			}, nil
		},
		updateNetworkSecurityGroupFn: func(_ context.Context, req ocicore.UpdateNetworkSecurityGroupRequest) (ocicore.UpdateNetworkSecurityGroupResponse, error) {
			capturedReq = req
			return ocicore.UpdateNetworkSecurityGroupResponse{}, nil
		},
	}
	mgr := nsgMgrWithFake(fake)

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	nsg.Status.OsokStatus.Ocid = ociv1beta1.OCID(nsgID)
	nsg.Spec.DisplayName = "new-name"

	err := mgr.UpdateNetworkSecurityGroup(context.Background(), nsg)
	assert.NoError(t, err)
	assert.Equal(t, nsgID, *capturedReq.NetworkSecurityGroupId)
	assert.Equal(t, "new-name", *capturedReq.UpdateNetworkSecurityGroupDetails.DisplayName)
}

func TestUpdateNetworkSecurityGroup_NoUpdateNeeded(t *testing.T) {
	var updateCalled bool
	nsgID := "ocid1.networksecuritygroup.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error) {
			return ocicore.GetNetworkSecurityGroupResponse{
				NetworkSecurityGroup: ocicore.NetworkSecurityGroup{
					Id:          common.String(nsgID),
					DisplayName: common.String("same-name"),
				},
			}, nil
		},
		updateNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.UpdateNetworkSecurityGroupRequest) (ocicore.UpdateNetworkSecurityGroupResponse, error) {
			updateCalled = true
			return ocicore.UpdateNetworkSecurityGroupResponse{}, nil
		},
	}
	mgr := nsgMgrWithFake(fake)

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	nsg.Status.OsokStatus.Ocid = ociv1beta1.OCID(nsgID)
	nsg.Spec.DisplayName = "same-name"

	err := mgr.UpdateNetworkSecurityGroup(context.Background(), nsg)
	assert.NoError(t, err)
	assert.False(t, updateCalled)
}

func TestUpdateVcn_SendsDisplayName(t *testing.T) {
	var capturedReq ocicore.UpdateVcnRequest
	vcnID := "ocid1.vcn.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getVcnFn: func(_ context.Context, _ ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
			return ocicore.GetVcnResponse{
				Vcn: ocicore.Vcn{Id: common.String(vcnID), DisplayName: common.String("old-name")},
			}, nil
		},
		updateVcnFn: func(_ context.Context, req ocicore.UpdateVcnRequest) (ocicore.UpdateVcnResponse, error) {
			capturedReq = req
			return ocicore.UpdateVcnResponse{}, nil
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Status.OsokStatus.Ocid = ociv1beta1.OCID(vcnID)
	v.Spec.DisplayName = "new-name"

	err := mgr.UpdateVcn(context.Background(), v)
	assert.NoError(t, err)
	assert.Equal(t, vcnID, *capturedReq.VcnId)
	assert.Equal(t, "new-name", *capturedReq.UpdateVcnDetails.DisplayName)
}

func TestUpdateVcn_NoUpdateNeeded(t *testing.T) {
	var updateCalled bool
	vcnID := "ocid1.vcn.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getVcnFn: func(_ context.Context, _ ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
			return ocicore.GetVcnResponse{
				Vcn: ocicore.Vcn{Id: common.String(vcnID), DisplayName: common.String("same-name")},
			}, nil
		},
		updateVcnFn: func(_ context.Context, _ ocicore.UpdateVcnRequest) (ocicore.UpdateVcnResponse, error) {
			updateCalled = true
			return ocicore.UpdateVcnResponse{}, nil
		},
	}
	mgr := vcnMgrWithFake(fake)

	v := &ociv1beta1.OciVcn{}
	v.Status.OsokStatus.Ocid = ociv1beta1.OCID(vcnID)
	v.Spec.DisplayName = "same-name"

	err := mgr.UpdateVcn(context.Background(), v)
	assert.NoError(t, err)
	assert.False(t, updateCalled)
}

func TestUpdateSubnet_SendsDisplayName(t *testing.T) {
	var capturedReq ocicore.UpdateSubnetRequest
	subnetID := "ocid1.subnet.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getSubnetFn: func(_ context.Context, _ ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
			return ocicore.GetSubnetResponse{
				Subnet: ocicore.Subnet{Id: common.String(subnetID), DisplayName: common.String("old-name")},
			}, nil
		},
		updateSubnetFn: func(_ context.Context, req ocicore.UpdateSubnetRequest) (ocicore.UpdateSubnetResponse, error) {
			capturedReq = req
			return ocicore.UpdateSubnetResponse{}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Status.OsokStatus.Ocid = ociv1beta1.OCID(subnetID)
	s.Spec.DisplayName = "new-name"

	err := mgr.UpdateSubnet(context.Background(), s)
	assert.NoError(t, err)
	assert.Equal(t, subnetID, *capturedReq.SubnetId)
	assert.Equal(t, "new-name", *capturedReq.UpdateSubnetDetails.DisplayName)
}

func TestUpdateSubnet_NoUpdateNeeded(t *testing.T) {
	var updateCalled bool
	subnetID := "ocid1.subnet.oc1..test"
	fake := &fakeVirtualNetworkClient{
		getSubnetFn: func(_ context.Context, _ ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
			return ocicore.GetSubnetResponse{
				Subnet: ocicore.Subnet{Id: common.String(subnetID), DisplayName: common.String("same-name")},
			}, nil
		},
		updateSubnetFn: func(_ context.Context, _ ocicore.UpdateSubnetRequest) (ocicore.UpdateSubnetResponse, error) {
			updateCalled = true
			return ocicore.UpdateSubnetResponse{}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := &ociv1beta1.OciSubnet{}
	s.Status.OsokStatus.Ocid = ociv1beta1.OCID(subnetID)
	s.Spec.DisplayName = "same-name"

	err := mgr.UpdateSubnet(context.Background(), s)
	assert.NoError(t, err)
	assert.False(t, updateCalled)
}

// ---------------------------------------------------------------------------
// CreateOrUpdate "bind to existing" path for each resource type
// ---------------------------------------------------------------------------

func TestIGW_CreateOrUpdate_WithId_Binds(t *testing.T) {
	igwID := "ocid1.internetgateway.oc1..bind"
	fake := &fakeVirtualNetworkClient{
		getInternetGatewayFn: func(_ context.Context, _ ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error) {
			return ocicore.GetInternetGatewayResponse{
				InternetGateway: ocicore.InternetGateway{
					Id:             common.String(igwID),
					DisplayName:    common.String("bind-igw"),
					LifecycleState: ocicore.InternetGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := igwMgrWithFake(fake)

	igw := &ociv1beta1.OciInternetGateway{}
	igw.Spec.InternetGatewayId = ociv1beta1.OCID(igwID)
	igw.Spec.DisplayName = "bind-igw"
	igw.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	igw.Spec.VcnId = "ocid1.vcn.oc1..parent"
	igw.Status.OsokStatus.Ocid = ociv1beta1.OCID(igwID)

	resp, err := mgr.CreateOrUpdate(context.Background(), igw, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(igwID), igw.Status.OsokStatus.Ocid)
}

func TestNAT_CreateOrUpdate_WithId_Binds(t *testing.T) {
	natID := "ocid1.natgateway.oc1..bind"
	fake := &fakeVirtualNetworkClient{
		getNatGatewayFn: func(_ context.Context, _ ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error) {
			return ocicore.GetNatGatewayResponse{
				NatGateway: ocicore.NatGateway{
					Id:             common.String(natID),
					DisplayName:    common.String("bind-nat"),
					LifecycleState: ocicore.NatGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := natMgrWithFake(fake)

	nat := &ociv1beta1.OciNatGateway{}
	nat.Spec.NatGatewayId = ociv1beta1.OCID(natID)
	nat.Spec.DisplayName = "bind-nat"
	nat.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	nat.Spec.VcnId = "ocid1.vcn.oc1..parent"
	nat.Status.OsokStatus.Ocid = ociv1beta1.OCID(natID)

	resp, err := mgr.CreateOrUpdate(context.Background(), nat, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(natID), nat.Status.OsokStatus.Ocid)
}

func TestSGW_CreateOrUpdate_WithId_Binds(t *testing.T) {
	sgwID := "ocid1.servicegateway.oc1..bind"
	fake := &fakeVirtualNetworkClient{
		getServiceGatewayFn: func(_ context.Context, _ ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error) {
			return ocicore.GetServiceGatewayResponse{
				ServiceGateway: ocicore.ServiceGateway{
					Id:             common.String(sgwID),
					DisplayName:    common.String("bind-sgw"),
					LifecycleState: ocicore.ServiceGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := sgwMgrWithFake(fake)

	sgw := &ociv1beta1.OciServiceGateway{}
	sgw.Spec.ServiceGatewayId = ociv1beta1.OCID(sgwID)
	sgw.Spec.DisplayName = "bind-sgw"
	sgw.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	sgw.Spec.VcnId = "ocid1.vcn.oc1..parent"
	sgw.Spec.Services = []string{"ocid1.service.oc1..svc"}
	sgw.Status.OsokStatus.Ocid = ociv1beta1.OCID(sgwID)

	resp, err := mgr.CreateOrUpdate(context.Background(), sgw, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(sgwID), sgw.Status.OsokStatus.Ocid)
}

func TestDRG_CreateOrUpdate_WithId_Binds(t *testing.T) {
	drgID := "ocid1.drg.oc1..bind"
	fake := &fakeVirtualNetworkClient{
		getDrgFn: func(_ context.Context, _ ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error) {
			return ocicore.GetDrgResponse{
				Drg: ocicore.Drg{
					Id:             common.String(drgID),
					DisplayName:    common.String("bind-drg"),
					LifecycleState: ocicore.DrgLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := drgMgrWithFake(fake)

	drg := &ociv1beta1.OciDrg{}
	drg.Spec.DrgId = ociv1beta1.OCID(drgID)
	drg.Spec.DisplayName = "bind-drg"
	drg.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	drg.Status.OsokStatus.Ocid = ociv1beta1.OCID(drgID)

	resp, err := mgr.CreateOrUpdate(context.Background(), drg, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(drgID), drg.Status.OsokStatus.Ocid)
}

func TestSecurityList_CreateOrUpdate_WithId_Binds(t *testing.T) {
	slID := "ocid1.securitylist.oc1..bind"
	fake := &fakeVirtualNetworkClient{
		getSecurityListFn: func(_ context.Context, _ ocicore.GetSecurityListRequest) (ocicore.GetSecurityListResponse, error) {
			return ocicore.GetSecurityListResponse{
				SecurityList: ocicore.SecurityList{
					Id:             common.String(slID),
					DisplayName:    common.String("bind-sl"),
					LifecycleState: ocicore.SecurityListLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := securityListMgrWithFake(fake)

	sl := &ociv1beta1.OciSecurityList{}
	sl.Spec.SecurityListId = ociv1beta1.OCID(slID)
	sl.Spec.DisplayName = "bind-sl"
	sl.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	sl.Spec.VcnId = "ocid1.vcn.oc1..xxx"
	sl.Status.OsokStatus.Ocid = ociv1beta1.OCID(slID)

	resp, err := mgr.CreateOrUpdate(context.Background(), sl, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(slID), sl.Status.OsokStatus.Ocid)
}

func TestNSG_CreateOrUpdate_WithId_Binds(t *testing.T) {
	nsgID := "ocid1.networksecuritygroup.oc1..bind"
	fake := &fakeVirtualNetworkClient{
		getNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error) {
			return ocicore.GetNetworkSecurityGroupResponse{
				NetworkSecurityGroup: ocicore.NetworkSecurityGroup{
					Id:             common.String(nsgID),
					DisplayName:    common.String("bind-nsg"),
					LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := nsgMgrWithFake(fake)

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	nsg.Spec.NetworkSecurityGroupId = ociv1beta1.OCID(nsgID)
	nsg.Spec.DisplayName = "bind-nsg"
	nsg.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	nsg.Spec.VcnId = "ocid1.vcn.oc1..xxx"
	nsg.Status.OsokStatus.Ocid = ociv1beta1.OCID(nsgID)

	resp, err := mgr.CreateOrUpdate(context.Background(), nsg, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(nsgID), nsg.Status.OsokStatus.Ocid)
}

func TestRouteTable_CreateOrUpdate_WithId_Binds(t *testing.T) {
	rtID := "ocid1.routetable.oc1..bind"
	fake := &fakeVirtualNetworkClient{
		getRouteTableFn: func(_ context.Context, _ ocicore.GetRouteTableRequest) (ocicore.GetRouteTableResponse, error) {
			return ocicore.GetRouteTableResponse{
				RouteTable: ocicore.RouteTable{
					Id:             common.String(rtID),
					DisplayName:    common.String("bind-rt"),
					LifecycleState: ocicore.RouteTableLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := routeTableMgrWithFake(fake)

	rt := &ociv1beta1.OciRouteTable{}
	rt.Spec.RouteTableId = ociv1beta1.OCID(rtID)
	rt.Spec.DisplayName = "bind-rt"
	rt.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	rt.Spec.VcnId = "ocid1.vcn.oc1..xxx"
	rt.Status.OsokStatus.Ocid = ociv1beta1.OCID(rtID)

	resp, err := mgr.CreateOrUpdate(context.Background(), rt, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(rtID), rt.Status.OsokStatus.Ocid)
}

// ---------------------------------------------------------------------------
// Delete error / empty OCID path tests
// ---------------------------------------------------------------------------

func TestIGW_Delete_NoOcid(t *testing.T) {
	mgr := NewOciInternetGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	igw := &ociv1beta1.OciInternetGateway{}
	done, err := mgr.Delete(context.Background(), igw)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestIGW_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteInternetGatewayFn: func(_ context.Context, _ ocicore.DeleteInternetGatewayRequest) (ocicore.DeleteInternetGatewayResponse, error) {
			return ocicore.DeleteInternetGatewayResponse{}, errors.New("delete failed")
		},
	}
	mgr := igwMgrWithFake(fake)

	igw := &ociv1beta1.OciInternetGateway{}
	igw.Status.OsokStatus.Ocid = "ocid1.internetgateway.oc1..del"

	done, err := mgr.Delete(context.Background(), igw)
	assert.Error(t, err)
	assert.False(t, done)
}

func TestNAT_Delete_NoOcid(t *testing.T) {
	mgr := NewOciNatGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	nat := &ociv1beta1.OciNatGateway{}
	done, err := mgr.Delete(context.Background(), nat)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestNAT_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteNatGatewayFn: func(_ context.Context, _ ocicore.DeleteNatGatewayRequest) (ocicore.DeleteNatGatewayResponse, error) {
			return ocicore.DeleteNatGatewayResponse{}, errors.New("delete failed")
		},
	}
	mgr := natMgrWithFake(fake)

	nat := &ociv1beta1.OciNatGateway{}
	nat.Status.OsokStatus.Ocid = "ocid1.natgateway.oc1..del"

	done, err := mgr.Delete(context.Background(), nat)
	assert.Error(t, err)
	assert.False(t, done)
}

func TestSGW_Delete_NoOcid(t *testing.T) {
	mgr := NewOciServiceGatewayServiceManager(emptyProvider(), nil, nil, defaultLog())

	sgw := &ociv1beta1.OciServiceGateway{}
	done, err := mgr.Delete(context.Background(), sgw)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestSGW_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteServiceGatewayFn: func(_ context.Context, _ ocicore.DeleteServiceGatewayRequest) (ocicore.DeleteServiceGatewayResponse, error) {
			return ocicore.DeleteServiceGatewayResponse{}, errors.New("delete failed")
		},
	}
	mgr := sgwMgrWithFake(fake)

	sgw := &ociv1beta1.OciServiceGateway{}
	sgw.Status.OsokStatus.Ocid = "ocid1.servicegateway.oc1..del"

	done, err := mgr.Delete(context.Background(), sgw)
	assert.Error(t, err)
	assert.False(t, done)
}

func TestDRG_Delete_NoOcid(t *testing.T) {
	mgr := NewOciDrgServiceManager(emptyProvider(), nil, nil, defaultLog())

	drg := &ociv1beta1.OciDrg{}
	done, err := mgr.Delete(context.Background(), drg)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestDRG_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteDrgFn: func(_ context.Context, _ ocicore.DeleteDrgRequest) (ocicore.DeleteDrgResponse, error) {
			return ocicore.DeleteDrgResponse{}, errors.New("delete failed")
		},
	}
	mgr := drgMgrWithFake(fake)

	drg := &ociv1beta1.OciDrg{}
	drg.Status.OsokStatus.Ocid = "ocid1.drg.oc1..del"

	done, err := mgr.Delete(context.Background(), drg)
	assert.Error(t, err)
	assert.False(t, done)
}

func TestSecurityList_Delete_NoOcid(t *testing.T) {
	mgr := NewOciSecurityListServiceManager(emptyProvider(), nil, nil, defaultLog())

	sl := &ociv1beta1.OciSecurityList{}
	done, err := mgr.Delete(context.Background(), sl)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestSecurityList_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteSecurityListFn: func(_ context.Context, _ ocicore.DeleteSecurityListRequest) (ocicore.DeleteSecurityListResponse, error) {
			return ocicore.DeleteSecurityListResponse{}, errors.New("delete failed")
		},
	}
	mgr := securityListMgrWithFake(fake)

	sl := &ociv1beta1.OciSecurityList{}
	sl.Status.OsokStatus.Ocid = "ocid1.securitylist.oc1..del"

	done, err := mgr.Delete(context.Background(), sl)
	assert.Error(t, err)
	assert.False(t, done)
}

func TestNSG_Delete_NoOcid(t *testing.T) {
	mgr := NewOciNetworkSecurityGroupServiceManager(emptyProvider(), nil, nil, defaultLog())

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	done, err := mgr.Delete(context.Background(), nsg)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestNSG_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.DeleteNetworkSecurityGroupRequest) (ocicore.DeleteNetworkSecurityGroupResponse, error) {
			return ocicore.DeleteNetworkSecurityGroupResponse{}, errors.New("delete failed")
		},
	}
	mgr := nsgMgrWithFake(fake)

	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	nsg.Status.OsokStatus.Ocid = "ocid1.networksecuritygroup.oc1..del"

	done, err := mgr.Delete(context.Background(), nsg)
	assert.Error(t, err)
	assert.False(t, done)
}

func TestRouteTable_Delete_NoOcid(t *testing.T) {
	mgr := NewOciRouteTableServiceManager(emptyProvider(), nil, nil, defaultLog())

	rt := &ociv1beta1.OciRouteTable{}
	done, err := mgr.Delete(context.Background(), rt)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestRouteTable_Delete_Error(t *testing.T) {
	fake := &fakeVirtualNetworkClient{
		deleteRouteTableFn: func(_ context.Context, _ ocicore.DeleteRouteTableRequest) (ocicore.DeleteRouteTableResponse, error) {
			return ocicore.DeleteRouteTableResponse{}, errors.New("delete failed")
		},
	}
	mgr := routeTableMgrWithFake(fake)

	rt := &ociv1beta1.OciRouteTable{}
	rt.Status.OsokStatus.Ocid = "ocid1.routetable.oc1..del"

	done, err := mgr.Delete(context.Background(), rt)
	assert.Error(t, err)
	assert.False(t, done)
}

// ---------------------------------------------------------------------------
// CreateNatGateway optional fields: BlockTraffic
// ---------------------------------------------------------------------------

func TestCreateNatGateway_WithBlockTraffic(t *testing.T) {
	var capturedReq ocicore.CreateNatGatewayRequest
	natID := "ocid1.natgateway.oc1..block"
	fake := &fakeVirtualNetworkClient{
		createNatGatewayFn: func(_ context.Context, req ocicore.CreateNatGatewayRequest) (ocicore.CreateNatGatewayResponse, error) {
			capturedReq = req
			return ocicore.CreateNatGatewayResponse{
				NatGateway: ocicore.NatGateway{
					Id:             common.String(natID),
					DisplayName:    common.String("block-nat"),
					LifecycleState: ocicore.NatGatewayLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := natMgrWithFake(fake)

	nat := ociv1beta1.OciNatGateway{}
	nat.Spec.DisplayName = "block-nat"
	nat.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	nat.Spec.VcnId = "ocid1.vcn.oc1..parent"
	nat.Spec.BlockTraffic = true

	result, err := mgr.CreateNatGateway(context.Background(), nat)
	assert.NoError(t, err)
	assert.Equal(t, natID, *result.Id)
	assert.NotNil(t, capturedReq.CreateNatGatewayDetails.BlockTraffic)
	assert.True(t, *capturedReq.CreateNatGatewayDetails.BlockTraffic)
}

// ---------------------------------------------------------------------------
// CreateSubnet optional fields
// ---------------------------------------------------------------------------

func TestCreateSubnet_WithOptionalFields(t *testing.T) {
	var capturedReq ocicore.CreateSubnetRequest
	subnetID := "ocid1.subnet.oc1..opts"
	vcnID := "ocid1.vcn.oc1..parent"
	rtID := "ocid1.routetable.oc1..rt"
	slID := "ocid1.securitylist.oc1..sl"
	fake := &fakeVirtualNetworkClient{
		createSubnetFn: func(_ context.Context, req ocicore.CreateSubnetRequest) (ocicore.CreateSubnetResponse, error) {
			capturedReq = req
			return ocicore.CreateSubnetResponse{
				Subnet: ocicore.Subnet{
					Id:             common.String(subnetID),
					DisplayName:    common.String("opts-subnet"),
					LifecycleState: ocicore.SubnetLifecycleStateAvailable,
				},
			}, nil
		},
	}
	mgr := subnetMgrWithFake(fake)

	s := ociv1beta1.OciSubnet{}
	s.Spec.DisplayName = "opts-subnet"
	s.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	s.Spec.VcnId = ociv1beta1.OCID(vcnID)
	s.Spec.CidrBlock = "10.0.2.0/24"
	s.Spec.DnsLabel = "optsubnet"
	s.Spec.ProhibitPublicIpOnVnic = true
	s.Spec.RouteTableId = ociv1beta1.OCID(rtID)
	s.Spec.SecurityListIds = []ociv1beta1.OCID{ociv1beta1.OCID(slID)}

	result, err := mgr.CreateSubnet(context.Background(), s)
	assert.NoError(t, err)
	assert.Equal(t, subnetID, *result.Id)
	assert.Equal(t, "optsubnet", *capturedReq.CreateSubnetDetails.DnsLabel)
	assert.NotNil(t, capturedReq.CreateSubnetDetails.ProhibitPublicIpOnVnic)
	assert.True(t, *capturedReq.CreateSubnetDetails.ProhibitPublicIpOnVnic)
	assert.Equal(t, rtID, *capturedReq.CreateSubnetDetails.RouteTableId)
	assert.Equal(t, []string{slID}, capturedReq.CreateSubnetDetails.SecurityListIds)
}

// ---------------------------------------------------------------------------
// buildIngressRules / buildEgressRules — table-driven coverage
// ---------------------------------------------------------------------------

func TestBuildIngressRules_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		input ociv1beta1.IngressSecurityRule
		check func(t *testing.T, r ocicore.IngressSecurityRule)
	}{
		{
			name: "minimal",
			input: ociv1beta1.IngressSecurityRule{
				Protocol:    "all",
				Source:      "0.0.0.0/0",
				IsStateless: false,
			},
			check: func(t *testing.T, r ocicore.IngressSecurityRule) {
				assert.Equal(t, "all", *r.Protocol)
				assert.Equal(t, "0.0.0.0/0", *r.Source)
				assert.False(t, *r.IsStateless)
				assert.Nil(t, r.Description)
				assert.Nil(t, r.TcpOptions)
				assert.Nil(t, r.UdpOptions)
			},
		},
		{
			name: "with_description",
			input: ociv1beta1.IngressSecurityRule{
				Protocol:    "6",
				Source:      "10.0.0.0/8",
				Description: "allow tcp",
				IsStateless: true,
			},
			check: func(t *testing.T, r ocicore.IngressSecurityRule) {
				assert.Equal(t, "allow tcp", *r.Description)
				assert.True(t, *r.IsStateless)
			},
		},
		{
			name: "with_tcp_dest_port",
			input: ociv1beta1.IngressSecurityRule{
				Protocol: "6",
				Source:   "10.0.0.0/8",
				TcpOptions: &ociv1beta1.TcpOptions{
					DestinationPortRange: &ociv1beta1.PortRange{Min: 443, Max: 443},
				},
			},
			check: func(t *testing.T, r ocicore.IngressSecurityRule) {
				assert.NotNil(t, r.TcpOptions)
				assert.Equal(t, 443, *r.TcpOptions.DestinationPortRange.Min)
				assert.Equal(t, 443, *r.TcpOptions.DestinationPortRange.Max)
				assert.Nil(t, r.TcpOptions.SourcePortRange)
			},
		},
		{
			name: "with_tcp_src_port",
			input: ociv1beta1.IngressSecurityRule{
				Protocol: "6",
				Source:   "10.0.0.0/8",
				TcpOptions: &ociv1beta1.TcpOptions{
					SourcePortRange: &ociv1beta1.PortRange{Min: 1024, Max: 65535},
				},
			},
			check: func(t *testing.T, r ocicore.IngressSecurityRule) {
				assert.NotNil(t, r.TcpOptions)
				assert.Nil(t, r.TcpOptions.DestinationPortRange)
				assert.Equal(t, 1024, *r.TcpOptions.SourcePortRange.Min)
			},
		},
		{
			name: "with_udp_dest_port",
			input: ociv1beta1.IngressSecurityRule{
				Protocol: "17",
				Source:   "10.0.0.0/8",
				UdpOptions: &ociv1beta1.UdpOptions{
					DestinationPortRange: &ociv1beta1.PortRange{Min: 53, Max: 53},
				},
			},
			check: func(t *testing.T, r ocicore.IngressSecurityRule) {
				assert.NotNil(t, r.UdpOptions)
				assert.Equal(t, 53, *r.UdpOptions.DestinationPortRange.Min)
			},
		},
		{
			name: "with_udp_src_port",
			input: ociv1beta1.IngressSecurityRule{
				Protocol: "17",
				Source:   "10.0.0.0/8",
				UdpOptions: &ociv1beta1.UdpOptions{
					SourcePortRange: &ociv1beta1.PortRange{Min: 1024, Max: 65535},
				},
			},
			check: func(t *testing.T, r ocicore.IngressSecurityRule) {
				assert.NotNil(t, r.UdpOptions)
				assert.Nil(t, r.UdpOptions.DestinationPortRange)
				assert.Equal(t, 1024, *r.UdpOptions.SourcePortRange.Min)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Exercise buildIngressRules via CreateSecurityList which calls it.
			slID := "ocid1.securitylist.oc1..build"
			var capturedReq ocicore.CreateSecurityListRequest
			fake := &fakeVirtualNetworkClient{
				createSecurityListFn: func(_ context.Context, req ocicore.CreateSecurityListRequest) (ocicore.CreateSecurityListResponse, error) {
					capturedReq = req
					return ocicore.CreateSecurityListResponse{
						SecurityList: ocicore.SecurityList{
							Id:             common.String(slID),
							LifecycleState: ocicore.SecurityListLifecycleStateAvailable,
						},
					}, nil
				},
			}
			mgr := securityListMgrWithFake(fake)

			sl := ociv1beta1.OciSecurityList{}
			sl.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
			sl.Spec.VcnId = "ocid1.vcn.oc1..xxx"
			sl.Spec.IngressSecurityRules = []ociv1beta1.IngressSecurityRule{tc.input}

			_, err := mgr.CreateSecurityList(context.Background(), sl)
			assert.NoError(t, err)
			assert.Len(t, capturedReq.CreateSecurityListDetails.IngressSecurityRules, 1)
			tc.check(t, capturedReq.CreateSecurityListDetails.IngressSecurityRules[0])
		})
	}
}

func TestBuildEgressRules_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		input ociv1beta1.EgressSecurityRule
		check func(t *testing.T, r ocicore.EgressSecurityRule)
	}{
		{
			name: "minimal",
			input: ociv1beta1.EgressSecurityRule{
				Protocol:    "all",
				Destination: "0.0.0.0/0",
				IsStateless: false,
			},
			check: func(t *testing.T, r ocicore.EgressSecurityRule) {
				assert.Equal(t, "all", *r.Protocol)
				assert.Equal(t, "0.0.0.0/0", *r.Destination)
				assert.False(t, *r.IsStateless)
				assert.Nil(t, r.Description)
				assert.Nil(t, r.TcpOptions)
				assert.Nil(t, r.UdpOptions)
			},
		},
		{
			name: "with_destination_type",
			input: ociv1beta1.EgressSecurityRule{
				Protocol:        "all",
				Destination:     "all-iad-services-in-oracle-services-network",
				DestinationType: "SERVICE_CIDR_BLOCK",
			},
			check: func(t *testing.T, r ocicore.EgressSecurityRule) {
				assert.Equal(t, ocicore.EgressSecurityRuleDestinationTypeEnum("SERVICE_CIDR_BLOCK"), r.DestinationType)
			},
		},
		{
			name: "with_description",
			input: ociv1beta1.EgressSecurityRule{
				Protocol:    "6",
				Destination: "10.0.0.0/8",
				Description: "allow egress tcp",
				IsStateless: true,
			},
			check: func(t *testing.T, r ocicore.EgressSecurityRule) {
				assert.Equal(t, "allow egress tcp", *r.Description)
				assert.True(t, *r.IsStateless)
			},
		},
		{
			name: "with_tcp_dest_port",
			input: ociv1beta1.EgressSecurityRule{
				Protocol:    "6",
				Destination: "10.0.0.0/8",
				TcpOptions: &ociv1beta1.TcpOptions{
					DestinationPortRange: &ociv1beta1.PortRange{Min: 80, Max: 80},
				},
			},
			check: func(t *testing.T, r ocicore.EgressSecurityRule) {
				assert.NotNil(t, r.TcpOptions)
				assert.Equal(t, 80, *r.TcpOptions.DestinationPortRange.Min)
			},
		},
		{
			name: "with_tcp_src_port",
			input: ociv1beta1.EgressSecurityRule{
				Protocol:    "6",
				Destination: "10.0.0.0/8",
				TcpOptions: &ociv1beta1.TcpOptions{
					SourcePortRange: &ociv1beta1.PortRange{Min: 1024, Max: 65535},
				},
			},
			check: func(t *testing.T, r ocicore.EgressSecurityRule) {
				assert.NotNil(t, r.TcpOptions)
				assert.Nil(t, r.TcpOptions.DestinationPortRange)
				assert.Equal(t, 1024, *r.TcpOptions.SourcePortRange.Min)
			},
		},
		{
			name: "with_udp_dest_port",
			input: ociv1beta1.EgressSecurityRule{
				Protocol:    "17",
				Destination: "10.0.0.0/8",
				UdpOptions: &ociv1beta1.UdpOptions{
					DestinationPortRange: &ociv1beta1.PortRange{Min: 53, Max: 53},
				},
			},
			check: func(t *testing.T, r ocicore.EgressSecurityRule) {
				assert.NotNil(t, r.UdpOptions)
				assert.Equal(t, 53, *r.UdpOptions.DestinationPortRange.Min)
			},
		},
		{
			name: "with_udp_src_port",
			input: ociv1beta1.EgressSecurityRule{
				Protocol:    "17",
				Destination: "10.0.0.0/8",
				UdpOptions: &ociv1beta1.UdpOptions{
					SourcePortRange: &ociv1beta1.PortRange{Min: 1024, Max: 65535},
				},
			},
			check: func(t *testing.T, r ocicore.EgressSecurityRule) {
				assert.NotNil(t, r.UdpOptions)
				assert.Nil(t, r.UdpOptions.DestinationPortRange)
				assert.Equal(t, 1024, *r.UdpOptions.SourcePortRange.Min)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			slID := "ocid1.securitylist.oc1..build"
			var capturedReq ocicore.CreateSecurityListRequest
			fake := &fakeVirtualNetworkClient{
				createSecurityListFn: func(_ context.Context, req ocicore.CreateSecurityListRequest) (ocicore.CreateSecurityListResponse, error) {
					capturedReq = req
					return ocicore.CreateSecurityListResponse{
						SecurityList: ocicore.SecurityList{
							Id:             common.String(slID),
							LifecycleState: ocicore.SecurityListLifecycleStateAvailable,
						},
					}, nil
				},
			}
			mgr := securityListMgrWithFake(fake)

			sl := ociv1beta1.OciSecurityList{}
			sl.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
			sl.Spec.VcnId = "ocid1.vcn.oc1..xxx"
			sl.Spec.EgressSecurityRules = []ociv1beta1.EgressSecurityRule{tc.input}

			_, err := mgr.CreateSecurityList(context.Background(), sl)
			assert.NoError(t, err)
			assert.Len(t, capturedReq.CreateSecurityListDetails.EgressSecurityRules, 1)
			tc.check(t, capturedReq.CreateSecurityListDetails.EgressSecurityRules[0])
		})
	}
}
