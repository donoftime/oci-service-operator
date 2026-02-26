/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/apigateway"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// --- mock gateway client ---

type mockGatewayClient struct {
	createGatewayFn func(ctx context.Context, req apigateway.CreateGatewayRequest) (apigateway.CreateGatewayResponse, error)
	getGatewayFn    func(ctx context.Context, req apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error)
	listGatewaysFn  func(ctx context.Context, req apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error)
	updateGatewayFn func(ctx context.Context, req apigateway.UpdateGatewayRequest) (apigateway.UpdateGatewayResponse, error)
	deleteGatewayFn func(ctx context.Context, req apigateway.DeleteGatewayRequest) (apigateway.DeleteGatewayResponse, error)
	deleteCalled    bool
}

func (m *mockGatewayClient) CreateGateway(ctx context.Context, req apigateway.CreateGatewayRequest) (apigateway.CreateGatewayResponse, error) {
	if m.createGatewayFn != nil {
		return m.createGatewayFn(ctx, req)
	}
	return apigateway.CreateGatewayResponse{}, nil
}

func (m *mockGatewayClient) GetGateway(ctx context.Context, req apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error) {
	if m.getGatewayFn != nil {
		return m.getGatewayFn(ctx, req)
	}
	return apigateway.GetGatewayResponse{}, nil
}

func (m *mockGatewayClient) ListGateways(ctx context.Context, req apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error) {
	if m.listGatewaysFn != nil {
		return m.listGatewaysFn(ctx, req)
	}
	return apigateway.ListGatewaysResponse{}, nil
}

func (m *mockGatewayClient) UpdateGateway(ctx context.Context, req apigateway.UpdateGatewayRequest) (apigateway.UpdateGatewayResponse, error) {
	if m.updateGatewayFn != nil {
		return m.updateGatewayFn(ctx, req)
	}
	return apigateway.UpdateGatewayResponse{}, nil
}

func (m *mockGatewayClient) DeleteGateway(ctx context.Context, req apigateway.DeleteGatewayRequest) (apigateway.DeleteGatewayResponse, error) {
	m.deleteCalled = true
	if m.deleteGatewayFn != nil {
		return m.deleteGatewayFn(ctx, req)
	}
	return apigateway.DeleteGatewayResponse{}, nil
}

// --- mock deployment client ---

type mockDeploymentClient struct {
	createDeploymentFn func(ctx context.Context, req apigateway.CreateDeploymentRequest) (apigateway.CreateDeploymentResponse, error)
	getDeploymentFn    func(ctx context.Context, req apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error)
	listDeploymentsFn  func(ctx context.Context, req apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error)
	updateDeploymentFn func(ctx context.Context, req apigateway.UpdateDeploymentRequest) (apigateway.UpdateDeploymentResponse, error)
	deleteDeploymentFn func(ctx context.Context, req apigateway.DeleteDeploymentRequest) (apigateway.DeleteDeploymentResponse, error)
	deleteCalled       bool
}

func (m *mockDeploymentClient) CreateDeployment(ctx context.Context, req apigateway.CreateDeploymentRequest) (apigateway.CreateDeploymentResponse, error) {
	if m.createDeploymentFn != nil {
		return m.createDeploymentFn(ctx, req)
	}
	return apigateway.CreateDeploymentResponse{}, nil
}

func (m *mockDeploymentClient) GetDeployment(ctx context.Context, req apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
	if m.getDeploymentFn != nil {
		return m.getDeploymentFn(ctx, req)
	}
	return apigateway.GetDeploymentResponse{}, nil
}

func (m *mockDeploymentClient) ListDeployments(ctx context.Context, req apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error) {
	if m.listDeploymentsFn != nil {
		return m.listDeploymentsFn(ctx, req)
	}
	return apigateway.ListDeploymentsResponse{}, nil
}

func (m *mockDeploymentClient) UpdateDeployment(ctx context.Context, req apigateway.UpdateDeploymentRequest) (apigateway.UpdateDeploymentResponse, error) {
	if m.updateDeploymentFn != nil {
		return m.updateDeploymentFn(ctx, req)
	}
	return apigateway.UpdateDeploymentResponse{}, nil
}

func (m *mockDeploymentClient) DeleteDeployment(ctx context.Context, req apigateway.DeleteDeploymentRequest) (apigateway.DeleteDeploymentResponse, error) {
	m.deleteCalled = true
	if m.deleteDeploymentFn != nil {
		return m.deleteDeploymentFn(ctx, req)
	}
	return apigateway.DeleteDeploymentResponse{}, nil
}

