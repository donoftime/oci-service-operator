/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestPropertyNetworkingPendingStatesRequestRequeue(t *testing.T) {
	for _, state := range []string{"PROVISIONING", "UPDATING"} {
		t.Run(state, func(t *testing.T) {
			cases := []struct {
				name string
				run  func(*testing.T)
			}{
				{
					name: "vcn",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listVcnsFn: func(_ context.Context, _ ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
								return ocicore.ListVcnsResponse{Items: []ocicore.Vcn{{Id: common.String("ocid1.vcn.oc1..pending"), LifecycleState: ocicore.VcnLifecycleStateEnum(state)}}}, nil
							},
							getVcnFn: func(_ context.Context, _ ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
								return ocicore.GetVcnResponse{Vcn: ocicore.Vcn{Id: common.String("ocid1.vcn.oc1..pending"), DisplayName: common.String("pending-vcn"), LifecycleState: ocicore.VcnLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := vcnMgrWithFake(fake)
						v := &ociv1beta1.OciVcn{}
						v.Spec.DisplayName = "pending-vcn"
						v.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
				{
					name: "subnet",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listSubnetsFn: func(_ context.Context, _ ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
								return ocicore.ListSubnetsResponse{Items: []ocicore.Subnet{{Id: common.String("ocid1.subnet.oc1..pending"), LifecycleState: ocicore.SubnetLifecycleStateEnum(state)}}}, nil
							},
							getSubnetFn: func(_ context.Context, _ ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
								return ocicore.GetSubnetResponse{Subnet: ocicore.Subnet{Id: common.String("ocid1.subnet.oc1..pending"), DisplayName: common.String("pending-subnet"), LifecycleState: ocicore.SubnetLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := subnetMgrWithFake(fake)
						s := &ociv1beta1.OciSubnet{}
						s.Spec.DisplayName = "pending-subnet"
						s.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						s.Spec.VcnId = "ocid1.vcn.oc1..parent"
						s.Spec.CidrBlock = "10.0.0.0/24"
						resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
				{
					name: "internet-gateway",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listInternetGatewaysFn: func(_ context.Context, _ ocicore.ListInternetGatewaysRequest) (ocicore.ListInternetGatewaysResponse, error) {
								return ocicore.ListInternetGatewaysResponse{Items: []ocicore.InternetGateway{{Id: common.String("ocid1.igw.oc1..pending"), DisplayName: common.String("pending-igw"), LifecycleState: ocicore.InternetGatewayLifecycleStateEnum(state)}}}, nil
							},
							getInternetGatewayFn: func(_ context.Context, _ ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error) {
								return ocicore.GetInternetGatewayResponse{InternetGateway: ocicore.InternetGateway{Id: common.String("ocid1.igw.oc1..pending"), DisplayName: common.String("pending-igw"), LifecycleState: ocicore.InternetGatewayLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := igwMgrWithFake(fake)
						igw := &ociv1beta1.OciInternetGateway{}
						igw.Spec.DisplayName = "pending-igw"
						igw.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						igw.Spec.VcnId = "ocid1.vcn.oc1..parent"
						resp, err := mgr.CreateOrUpdate(context.Background(), igw, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
				{
					name: "nat-gateway",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listNatGatewaysFn: func(_ context.Context, _ ocicore.ListNatGatewaysRequest) (ocicore.ListNatGatewaysResponse, error) {
								return ocicore.ListNatGatewaysResponse{Items: []ocicore.NatGateway{{Id: common.String("ocid1.nat.oc1..pending"), DisplayName: common.String("pending-nat"), LifecycleState: ocicore.NatGatewayLifecycleStateEnum(state)}}}, nil
							},
							getNatGatewayFn: func(_ context.Context, _ ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error) {
								return ocicore.GetNatGatewayResponse{NatGateway: ocicore.NatGateway{Id: common.String("ocid1.nat.oc1..pending"), DisplayName: common.String("pending-nat"), LifecycleState: ocicore.NatGatewayLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := natMgrWithFake(fake)
						nat := &ociv1beta1.OciNatGateway{}
						nat.Spec.DisplayName = "pending-nat"
						nat.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						nat.Spec.VcnId = "ocid1.vcn.oc1..parent"
						resp, err := mgr.CreateOrUpdate(context.Background(), nat, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
				{
					name: "service-gateway",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listServiceGatewaysFn: func(_ context.Context, _ ocicore.ListServiceGatewaysRequest) (ocicore.ListServiceGatewaysResponse, error) {
								return ocicore.ListServiceGatewaysResponse{Items: []ocicore.ServiceGateway{{Id: common.String("ocid1.sgw.oc1..pending"), DisplayName: common.String("pending-sgw"), LifecycleState: ocicore.ServiceGatewayLifecycleStateEnum(state)}}}, nil
							},
							getServiceGatewayFn: func(_ context.Context, _ ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error) {
								return ocicore.GetServiceGatewayResponse{ServiceGateway: ocicore.ServiceGateway{Id: common.String("ocid1.sgw.oc1..pending"), DisplayName: common.String("pending-sgw"), LifecycleState: ocicore.ServiceGatewayLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := sgwMgrWithFake(fake)
						sgw := &ociv1beta1.OciServiceGateway{}
						sgw.Spec.DisplayName = "pending-sgw"
						sgw.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						sgw.Spec.VcnId = "ocid1.vcn.oc1..parent"
						sgw.Spec.Services = []string{"ocid1.service.oc1..svc"}
						resp, err := mgr.CreateOrUpdate(context.Background(), sgw, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
				{
					name: "drg",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listDrgsFn: func(_ context.Context, _ ocicore.ListDrgsRequest) (ocicore.ListDrgsResponse, error) {
								return ocicore.ListDrgsResponse{Items: []ocicore.Drg{{Id: common.String("ocid1.drg.oc1..pending"), DisplayName: common.String("pending-drg"), LifecycleState: ocicore.DrgLifecycleStateEnum(state)}}}, nil
							},
							getDrgFn: func(_ context.Context, _ ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error) {
								return ocicore.GetDrgResponse{Drg: ocicore.Drg{Id: common.String("ocid1.drg.oc1..pending"), DisplayName: common.String("pending-drg"), LifecycleState: ocicore.DrgLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := drgMgrWithFake(fake)
						drg := &ociv1beta1.OciDrg{}
						drg.Spec.DisplayName = "pending-drg"
						drg.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						resp, err := mgr.CreateOrUpdate(context.Background(), drg, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
				{
					name: "security-list",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listSecurityListsFn: func(_ context.Context, _ ocicore.ListSecurityListsRequest) (ocicore.ListSecurityListsResponse, error) {
								return ocicore.ListSecurityListsResponse{Items: []ocicore.SecurityList{{Id: common.String("ocid1.sl.oc1..pending"), DisplayName: common.String("pending-sl"), LifecycleState: ocicore.SecurityListLifecycleStateEnum(state)}}}, nil
							},
							getSecurityListFn: func(_ context.Context, _ ocicore.GetSecurityListRequest) (ocicore.GetSecurityListResponse, error) {
								return ocicore.GetSecurityListResponse{SecurityList: ocicore.SecurityList{Id: common.String("ocid1.sl.oc1..pending"), DisplayName: common.String("pending-sl"), LifecycleState: ocicore.SecurityListLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := securityListMgrWithFake(fake)
						sl := &ociv1beta1.OciSecurityList{}
						sl.Spec.DisplayName = "pending-sl"
						sl.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						sl.Spec.VcnId = "ocid1.vcn.oc1..parent"
						resp, err := mgr.CreateOrUpdate(context.Background(), sl, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
				{
					name: "network-security-group",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listNetworkSecurityGroupsFn: func(_ context.Context, _ ocicore.ListNetworkSecurityGroupsRequest) (ocicore.ListNetworkSecurityGroupsResponse, error) {
								return ocicore.ListNetworkSecurityGroupsResponse{Items: []ocicore.NetworkSecurityGroup{{Id: common.String("ocid1.nsg.oc1..pending"), DisplayName: common.String("pending-nsg"), LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateEnum(state)}}}, nil
							},
							getNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error) {
								return ocicore.GetNetworkSecurityGroupResponse{NetworkSecurityGroup: ocicore.NetworkSecurityGroup{Id: common.String("ocid1.nsg.oc1..pending"), DisplayName: common.String("pending-nsg"), LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := nsgMgrWithFake(fake)
						nsg := &ociv1beta1.OciNetworkSecurityGroup{}
						nsg.Spec.DisplayName = "pending-nsg"
						nsg.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						nsg.Spec.VcnId = "ocid1.vcn.oc1..parent"
						resp, err := mgr.CreateOrUpdate(context.Background(), nsg, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
				{
					name: "route-table",
					run: func(t *testing.T) {
						fake := &fakeVirtualNetworkClient{
							listRouteTablesFn: func(_ context.Context, _ ocicore.ListRouteTablesRequest) (ocicore.ListRouteTablesResponse, error) {
								return ocicore.ListRouteTablesResponse{Items: []ocicore.RouteTable{{Id: common.String("ocid1.rt.oc1..pending"), DisplayName: common.String("pending-rt"), LifecycleState: ocicore.RouteTableLifecycleStateEnum(state)}}}, nil
							},
							getRouteTableFn: func(_ context.Context, _ ocicore.GetRouteTableRequest) (ocicore.GetRouteTableResponse, error) {
								return ocicore.GetRouteTableResponse{RouteTable: ocicore.RouteTable{Id: common.String("ocid1.rt.oc1..pending"), DisplayName: common.String("pending-rt"), LifecycleState: ocicore.RouteTableLifecycleStateEnum(state)}}, nil
							},
						}
						mgr := routeTableMgrWithFake(fake)
						rt := &ociv1beta1.OciRouteTable{}
						rt.Spec.DisplayName = "pending-rt"
						rt.Spec.CompartmentId = "ocid1.compartment.oc1..x"
						rt.Spec.VcnId = "ocid1.vcn.oc1..parent"
						resp, err := mgr.CreateOrUpdate(context.Background(), rt, ctrl.Request{})
						assert.NoError(t, err)
						assert.False(t, resp.IsSuccessful)
						assert.True(t, resp.ShouldRequeue)
					},
				},
			}

			for _, tc := range cases {
				t.Run(tc.name, tc.run)
			}
		})
	}
}

func TestPropertyNetworkingBindByIDUsesExplicitSpecIDWhenStatusIsEmpty(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "vcn",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getVcnFn: func(_ context.Context, req ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
						return ocicore.GetVcnResponse{Vcn: makeAvailableVcn(*req.VcnId, "existing-vcn")}, nil
					},
					updateVcnFn: func(_ context.Context, req ocicore.UpdateVcnRequest) (ocicore.UpdateVcnResponse, error) {
						updatedID = *req.VcnId
						return ocicore.UpdateVcnResponse{}, nil
					},
				}
				mgr := vcnMgrWithFake(fake)
				v := &ociv1beta1.OciVcn{}
				v.Spec.VcnId = "ocid1.vcn.oc1..bind-empty-status"
				v.Spec.DisplayName = "desired-vcn"
				resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(v.Spec.VcnId), updatedID)
			},
		},
		{
			name: "subnet",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getSubnetFn: func(_ context.Context, req ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
						return ocicore.GetSubnetResponse{Subnet: makeAvailableSubnet(*req.SubnetId, "existing-subnet", "ocid1.vcn.oc1..parent")}, nil
					},
					updateSubnetFn: func(_ context.Context, req ocicore.UpdateSubnetRequest) (ocicore.UpdateSubnetResponse, error) {
						updatedID = *req.SubnetId
						return ocicore.UpdateSubnetResponse{}, nil
					},
				}
				mgr := subnetMgrWithFake(fake)
				s := &ociv1beta1.OciSubnet{}
				s.Spec.SubnetId = "ocid1.subnet.oc1..bind-empty-status"
				s.Spec.DisplayName = "desired-subnet"
				s.Spec.VcnId = "ocid1.vcn.oc1..parent"
				s.Spec.CidrBlock = "10.0.0.0/24"
				resp, err := mgr.CreateOrUpdate(context.Background(), s, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(s.Spec.SubnetId), updatedID)
			},
		},
		{
			name: "internet-gateway",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getInternetGatewayFn: func(_ context.Context, req ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error) {
						return ocicore.GetInternetGatewayResponse{InternetGateway: ocicore.InternetGateway{Id: req.IgId, DisplayName: common.String("existing-igw"), LifecycleState: ocicore.InternetGatewayLifecycleStateAvailable}}, nil
					},
					updateInternetGatewayFn: func(_ context.Context, req ocicore.UpdateInternetGatewayRequest) (ocicore.UpdateInternetGatewayResponse, error) {
						updatedID = *req.IgId
						return ocicore.UpdateInternetGatewayResponse{}, nil
					},
				}
				mgr := igwMgrWithFake(fake)
				igw := &ociv1beta1.OciInternetGateway{}
				igw.Spec.InternetGatewayId = "ocid1.internetgateway.oc1..bind-empty-status"
				igw.Spec.DisplayName = "desired-igw"
				resp, err := mgr.CreateOrUpdate(context.Background(), igw, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(igw.Spec.InternetGatewayId), updatedID)
			},
		},
		{
			name: "nat-gateway",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getNatGatewayFn: func(_ context.Context, req ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error) {
						return ocicore.GetNatGatewayResponse{NatGateway: ocicore.NatGateway{Id: req.NatGatewayId, DisplayName: common.String("existing-nat"), LifecycleState: ocicore.NatGatewayLifecycleStateAvailable}}, nil
					},
					updateNatGatewayFn: func(_ context.Context, req ocicore.UpdateNatGatewayRequest) (ocicore.UpdateNatGatewayResponse, error) {
						updatedID = *req.NatGatewayId
						return ocicore.UpdateNatGatewayResponse{}, nil
					},
				}
				mgr := natMgrWithFake(fake)
				nat := &ociv1beta1.OciNatGateway{}
				nat.Spec.NatGatewayId = "ocid1.natgateway.oc1..bind-empty-status"
				nat.Spec.DisplayName = "desired-nat"
				resp, err := mgr.CreateOrUpdate(context.Background(), nat, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(nat.Spec.NatGatewayId), updatedID)
			},
		},
		{
			name: "service-gateway",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getServiceGatewayFn: func(_ context.Context, req ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error) {
						return ocicore.GetServiceGatewayResponse{ServiceGateway: ocicore.ServiceGateway{Id: req.ServiceGatewayId, DisplayName: common.String("existing-sgw"), LifecycleState: ocicore.ServiceGatewayLifecycleStateAvailable}}, nil
					},
					updateServiceGatewayFn: func(_ context.Context, req ocicore.UpdateServiceGatewayRequest) (ocicore.UpdateServiceGatewayResponse, error) {
						updatedID = *req.ServiceGatewayId
						return ocicore.UpdateServiceGatewayResponse{}, nil
					},
				}
				mgr := sgwMgrWithFake(fake)
				sgw := &ociv1beta1.OciServiceGateway{}
				sgw.Spec.ServiceGatewayId = "ocid1.servicegateway.oc1..bind-empty-status"
				sgw.Spec.DisplayName = "desired-sgw"
				sgw.Spec.Services = []string{"ocid1.service.oc1..svc"}
				resp, err := mgr.CreateOrUpdate(context.Background(), sgw, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(sgw.Spec.ServiceGatewayId), updatedID)
			},
		},
		{
			name: "drg",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getDrgFn: func(_ context.Context, req ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error) {
						return ocicore.GetDrgResponse{Drg: ocicore.Drg{Id: req.DrgId, DisplayName: common.String("existing-drg"), LifecycleState: ocicore.DrgLifecycleStateAvailable}}, nil
					},
					updateDrgFn: func(_ context.Context, req ocicore.UpdateDrgRequest) (ocicore.UpdateDrgResponse, error) {
						updatedID = *req.DrgId
						return ocicore.UpdateDrgResponse{}, nil
					},
				}
				mgr := drgMgrWithFake(fake)
				drg := &ociv1beta1.OciDrg{}
				drg.Spec.DrgId = "ocid1.drg.oc1..bind-empty-status"
				drg.Spec.DisplayName = "desired-drg"
				resp, err := mgr.CreateOrUpdate(context.Background(), drg, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(drg.Spec.DrgId), updatedID)
			},
		},
		{
			name: "security-list",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getSecurityListFn: func(_ context.Context, req ocicore.GetSecurityListRequest) (ocicore.GetSecurityListResponse, error) {
						return ocicore.GetSecurityListResponse{SecurityList: ocicore.SecurityList{Id: req.SecurityListId, DisplayName: common.String("existing-sl"), LifecycleState: ocicore.SecurityListLifecycleStateAvailable}}, nil
					},
					updateSecurityListFn: func(_ context.Context, req ocicore.UpdateSecurityListRequest) (ocicore.UpdateSecurityListResponse, error) {
						updatedID = *req.SecurityListId
						return ocicore.UpdateSecurityListResponse{}, nil
					},
				}
				mgr := securityListMgrWithFake(fake)
				sl := &ociv1beta1.OciSecurityList{}
				sl.Spec.SecurityListId = "ocid1.securitylist.oc1..bind-empty-status"
				sl.Spec.DisplayName = "desired-sl"
				resp, err := mgr.CreateOrUpdate(context.Background(), sl, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(sl.Spec.SecurityListId), updatedID)
			},
		},
		{
			name: "network-security-group",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getNetworkSecurityGroupFn: func(_ context.Context, req ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error) {
						return ocicore.GetNetworkSecurityGroupResponse{NetworkSecurityGroup: ocicore.NetworkSecurityGroup{Id: req.NetworkSecurityGroupId, DisplayName: common.String("existing-nsg"), LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateAvailable}}, nil
					},
					updateNetworkSecurityGroupFn: func(_ context.Context, req ocicore.UpdateNetworkSecurityGroupRequest) (ocicore.UpdateNetworkSecurityGroupResponse, error) {
						updatedID = *req.NetworkSecurityGroupId
						return ocicore.UpdateNetworkSecurityGroupResponse{}, nil
					},
				}
				mgr := nsgMgrWithFake(fake)
				nsg := &ociv1beta1.OciNetworkSecurityGroup{}
				nsg.Spec.NetworkSecurityGroupId = "ocid1.networksecuritygroup.oc1..bind-empty-status"
				nsg.Spec.DisplayName = "desired-nsg"
				resp, err := mgr.CreateOrUpdate(context.Background(), nsg, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(nsg.Spec.NetworkSecurityGroupId), updatedID)
			},
		},
		{
			name: "route-table",
			run: func(t *testing.T) {
				var updatedID string
				fake := &fakeVirtualNetworkClient{
					getRouteTableFn: func(_ context.Context, req ocicore.GetRouteTableRequest) (ocicore.GetRouteTableResponse, error) {
						return ocicore.GetRouteTableResponse{RouteTable: ocicore.RouteTable{Id: req.RtId, DisplayName: common.String("existing-rt"), LifecycleState: ocicore.RouteTableLifecycleStateAvailable}}, nil
					},
					updateRouteTableFn: func(_ context.Context, req ocicore.UpdateRouteTableRequest) (ocicore.UpdateRouteTableResponse, error) {
						updatedID = *req.RtId
						return ocicore.UpdateRouteTableResponse{}, nil
					},
				}
				mgr := routeTableMgrWithFake(fake)
				rt := &ociv1beta1.OciRouteTable{}
				rt.Spec.RouteTableId = "ocid1.routetable.oc1..bind-empty-status"
				rt.Spec.DisplayName = "desired-rt"
				resp, err := mgr.CreateOrUpdate(context.Background(), rt, ctrl.Request{})
				assert.NoError(t, err)
				assert.True(t, resp.IsSuccessful)
				assert.Equal(t, string(rt.Spec.RouteTableId), updatedID)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestPropertyNetworkingDeleteWaitsForConfirmedDisappearance(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "vcn",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteVcnFn: func(_ context.Context, _ ocicore.DeleteVcnRequest) (ocicore.DeleteVcnResponse, error) {
					return ocicore.DeleteVcnResponse{}, nil
				}}
				mgr := vcnMgrWithFake(fake)
				done, err := mgr.Delete(context.Background(), &ociv1beta1.OciVcn{Status: ociv1beta1.OciVcnStatus{OsokStatus: ociv1beta1.OSOKStatus{Ocid: "ocid1.vcn.oc1..still-there"}}})
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
		{
			name: "subnet",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteSubnetFn: func(_ context.Context, _ ocicore.DeleteSubnetRequest) (ocicore.DeleteSubnetResponse, error) {
					return ocicore.DeleteSubnetResponse{}, nil
				}}
				mgr := subnetMgrWithFake(fake)
				done, err := mgr.Delete(context.Background(), &ociv1beta1.OciSubnet{Status: ociv1beta1.OciSubnetStatus{OsokStatus: ociv1beta1.OSOKStatus{Ocid: "ocid1.subnet.oc1..still-there"}}})
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
		{
			name: "internet-gateway",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteInternetGatewayFn: func(_ context.Context, _ ocicore.DeleteInternetGatewayRequest) (ocicore.DeleteInternetGatewayResponse, error) {
					return ocicore.DeleteInternetGatewayResponse{}, nil
				}}
				mgr := igwMgrWithFake(fake)
				igw := &ociv1beta1.OciInternetGateway{}
				igw.Status.OsokStatus.Ocid = "ocid1.internetgateway.oc1..still-there"
				done, err := mgr.Delete(context.Background(), igw)
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
		{
			name: "nat-gateway",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteNatGatewayFn: func(_ context.Context, _ ocicore.DeleteNatGatewayRequest) (ocicore.DeleteNatGatewayResponse, error) {
					return ocicore.DeleteNatGatewayResponse{}, nil
				}}
				mgr := natMgrWithFake(fake)
				nat := &ociv1beta1.OciNatGateway{}
				nat.Status.OsokStatus.Ocid = "ocid1.natgateway.oc1..still-there"
				done, err := mgr.Delete(context.Background(), nat)
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
		{
			name: "service-gateway",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteServiceGatewayFn: func(_ context.Context, _ ocicore.DeleteServiceGatewayRequest) (ocicore.DeleteServiceGatewayResponse, error) {
					return ocicore.DeleteServiceGatewayResponse{}, nil
				}}
				mgr := sgwMgrWithFake(fake)
				sgw := &ociv1beta1.OciServiceGateway{}
				sgw.Status.OsokStatus.Ocid = "ocid1.servicegateway.oc1..still-there"
				done, err := mgr.Delete(context.Background(), sgw)
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
		{
			name: "drg",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteDrgFn: func(_ context.Context, _ ocicore.DeleteDrgRequest) (ocicore.DeleteDrgResponse, error) {
					return ocicore.DeleteDrgResponse{}, nil
				}}
				mgr := drgMgrWithFake(fake)
				drg := &ociv1beta1.OciDrg{}
				drg.Status.OsokStatus.Ocid = "ocid1.drg.oc1..still-there"
				done, err := mgr.Delete(context.Background(), drg)
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
		{
			name: "security-list",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteSecurityListFn: func(_ context.Context, _ ocicore.DeleteSecurityListRequest) (ocicore.DeleteSecurityListResponse, error) {
					return ocicore.DeleteSecurityListResponse{}, nil
				}}
				mgr := securityListMgrWithFake(fake)
				sl := &ociv1beta1.OciSecurityList{}
				sl.Status.OsokStatus.Ocid = "ocid1.securitylist.oc1..still-there"
				done, err := mgr.Delete(context.Background(), sl)
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
		{
			name: "network-security-group",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.DeleteNetworkSecurityGroupRequest) (ocicore.DeleteNetworkSecurityGroupResponse, error) {
					return ocicore.DeleteNetworkSecurityGroupResponse{}, nil
				}}
				mgr := nsgMgrWithFake(fake)
				nsg := &ociv1beta1.OciNetworkSecurityGroup{}
				nsg.Status.OsokStatus.Ocid = "ocid1.networksecuritygroup.oc1..still-there"
				done, err := mgr.Delete(context.Background(), nsg)
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
		{
			name: "route-table",
			run: func(t *testing.T) {
				fake := &fakeVirtualNetworkClient{deleteRouteTableFn: func(_ context.Context, _ ocicore.DeleteRouteTableRequest) (ocicore.DeleteRouteTableResponse, error) {
					return ocicore.DeleteRouteTableResponse{}, nil
				}}
				mgr := routeTableMgrWithFake(fake)
				rt := &ociv1beta1.OciRouteTable{}
				rt.Status.OsokStatus.Ocid = "ocid1.routetable.oc1..still-there"
				done, err := mgr.Delete(context.Background(), rt)
				assert.NoError(t, err)
				assert.False(t, done)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestPropertyNetworkingPaginatedLookupFindsSecondPage(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "vcn",
			run: func(t *testing.T) {
				secondPageID := "ocid1.vcn.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listVcnsFn: func(_ context.Context, req ocicore.ListVcnsRequest) (ocicore.ListVcnsResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListVcnsResponse{
								Items:       []ocicore.Vcn{{Id: common.String("ocid1.vcn.oc1..page1"), DisplayName: common.String("other-vcn")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListVcnsResponse{
							Items: []ocicore.Vcn{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-vcn"),
								LifecycleState: ocicore.VcnLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := vcnMgrWithFake(fake)
				vcn := ociv1beta1.OciVcn{}
				vcn.Spec.DisplayName = "target-vcn"
				vcn.Spec.CompartmentId = "ocid1.compartment.oc1..x"

				ocid, err := mgr.GetVcnOcid(context.Background(), vcn)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
		{
			name: "subnet",
			run: func(t *testing.T) {
				secondPageID := "ocid1.subnet.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listSubnetsFn: func(_ context.Context, req ocicore.ListSubnetsRequest) (ocicore.ListSubnetsResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListSubnetsResponse{
								Items:       []ocicore.Subnet{{Id: common.String("ocid1.subnet.oc1..page1"), DisplayName: common.String("other-subnet")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListSubnetsResponse{
							Items: []ocicore.Subnet{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-subnet"),
								LifecycleState: ocicore.SubnetLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := subnetMgrWithFake(fake)
				subnet := ociv1beta1.OciSubnet{}
				subnet.Spec.DisplayName = "target-subnet"
				subnet.Spec.CompartmentId = "ocid1.compartment.oc1..x"
				subnet.Spec.VcnId = "ocid1.vcn.oc1..parent"

				ocid, err := mgr.GetSubnetOcid(context.Background(), subnet)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
		{
			name: "internet-gateway",
			run: func(t *testing.T) {
				secondPageID := "ocid1.igw.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listInternetGatewaysFn: func(_ context.Context, req ocicore.ListInternetGatewaysRequest) (ocicore.ListInternetGatewaysResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListInternetGatewaysResponse{
								Items:       []ocicore.InternetGateway{{Id: common.String("ocid1.igw.oc1..page1"), DisplayName: common.String("other-igw")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListInternetGatewaysResponse{
							Items: []ocicore.InternetGateway{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-igw"),
								LifecycleState: ocicore.InternetGatewayLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := igwMgrWithFake(fake)
				igw := ociv1beta1.OciInternetGateway{}
				igw.Spec.DisplayName = "target-igw"
				igw.Spec.CompartmentId = "ocid1.compartment.oc1..x"
				igw.Spec.VcnId = "ocid1.vcn.oc1..parent"

				ocid, err := mgr.GetInternetGatewayOcid(context.Background(), igw)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
		{
			name: "nat-gateway",
			run: func(t *testing.T) {
				secondPageID := "ocid1.nat.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listNatGatewaysFn: func(_ context.Context, req ocicore.ListNatGatewaysRequest) (ocicore.ListNatGatewaysResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListNatGatewaysResponse{
								Items:       []ocicore.NatGateway{{Id: common.String("ocid1.nat.oc1..page1"), DisplayName: common.String("other-nat")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListNatGatewaysResponse{
							Items: []ocicore.NatGateway{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-nat"),
								LifecycleState: ocicore.NatGatewayLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := natMgrWithFake(fake)
				nat := ociv1beta1.OciNatGateway{}
				nat.Spec.DisplayName = "target-nat"
				nat.Spec.CompartmentId = "ocid1.compartment.oc1..x"
				nat.Spec.VcnId = "ocid1.vcn.oc1..parent"

				ocid, err := mgr.GetNatGatewayOcid(context.Background(), nat)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
		{
			name: "service-gateway",
			run: func(t *testing.T) {
				secondPageID := "ocid1.sgw.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listServiceGatewaysFn: func(_ context.Context, req ocicore.ListServiceGatewaysRequest) (ocicore.ListServiceGatewaysResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListServiceGatewaysResponse{
								Items:       []ocicore.ServiceGateway{{Id: common.String("ocid1.sgw.oc1..page1"), DisplayName: common.String("other-sgw")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListServiceGatewaysResponse{
							Items: []ocicore.ServiceGateway{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-sgw"),
								LifecycleState: ocicore.ServiceGatewayLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := sgwMgrWithFake(fake)
				sgw := ociv1beta1.OciServiceGateway{}
				sgw.Spec.DisplayName = "target-sgw"
				sgw.Spec.CompartmentId = "ocid1.compartment.oc1..x"
				sgw.Spec.VcnId = "ocid1.vcn.oc1..parent"

				ocid, err := mgr.GetServiceGatewayOcid(context.Background(), sgw)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
		{
			name: "drg",
			run: func(t *testing.T) {
				secondPageID := "ocid1.drg.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listDrgsFn: func(_ context.Context, req ocicore.ListDrgsRequest) (ocicore.ListDrgsResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListDrgsResponse{
								Items:       []ocicore.Drg{{Id: common.String("ocid1.drg.oc1..page1"), DisplayName: common.String("other-drg")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListDrgsResponse{
							Items: []ocicore.Drg{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-drg"),
								LifecycleState: ocicore.DrgLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := drgMgrWithFake(fake)
				drg := ociv1beta1.OciDrg{}
				drg.Spec.DisplayName = "target-drg"
				drg.Spec.CompartmentId = "ocid1.compartment.oc1..x"

				ocid, err := mgr.GetDrgOcid(context.Background(), drg)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
		{
			name: "security-list",
			run: func(t *testing.T) {
				secondPageID := "ocid1.sl.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listSecurityListsFn: func(_ context.Context, req ocicore.ListSecurityListsRequest) (ocicore.ListSecurityListsResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListSecurityListsResponse{
								Items:       []ocicore.SecurityList{{Id: common.String("ocid1.sl.oc1..page1"), DisplayName: common.String("other-sl")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListSecurityListsResponse{
							Items: []ocicore.SecurityList{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-sl"),
								LifecycleState: ocicore.SecurityListLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := securityListMgrWithFake(fake)
				sl := ociv1beta1.OciSecurityList{}
				sl.Spec.DisplayName = "target-sl"
				sl.Spec.CompartmentId = "ocid1.compartment.oc1..x"
				sl.Spec.VcnId = "ocid1.vcn.oc1..parent"

				ocid, err := mgr.GetSecurityListOcid(context.Background(), sl)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
		{
			name: "network-security-group",
			run: func(t *testing.T) {
				secondPageID := "ocid1.nsg.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listNetworkSecurityGroupsFn: func(_ context.Context, req ocicore.ListNetworkSecurityGroupsRequest) (ocicore.ListNetworkSecurityGroupsResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListNetworkSecurityGroupsResponse{
								Items:       []ocicore.NetworkSecurityGroup{{Id: common.String("ocid1.nsg.oc1..page1"), DisplayName: common.String("other-nsg")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListNetworkSecurityGroupsResponse{
							Items: []ocicore.NetworkSecurityGroup{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-nsg"),
								LifecycleState: ocicore.NetworkSecurityGroupLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := nsgMgrWithFake(fake)
				nsg := ociv1beta1.OciNetworkSecurityGroup{}
				nsg.Spec.DisplayName = "target-nsg"
				nsg.Spec.CompartmentId = "ocid1.compartment.oc1..x"
				nsg.Spec.VcnId = "ocid1.vcn.oc1..parent"

				ocid, err := mgr.GetNetworkSecurityGroupOcid(context.Background(), nsg)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
		{
			name: "route-table",
			run: func(t *testing.T) {
				secondPageID := "ocid1.rt.oc1..page2"
				fake := &fakeVirtualNetworkClient{
					listRouteTablesFn: func(_ context.Context, req ocicore.ListRouteTablesRequest) (ocicore.ListRouteTablesResponse, error) {
						if req.Page == nil {
							next := "page-2"
							return ocicore.ListRouteTablesResponse{
								Items:       []ocicore.RouteTable{{Id: common.String("ocid1.rt.oc1..page1"), DisplayName: common.String("other-rt")}},
								OpcNextPage: &next,
							}, nil
						}
						return ocicore.ListRouteTablesResponse{
							Items: []ocicore.RouteTable{{
								Id:             common.String(secondPageID),
								DisplayName:    common.String("target-rt"),
								LifecycleState: ocicore.RouteTableLifecycleStateAvailable,
							}},
						}, nil
					},
				}
				mgr := routeTableMgrWithFake(fake)
				rt := ociv1beta1.OciRouteTable{}
				rt.Spec.DisplayName = "target-rt"
				rt.Spec.CompartmentId = "ocid1.compartment.oc1..x"
				rt.Spec.VcnId = "ocid1.vcn.oc1..parent"

				ocid, err := mgr.GetRouteTableOcid(context.Background(), rt)
				assert.NoError(t, err)
				assert.NotNil(t, ocid)
				assert.Equal(t, ociv1beta1.OCID(secondPageID), *ocid)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func TestPropertyNetworkingDefinedTagDriftTriggersUpdate(t *testing.T) {
	desiredTags := map[string]ociv1beta1.MapValue{
		"ops": {"env": "prod"},
	}
	expectedTags := map[string]map[string]interface{}{
		"ops": {"env": "prod"},
	}

	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "vcn",
			run: func(t *testing.T) {
				var captured ocicore.UpdateVcnRequest
				fake := &fakeVirtualNetworkClient{
					getVcnFn: func(_ context.Context, _ ocicore.GetVcnRequest) (ocicore.GetVcnResponse, error) {
						return ocicore.GetVcnResponse{Vcn: ocicore.Vcn{Id: common.String("ocid1.vcn.oc1..tags"), DefinedTags: map[string]map[string]interface{}{"ops": {"env": "dev"}}}}, nil
					},
					updateVcnFn: func(_ context.Context, req ocicore.UpdateVcnRequest) (ocicore.UpdateVcnResponse, error) {
						captured = req
						return ocicore.UpdateVcnResponse{}, nil
					},
				}
				mgr := vcnMgrWithFake(fake)
				v := &ociv1beta1.OciVcn{}
				v.Status.OsokStatus.Ocid = "ocid1.vcn.oc1..tags"
				v.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateVcn(context.Background(), v))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
		{
			name: "subnet",
			run: func(t *testing.T) {
				var captured ocicore.UpdateSubnetRequest
				fake := &fakeVirtualNetworkClient{
					getSubnetFn: func(_ context.Context, _ ocicore.GetSubnetRequest) (ocicore.GetSubnetResponse, error) {
						return ocicore.GetSubnetResponse{Subnet: ocicore.Subnet{Id: common.String("ocid1.subnet.oc1..tags"), DefinedTags: map[string]map[string]interface{}{"ops": {"env": "dev"}}}}, nil
					},
					updateSubnetFn: func(_ context.Context, req ocicore.UpdateSubnetRequest) (ocicore.UpdateSubnetResponse, error) {
						captured = req
						return ocicore.UpdateSubnetResponse{}, nil
					},
				}
				mgr := subnetMgrWithFake(fake)
				s := &ociv1beta1.OciSubnet{}
				s.Status.OsokStatus.Ocid = "ocid1.subnet.oc1..tags"
				s.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateSubnet(context.Background(), s))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
		{
			name: "internet-gateway",
			run: func(t *testing.T) {
				var captured ocicore.UpdateInternetGatewayRequest
				fake := &fakeVirtualNetworkClient{
					getInternetGatewayFn: func(_ context.Context, _ ocicore.GetInternetGatewayRequest) (ocicore.GetInternetGatewayResponse, error) {
						return ocicore.GetInternetGatewayResponse{InternetGateway: ocicore.InternetGateway{Id: common.String("ocid1.igw.oc1..tags"), DefinedTags: map[string]map[string]interface{}{"ops": {"env": "dev"}}}}, nil
					},
					updateInternetGatewayFn: func(_ context.Context, req ocicore.UpdateInternetGatewayRequest) (ocicore.UpdateInternetGatewayResponse, error) {
						captured = req
						return ocicore.UpdateInternetGatewayResponse{}, nil
					},
				}
				mgr := igwMgrWithFake(fake)
				igw := &ociv1beta1.OciInternetGateway{}
				igw.Status.OsokStatus.Ocid = "ocid1.igw.oc1..tags"
				igw.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateInternetGateway(context.Background(), igw))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
		{
			name: "nat-gateway",
			run: func(t *testing.T) {
				var captured ocicore.UpdateNatGatewayRequest
				fake := &fakeVirtualNetworkClient{
					getNatGatewayFn: func(_ context.Context, _ ocicore.GetNatGatewayRequest) (ocicore.GetNatGatewayResponse, error) {
						return ocicore.GetNatGatewayResponse{NatGateway: ocicore.NatGateway{Id: common.String("ocid1.nat.oc1..tags"), DefinedTags: map[string]map[string]interface{}{"ops": {"env": "dev"}}}}, nil
					},
					updateNatGatewayFn: func(_ context.Context, req ocicore.UpdateNatGatewayRequest) (ocicore.UpdateNatGatewayResponse, error) {
						captured = req
						return ocicore.UpdateNatGatewayResponse{}, nil
					},
				}
				mgr := natMgrWithFake(fake)
				nat := &ociv1beta1.OciNatGateway{}
				nat.Status.OsokStatus.Ocid = "ocid1.nat.oc1..tags"
				nat.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateNatGateway(context.Background(), nat))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
		{
			name: "service-gateway",
			run: func(t *testing.T) {
				var captured ocicore.UpdateServiceGatewayRequest
				fake := &fakeVirtualNetworkClient{
					getServiceGatewayFn: func(_ context.Context, _ ocicore.GetServiceGatewayRequest) (ocicore.GetServiceGatewayResponse, error) {
						return ocicore.GetServiceGatewayResponse{ServiceGateway: ocicore.ServiceGateway{Id: common.String("ocid1.sgw.oc1..tags"), DefinedTags: map[string]map[string]interface{}{"ops": {"env": "dev"}}}}, nil
					},
					updateServiceGatewayFn: func(_ context.Context, req ocicore.UpdateServiceGatewayRequest) (ocicore.UpdateServiceGatewayResponse, error) {
						captured = req
						return ocicore.UpdateServiceGatewayResponse{}, nil
					},
				}
				mgr := sgwMgrWithFake(fake)
				sgw := &ociv1beta1.OciServiceGateway{}
				sgw.Status.OsokStatus.Ocid = "ocid1.sgw.oc1..tags"
				sgw.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateServiceGateway(context.Background(), sgw))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
		{
			name: "drg",
			run: func(t *testing.T) {
				var captured ocicore.UpdateDrgRequest
				fake := &fakeVirtualNetworkClient{
					getDrgFn: func(_ context.Context, _ ocicore.GetDrgRequest) (ocicore.GetDrgResponse, error) {
						return ocicore.GetDrgResponse{Drg: ocicore.Drg{Id: common.String("ocid1.drg.oc1..tags"), DefinedTags: map[string]map[string]interface{}{"ops": {"env": "dev"}}}}, nil
					},
					updateDrgFn: func(_ context.Context, req ocicore.UpdateDrgRequest) (ocicore.UpdateDrgResponse, error) {
						captured = req
						return ocicore.UpdateDrgResponse{}, nil
					},
				}
				mgr := drgMgrWithFake(fake)
				drg := &ociv1beta1.OciDrg{}
				drg.Status.OsokStatus.Ocid = "ocid1.drg.oc1..tags"
				drg.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateDrg(context.Background(), drg))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
		{
			name: "network-security-group",
			run: func(t *testing.T) {
				var captured ocicore.UpdateNetworkSecurityGroupRequest
				fake := &fakeVirtualNetworkClient{
					getNetworkSecurityGroupFn: func(_ context.Context, _ ocicore.GetNetworkSecurityGroupRequest) (ocicore.GetNetworkSecurityGroupResponse, error) {
						return ocicore.GetNetworkSecurityGroupResponse{NetworkSecurityGroup: ocicore.NetworkSecurityGroup{Id: common.String("ocid1.nsg.oc1..tags"), DefinedTags: map[string]map[string]interface{}{"ops": {"env": "dev"}}}}, nil
					},
					updateNetworkSecurityGroupFn: func(_ context.Context, req ocicore.UpdateNetworkSecurityGroupRequest) (ocicore.UpdateNetworkSecurityGroupResponse, error) {
						captured = req
						return ocicore.UpdateNetworkSecurityGroupResponse{}, nil
					},
				}
				mgr := nsgMgrWithFake(fake)
				nsg := &ociv1beta1.OciNetworkSecurityGroup{}
				nsg.Status.OsokStatus.Ocid = "ocid1.nsg.oc1..tags"
				nsg.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateNetworkSecurityGroup(context.Background(), nsg))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
		{
			name: "security-list",
			run: func(t *testing.T) {
				var captured ocicore.UpdateSecurityListRequest
				fake := &fakeVirtualNetworkClient{
					updateSecurityListFn: func(_ context.Context, req ocicore.UpdateSecurityListRequest) (ocicore.UpdateSecurityListResponse, error) {
						captured = req
						return ocicore.UpdateSecurityListResponse{}, nil
					},
				}
				mgr := securityListMgrWithFake(fake)
				sl := &ociv1beta1.OciSecurityList{}
				sl.Status.OsokStatus.Ocid = "ocid1.sl.oc1..tags"
				sl.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateSecurityList(context.Background(), sl))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
		{
			name: "route-table",
			run: func(t *testing.T) {
				var captured ocicore.UpdateRouteTableRequest
				fake := &fakeVirtualNetworkClient{
					updateRouteTableFn: func(_ context.Context, req ocicore.UpdateRouteTableRequest) (ocicore.UpdateRouteTableResponse, error) {
						captured = req
						return ocicore.UpdateRouteTableResponse{}, nil
					},
				}
				mgr := routeTableMgrWithFake(fake)
				rt := &ociv1beta1.OciRouteTable{}
				rt.Status.OsokStatus.Ocid = "ocid1.rt.oc1..tags"
				rt.Spec.DefinedTags = desiredTags
				assert.NoError(t, mgr.UpdateRouteTable(context.Background(), rt))
				assert.Equal(t, expectedTags, captured.DefinedTags)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}
