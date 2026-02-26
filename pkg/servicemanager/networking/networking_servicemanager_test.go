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