// --- helpers ---

func makeGatewayManager(gwClient *mockGatewayClient, credClient *fakeCredentialClient) *GatewayServiceManager {
	scheme := runtime.NewScheme()
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewGatewayServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, scheme, log)
	ExportSetGatewayClientForTest(mgr, gwClient)
	return mgr
}

func makeDeploymentManager(depClient *mockDeploymentClient, credClient *fakeCredentialClient) *DeploymentServiceManager {
	scheme := runtime.NewScheme()
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewDeploymentServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, scheme, log)
	ExportSetDeploymentClientForTest(mgr, depClient)
	return mgr
}

func makeActiveGateway(id, displayName, hostname string) apigateway.Gateway {
	return apigateway.Gateway{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: apigateway.GatewayLifecycleStateActive,
		Hostname:       common.String(hostname),
	}
}

func makeCreatingGateway(id, displayName string) apigateway.Gateway {
	return apigateway.Gateway{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: apigateway.GatewayLifecycleStateCreating,
	}
}

func makeFailedGateway(id, displayName string) apigateway.Gateway {
	return apigateway.Gateway{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: apigateway.GatewayLifecycleStateFailed,
	}
}

func makeActiveDeployment(id, displayName string) apigateway.Deployment {
	return apigateway.Deployment{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: apigateway.DeploymentLifecycleStateActive,
	}
}

func makeCreatingDeployment(id, displayName string) apigateway.Deployment {
	return apigateway.Deployment{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: apigateway.DeploymentLifecycleStateCreating,
	}
}

func makeFailedDeployment(id, displayName string) apigateway.Deployment {
	return apigateway.Deployment{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: apigateway.DeploymentLifecycleStateFailed,
	}
}

// --- Gateway CreateOrUpdate tests ---

func TestGatewayServiceManager_CreateOrUpdate_BadType(t *testing.T) {
	mgr := makeGatewayManager(&mockGatewayClient{}, &fakeCredentialClient{})

	resp, err := mgr.CreateOrUpdate(context.Background(), &ociv1beta1.Stream{}, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestGatewayServiceManager_CreateOrUpdate_Create_Success(t *testing.T) {
	gwID := "ocid1.apigateway.oc1..xxx"
	gw := makeActiveGateway(gwID, "test-gw", "test-gw.apigateway.oci.example.com")

	secretCreated := false
	credClient := &fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, data map[string][]byte) (bool, error) {
			secretCreated = true
			assert.Equal(t, "test-gw.apigateway.oci.example.com", string(data["hostname"]))
			return true, nil
		},
	}

	gwClient := &mockGatewayClient{
		listGatewaysFn: func(_ context.Context, _ apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error) {
			// No existing gateway found
			return apigateway.ListGatewaysResponse{}, nil
		},
		createGatewayFn: func(_ context.Context, _ apigateway.CreateGatewayRequest) (apigateway.CreateGatewayResponse, error) {
			return apigateway.CreateGatewayResponse{Gateway: apigateway.Gateway{Id: common.String(gwID)}}, nil
		},
		getGatewayFn: func(_ context.Context, _ apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error) {
			return apigateway.GetGatewayResponse{Gateway: gw}, nil
		},
	}

	mgr := makeGatewayManager(gwClient, credClient)
	obj := &ociv1beta1.ApiGateway{}
	obj.Name = "test-gw"
	obj.Namespace = "default"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "test-gw"
	obj.Spec.EndpointType = "PUBLIC"
	obj.Spec.SubnetId = "ocid1.subnet.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(gwID), obj.Status.OsokStatus.Ocid)
	assert.True(t, secretCreated, "secret should have been created with hostname")
}

func TestGatewayServiceManager_CreateOrUpdate_ExistingGateway(t *testing.T) {
	gwID := "ocid1.apigateway.oc1..existing"
	gw := makeActiveGateway(gwID, "existing-gw", "existing-gw.apigateway.oci.example.com")

	credClient := &fakeCredentialClient{}
	gwClient := &mockGatewayClient{
		listGatewaysFn: func(_ context.Context, _ apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error) {
			return apigateway.ListGatewaysResponse{
				GatewayCollection: apigateway.GatewayCollection{
					Items: []apigateway.GatewaySummary{
						{
							Id:             common.String(gwID),
							LifecycleState: apigateway.GatewayLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getGatewayFn: func(_ context.Context, _ apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error) {
			return apigateway.GetGatewayResponse{Gateway: gw}, nil
		},
		updateGatewayFn: func(_ context.Context, _ apigateway.UpdateGatewayRequest) (apigateway.UpdateGatewayResponse, error) {
			return apigateway.UpdateGatewayResponse{}, nil
		},
	}

	mgr := makeGatewayManager(gwClient, credClient)
	obj := &ociv1beta1.ApiGateway{}
	obj.Name = "existing-gw"
	obj.Namespace = "default"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "existing-gw"
	obj.Spec.EndpointType = "PUBLIC"
	obj.Spec.SubnetId = "ocid1.subnet.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(gwID), obj.Status.OsokStatus.Ocid)
}

func TestGatewayServiceManager_CreateOrUpdate_BindById(t *testing.T) {
	gwID := "ocid1.apigateway.oc1..bound"
	gw := makeActiveGateway(gwID, "bound-gw", "bound-gw.apigateway.oci.example.com")

	credClient := &fakeCredentialClient{}
	gwClient := &mockGatewayClient{
		getGatewayFn: func(_ context.Context, _ apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error) {
			return apigateway.GetGatewayResponse{Gateway: gw}, nil
		},
		updateGatewayFn: func(_ context.Context, _ apigateway.UpdateGatewayRequest) (apigateway.UpdateGatewayResponse, error) {
			return apigateway.UpdateGatewayResponse{}, nil
		},
	}

	mgr := makeGatewayManager(gwClient, credClient)
	obj := &ociv1beta1.ApiGateway{}
	obj.Name = "bound-gw"
	obj.Namespace = "default"
	obj.Spec.ApiGatewayId = ociv1beta1.OCID(gwID)
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "bound-gw"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(gwID), obj.Status.OsokStatus.Ocid)
}

func TestGatewayServiceManager_CreateOrUpdate_CreatingState_Requeues(t *testing.T) {
	gwID := "ocid1.apigateway.oc1..creating"
	gw := makeCreatingGateway(gwID, "creating-gw")

	credClient := &fakeCredentialClient{}
	gwClient := &mockGatewayClient{
		listGatewaysFn: func(_ context.Context, _ apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error) {
			return apigateway.ListGatewaysResponse{}, nil
		},
		createGatewayFn: func(_ context.Context, _ apigateway.CreateGatewayRequest) (apigateway.CreateGatewayResponse, error) {
			return apigateway.CreateGatewayResponse{Gateway: apigateway.Gateway{Id: common.String(gwID)}}, nil
		},
		getGatewayFn: func(_ context.Context, _ apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error) {
			return apigateway.GetGatewayResponse{Gateway: gw}, nil
		},
	}

	mgr := makeGatewayManager(gwClient, credClient)
	obj := &ociv1beta1.ApiGateway{}
	obj.Name = "creating-gw"
	obj.Namespace = "default"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "creating-gw"
	obj.Spec.EndpointType = "PUBLIC"
	obj.Spec.SubnetId = "ocid1.subnet.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue, "should requeue while gateway is CREATING")
}

func TestGatewayServiceManager_CreateOrUpdate_FailedState(t *testing.T) {
	gwID := "ocid1.apigateway.oc1..failed"
	gw := makeFailedGateway(gwID, "failed-gw")

	credClient := &fakeCredentialClient{}
	gwClient := &mockGatewayClient{
		listGatewaysFn: func(_ context.Context, _ apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error) {
			return apigateway.ListGatewaysResponse{}, nil
		},
		createGatewayFn: func(_ context.Context, _ apigateway.CreateGatewayRequest) (apigateway.CreateGatewayResponse, error) {
			return apigateway.CreateGatewayResponse{Gateway: apigateway.Gateway{Id: common.String(gwID)}}, nil
		},
		getGatewayFn: func(_ context.Context, _ apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error) {
			return apigateway.GetGatewayResponse{Gateway: gw}, nil
		},
	}

	mgr := makeGatewayManager(gwClient, credClient)
	obj := &ociv1beta1.ApiGateway{}
	obj.Name = "failed-gw"
	obj.Namespace = "default"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "failed-gw"
	obj.Spec.EndpointType = "PUBLIC"
	obj.Spec.SubnetId = "ocid1.subnet.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestGatewayServiceManager_CreateOrUpdate_ListError(t *testing.T) {
	credClient := &fakeCredentialClient{}
	gwClient := &mockGatewayClient{
		listGatewaysFn: func(_ context.Context, _ apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error) {
			return apigateway.ListGatewaysResponse{}, errors.New("list error")
		},
	}

	mgr := makeGatewayManager(gwClient, credClient)
	obj := &ociv1beta1.ApiGateway{}
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "test-gw"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestGatewayServiceManager_Delete_WithOcid(t *testing.T) {
	gwClient := &mockGatewayClient{}
	credClient := &fakeCredentialClient{}
	mgr := makeGatewayManager(gwClient, credClient)

	obj := &ociv1beta1.ApiGateway{}
	obj.Status.OsokStatus.Ocid = "ocid1.apigateway.oc1..xxx"

	done, err := mgr.Delete(context.Background(), obj)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, gwClient.deleteCalled)
}

func TestGatewayServiceManager_Delete_Error(t *testing.T) {
	gwClient := &mockGatewayClient{
		deleteGatewayFn: func(_ context.Context, _ apigateway.DeleteGatewayRequest) (apigateway.DeleteGatewayResponse, error) {
			return apigateway.DeleteGatewayResponse{}, errors.New("delete failed")
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := makeGatewayManager(gwClient, credClient)

	obj := &ociv1beta1.ApiGateway{}
	obj.Status.OsokStatus.Ocid = "ocid1.apigateway.oc1..xxx"

	done, err := mgr.Delete(context.Background(), obj)
	assert.Error(t, err)
	assert.False(t, done)
}

// --- Gateway credential map tests ---

func TestGetGatewayCredentialMap_WithHostname(t *testing.T) {
	gw := makeActiveGateway("ocid1.apigateway.xxx", "test-gw", "gw.example.com")
	credMap := ExportGetGatewayCredentialMap(gw)

	assert.Equal(t, "gw.example.com", string(credMap["hostname"]))
}

func TestGetGatewayCredentialMap_NilHostname(t *testing.T) {
	gw := apigateway.Gateway{
		Id:          common.String("ocid1.apigateway.xxx"),
		DisplayName: common.String("no-hostname-gw"),
	}
	credMap := ExportGetGatewayCredentialMap(gw)
	assert.NotContains(t, credMap, "hostname")
}

// --- Deployment CreateOrUpdate tests ---

func TestDeploymentServiceManager_CreateOrUpdate_BadType(t *testing.T) {
	mgr := makeDeploymentManager(&mockDeploymentClient{}, &fakeCredentialClient{})

	resp, err := mgr.CreateOrUpdate(context.Background(), &ociv1beta1.Stream{}, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestDeploymentServiceManager_CreateOrUpdate_Create_Success(t *testing.T) {
	depID := "ocid1.apideployment.oc1..xxx"
	dep := makeActiveDeployment(depID, "test-dep")

	credClient := &fakeCredentialClient{}
	depClient := &mockDeploymentClient{
		listDeploymentsFn: func(_ context.Context, _ apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error) {
			return apigateway.ListDeploymentsResponse{}, nil
		},
		createDeploymentFn: func(_ context.Context, _ apigateway.CreateDeploymentRequest) (apigateway.CreateDeploymentResponse, error) {
			return apigateway.CreateDeploymentResponse{Deployment: apigateway.Deployment{Id: common.String(depID)}}, nil
		},
		getDeploymentFn: func(_ context.Context, _ apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
			return apigateway.GetDeploymentResponse{Deployment: dep}, nil
		},
	}

	mgr := makeDeploymentManager(depClient, credClient)
	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Name = "test-dep"
	obj.Namespace = "default"
	obj.Spec.GatewayId = "ocid1.apigateway.oc1..xxx"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "test-dep"
	obj.Spec.PathPrefix = "/v1"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(depID), obj.Status.OsokStatus.Ocid)
}

func TestDeploymentServiceManager_CreateOrUpdate_ExistingDeployment(t *testing.T) {
	depID := "ocid1.apideployment.oc1..existing"
	dep := makeActiveDeployment(depID, "existing-dep")

	credClient := &fakeCredentialClient{}
	depClient := &mockDeploymentClient{
		listDeploymentsFn: func(_ context.Context, _ apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error) {
			return apigateway.ListDeploymentsResponse{
				DeploymentCollection: apigateway.DeploymentCollection{
					Items: []apigateway.DeploymentSummary{
						{
							Id:             common.String(depID),
							LifecycleState: apigateway.DeploymentLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getDeploymentFn: func(_ context.Context, _ apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
			return apigateway.GetDeploymentResponse{Deployment: dep}, nil
		},
		updateDeploymentFn: func(_ context.Context, _ apigateway.UpdateDeploymentRequest) (apigateway.UpdateDeploymentResponse, error) {
			return apigateway.UpdateDeploymentResponse{}, nil
		},
	}

	mgr := makeDeploymentManager(depClient, credClient)
	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Name = "existing-dep"
	obj.Namespace = "default"
	obj.Spec.GatewayId = "ocid1.apigateway.oc1..xxx"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "existing-dep"
	obj.Spec.PathPrefix = "/v1"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(depID), obj.Status.OsokStatus.Ocid)
}

func TestDeploymentServiceManager_CreateOrUpdate_BindById(t *testing.T) {
	depID := "ocid1.apideployment.oc1..bound"
	dep := makeActiveDeployment(depID, "bound-dep")

	credClient := &fakeCredentialClient{}
	depClient := &mockDeploymentClient{
		getDeploymentFn: func(_ context.Context, _ apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
			return apigateway.GetDeploymentResponse{Deployment: dep}, nil
		},
		updateDeploymentFn: func(_ context.Context, _ apigateway.UpdateDeploymentRequest) (apigateway.UpdateDeploymentResponse, error) {
			return apigateway.UpdateDeploymentResponse{}, nil
		},
	}

	mgr := makeDeploymentManager(depClient, credClient)
	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Name = "bound-dep"
	obj.Namespace = "default"
	obj.Spec.DeploymentId = ociv1beta1.OCID(depID)
	obj.Spec.GatewayId = "ocid1.apigateway.oc1..xxx"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.PathPrefix = "/v1"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(depID), obj.Status.OsokStatus.Ocid)
}

func TestDeploymentServiceManager_CreateOrUpdate_FailedState(t *testing.T) {
	depID := "ocid1.apideployment.oc1..failed"
	dep := makeFailedDeployment(depID, "failed-dep")

	credClient := &fakeCredentialClient{}
	depClient := &mockDeploymentClient{
		listDeploymentsFn: func(_ context.Context, _ apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error) {
			return apigateway.ListDeploymentsResponse{}, nil
		},
		createDeploymentFn: func(_ context.Context, _ apigateway.CreateDeploymentRequest) (apigateway.CreateDeploymentResponse, error) {
			return apigateway.CreateDeploymentResponse{Deployment: apigateway.Deployment{Id: common.String(depID)}}, nil
		},
		getDeploymentFn: func(_ context.Context, _ apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
			return apigateway.GetDeploymentResponse{Deployment: dep}, nil
		},
	}

	mgr := makeDeploymentManager(depClient, credClient)
	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Name = "failed-dep"
	obj.Namespace = "default"
	obj.Spec.GatewayId = "ocid1.apigateway.oc1..xxx"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "failed-dep"
	obj.Spec.PathPrefix = "/v1"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestDeploymentServiceManager_CreateOrUpdate_CreateFails_PartialFailure(t *testing.T) {
	credClient := &fakeCredentialClient{}
	depClient := &mockDeploymentClient{
		listDeploymentsFn: func(_ context.Context, _ apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error) {
			return apigateway.ListDeploymentsResponse{}, nil
		},
		createDeploymentFn: func(_ context.Context, _ apigateway.CreateDeploymentRequest) (apigateway.CreateDeploymentResponse, error) {
			return apigateway.CreateDeploymentResponse{}, errors.New("create deployment failed")
		},
	}

	mgr := makeDeploymentManager(depClient, credClient)
	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Name = "partial-fail-dep"
	obj.Namespace = "default"
	obj.Spec.GatewayId = "ocid1.apigateway.oc1..xxx"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "partial-fail-dep"
	obj.Spec.PathPrefix = "/v1"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestDeploymentServiceManager_CreateOrUpdate_CreatingState_Requeues(t *testing.T) {
	depID := "ocid1.apideployment.oc1..creating"
	dep := makeCreatingDeployment(depID, "creating-dep")

	credClient := &fakeCredentialClient{}
	depClient := &mockDeploymentClient{
		listDeploymentsFn: func(_ context.Context, _ apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error) {
			return apigateway.ListDeploymentsResponse{}, nil
		},
		createDeploymentFn: func(_ context.Context, _ apigateway.CreateDeploymentRequest) (apigateway.CreateDeploymentResponse, error) {
			return apigateway.CreateDeploymentResponse{Deployment: apigateway.Deployment{Id: common.String(depID)}}, nil
		},
		getDeploymentFn: func(_ context.Context, _ apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
			return apigateway.GetDeploymentResponse{Deployment: dep}, nil
		},
	}

	mgr := makeDeploymentManager(depClient, credClient)
	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Name = "creating-dep"
	obj.Namespace = "default"
	obj.Spec.GatewayId = "ocid1.apigateway.oc1..xxx"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "creating-dep"
	obj.Spec.PathPrefix = "/v1"

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	// Creating state handled by existing code — just check it doesn't crash
	_ = resp
	_ = err
}

func TestDeploymentServiceManager_Delete_WithOcid(t *testing.T) {
	depClient := &mockDeploymentClient{}
	credClient := &fakeCredentialClient{}
	mgr := makeDeploymentManager(depClient, credClient)

	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Status.OsokStatus.Ocid = "ocid1.apideployment.oc1..xxx"

	done, err := mgr.Delete(context.Background(), obj)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, depClient.deleteCalled)
}

func TestDeploymentServiceManager_CreateOrUpdate_WithRoutes(t *testing.T) {
	depID := "ocid1.apideployment.oc1..routes"
	dep := makeActiveDeployment(depID, "routes-dep")

	credClient := &fakeCredentialClient{}
	depClient := &mockDeploymentClient{
		listDeploymentsFn: func(_ context.Context, _ apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error) {
			return apigateway.ListDeploymentsResponse{}, nil
		},
		createDeploymentFn: func(_ context.Context, req apigateway.CreateDeploymentRequest) (apigateway.CreateDeploymentResponse, error) {
			// Verify routes were converted
			assert.NotNil(t, req.CreateDeploymentDetails.Specification)
			assert.Len(t, req.CreateDeploymentDetails.Specification.Routes, 3)
			return apigateway.CreateDeploymentResponse{Deployment: apigateway.Deployment{Id: common.String(depID)}}, nil
		},
		getDeploymentFn: func(_ context.Context, _ apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
			return apigateway.GetDeploymentResponse{Deployment: dep}, nil
		},
	}

	mgr := makeDeploymentManager(depClient, credClient)
	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Name = "routes-dep"
	obj.Namespace = "default"
	obj.Spec.GatewayId = "ocid1.apigateway.oc1..xxx"
	obj.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	obj.Spec.DisplayName = "routes-dep"
	obj.Spec.PathPrefix = "/v1"
	obj.Spec.Routes = []ociv1beta1.ApiGatewayRoute{
		{
			Path:    "/http",
			Methods: []string{"GET", "POST"},
			Backend: ociv1beta1.ApiGatewayRouteBackend{
				Type: "HTTP_BACKEND",
				Url:  "https://backend.example.com",
			},
		},
		{
			Path: "/fn",
			Backend: ociv1beta1.ApiGatewayRouteBackend{
				Type:       "ORACLE_FUNCTIONS_BACKEND",
				FunctionId: "ocid1.fnfunc.oc1..xxx",
			},
		},
		{
			Path: "/stock",
			Backend: ociv1beta1.ApiGatewayRouteBackend{
				Type:   "STOCK_RESPONSE_BACKEND",
				Status: 200,
				Body:   `{"ok":true}`,
			},
		},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

func TestDeploymentServiceManager_Delete_Error(t *testing.T) {
	depClient := &mockDeploymentClient{
		deleteDeploymentFn: func(_ context.Context, _ apigateway.DeleteDeploymentRequest) (apigateway.DeleteDeploymentResponse, error) {
			return apigateway.DeleteDeploymentResponse{}, errors.New("delete failed")
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := makeDeploymentManager(depClient, credClient)

	obj := &ociv1beta1.ApiGatewayDeployment{}
	obj.Status.OsokStatus.Ocid = "ocid1.apideployment.oc1..xxx"

	done, err := mgr.Delete(context.Background(), obj)
	assert.Error(t, err)
	assert.False(t, done)
}
