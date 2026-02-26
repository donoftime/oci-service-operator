/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/functions"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct {
	createSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	deleteSecretFn func(ctx context.Context, name, ns string) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, ns)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	return true, nil
}

// --- FunctionsApplication tests ---

// TestFunctionsApplication_Delete_NoOcid verifies deletion with no OCID set is a no-op.
func TestFunctionsApplication_Delete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	app := &ociv1beta1.FunctionsApplication{}
	app.Name = "test-app"
	app.Namespace = "default"

	done, err := mgr.Delete(context.Background(), app)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestFunctionsApplication_GetCrdStatus verifies status extraction.
func TestFunctionsApplication_GetCrdStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	app := &ociv1beta1.FunctionsApplication{}
	app.Status.OsokStatus.Ocid = "ocid1.fnapp.oc1..xxx"

	status, err := mgr.GetCrdStatus(app)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.fnapp.oc1..xxx"), status.Ocid)
}

// TestFunctionsApplication_GetCrdStatus_WrongType verifies convert fails on wrong type.
func TestFunctionsApplication_GetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestFunctionsApplication_CreateOrUpdate_BadType verifies CreateOrUpdate rejects non-FunctionsApplication objects.
func TestFunctionsApplication_CreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// --- FunctionsFunction tests ---

// TestFunctionsFunction_Delete_NoOcid verifies deletion with no OCID set is a no-op.
func TestFunctionsFunction_Delete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Name = "test-fn"
	fn.Namespace = "default"

	done, err := mgr.Delete(context.Background(), fn)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called when OCID is empty")
}

// TestFunctionsFunction_GetCrdStatus verifies status extraction.
func TestFunctionsFunction_GetCrdStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Status.OsokStatus.Ocid = "ocid1.fnfunc.oc1..xxx"

	status, err := mgr.GetCrdStatus(fn)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.fnfunc.oc1..xxx"), status.Ocid)
}

// TestFunctionsFunction_GetCrdStatus_WrongType verifies convert fails on wrong type.
func TestFunctionsFunction_GetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestFunctionsFunction_CreateOrUpdate_BadType verifies CreateOrUpdate rejects non-FunctionsFunction objects.
func TestFunctionsFunction_CreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestGetFunctionCredentialMap verifies the secret credential map is built correctly.
func TestGetFunctionCredentialMap(t *testing.T) {
	fn := ocifunctions.Function{
		Id:             common.String("ocid1.fnfunc.oc1..xxx"),
		DisplayName:    common.String("test-fn"),
		InvokeEndpoint: common.String("https://xyz.functions.oci.example.com/20181201/functions/ocid1.fnfunc.oc1..xxx/actions/invoke"),
	}

	credMap := GetFunctionCredentialMapForTest(fn)
	assert.Equal(t, "ocid1.fnfunc.oc1..xxx", string(credMap["functionId"]))
	assert.Contains(t, string(credMap["invokeEndpoint"]), "invoke")
}

// TestGetFunctionCredentialMap_NilFields verifies nil pointer fields are handled gracefully.
func TestGetFunctionCredentialMap_NilFields(t *testing.T) {
	fn := ocifunctions.Function{
		Id: common.String("ocid1.fnfunc.oc1..xxx"),
	}
	credMap := GetFunctionCredentialMapForTest(fn)
	assert.NotContains(t, credMap, "invokeEndpoint")
	assert.Equal(t, "ocid1.fnfunc.oc1..xxx", string(credMap["functionId"]))
}

// --- Mock OCI client ---

// mockFunctionsClient implements FunctionsManagementClientInterface for testing.
// Each method dispatches to a configurable function field; unset fields return zero values with no error.
type mockFunctionsClient struct {
	createApplicationFn func(ctx context.Context, req ocifunctions.CreateApplicationRequest) (ocifunctions.CreateApplicationResponse, error)
	getApplicationFn    func(ctx context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error)
	listApplicationsFn  func(ctx context.Context, req ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error)
	updateApplicationFn func(ctx context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error)
	deleteApplicationFn func(ctx context.Context, req ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error)
	createFunctionFn    func(ctx context.Context, req ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error)
	getFunctionFn       func(ctx context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error)
	listFunctionsFn     func(ctx context.Context, req ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error)
	updateFunctionFn    func(ctx context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error)
	deleteFunctionFn    func(ctx context.Context, req ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error)
}

func (m *mockFunctionsClient) CreateApplication(ctx context.Context, req ocifunctions.CreateApplicationRequest) (ocifunctions.CreateApplicationResponse, error) {
	if m.createApplicationFn != nil {
		return m.createApplicationFn(ctx, req)
	}
	return ocifunctions.CreateApplicationResponse{}, nil
}

func (m *mockFunctionsClient) GetApplication(ctx context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
	if m.getApplicationFn != nil {
		return m.getApplicationFn(ctx, req)
	}
	return ocifunctions.GetApplicationResponse{}, nil
}

func (m *mockFunctionsClient) ListApplications(ctx context.Context, req ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
	if m.listApplicationsFn != nil {
		return m.listApplicationsFn(ctx, req)
	}
	return ocifunctions.ListApplicationsResponse{}, nil
}

func (m *mockFunctionsClient) UpdateApplication(ctx context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
	if m.updateApplicationFn != nil {
		return m.updateApplicationFn(ctx, req)
	}
	return ocifunctions.UpdateApplicationResponse{}, nil
}

func (m *mockFunctionsClient) DeleteApplication(ctx context.Context, req ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error) {
	if m.deleteApplicationFn != nil {
		return m.deleteApplicationFn(ctx, req)
	}
	return ocifunctions.DeleteApplicationResponse{}, nil
}

func (m *mockFunctionsClient) CreateFunction(ctx context.Context, req ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error) {
	if m.createFunctionFn != nil {
		return m.createFunctionFn(ctx, req)
	}
	return ocifunctions.CreateFunctionResponse{}, nil
}

func (m *mockFunctionsClient) GetFunction(ctx context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
	if m.getFunctionFn != nil {
		return m.getFunctionFn(ctx, req)
	}
	return ocifunctions.GetFunctionResponse{}, nil
}

func (m *mockFunctionsClient) ListFunctions(ctx context.Context, req ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
	if m.listFunctionsFn != nil {
		return m.listFunctionsFn(ctx, req)
	}
	return ocifunctions.ListFunctionsResponse{}, nil
}

func (m *mockFunctionsClient) UpdateFunction(ctx context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
	if m.updateFunctionFn != nil {
		return m.updateFunctionFn(ctx, req)
	}
	return ocifunctions.UpdateFunctionResponse{}, nil
}

func (m *mockFunctionsClient) DeleteFunction(ctx context.Context, req ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
	if m.deleteFunctionFn != nil {
		return m.deleteFunctionFn(ctx, req)
	}
	return ocifunctions.DeleteFunctionResponse{}, nil
}

// --- Test helpers ---

func newAppMgr(t *testing.T, ociClient *mockFunctionsClient) *FunctionsApplicationServiceManager {
	t.Helper()
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		&fakeCredentialClient{}, nil, log)
	if ociClient != nil {
		ExportSetApplicationClientForTest(mgr, ociClient)
	}
	return mgr
}

func newFuncMgr(t *testing.T, credClient *fakeCredentialClient, ociClient *mockFunctionsClient) *FunctionsFunctionServiceManager {
	t.Helper()
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	if credClient == nil {
		credClient = &fakeCredentialClient{}
	}
	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)
	if ociClient != nil {
		ExportSetFunctionClientForTest(mgr, ociClient)
	}
	return mgr
}

func makeActiveApplication(id, displayName string) ocifunctions.Application {
	return ocifunctions.Application{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: ocifunctions.ApplicationLifecycleStateActive,
	}
}

func makeActiveFunction(id, displayName, invokeEndpoint string) ocifunctions.Function {
	fn := ocifunctions.Function{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: ocifunctions.FunctionLifecycleStateActive,
	}
	if invokeEndpoint != "" {
		fn.InvokeEndpoint = common.String(invokeEndpoint)
	}
	return fn
}

// --- FunctionsApplication mock-based tests ---

// TestFunctionsApplication_CreateOrUpdate_Create_Success verifies the full create path:
// no existing app by name → CreateApplication → GetApplication → ACTIVE → success.
func TestFunctionsApplication_CreateOrUpdate_Create_Success(t *testing.T) {
	appId := "ocid1.fnapp.oc1..aaa"
	ociClient := &mockFunctionsClient{
		listApplicationsFn: func(_ context.Context, _ ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
			return ocifunctions.ListApplicationsResponse{Items: []ocifunctions.ApplicationSummary{}}, nil
		},
		createApplicationFn: func(_ context.Context, _ ocifunctions.CreateApplicationRequest) (ocifunctions.CreateApplicationResponse, error) {
			return ocifunctions.CreateApplicationResponse{
				Application: ocifunctions.Application{Id: common.String(appId)},
			}, nil
		},
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			return ocifunctions.GetApplicationResponse{
				Application: makeActiveApplication(appId, "my-app"),
			}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Name = "my-app"
	app.Namespace = "default"
	app.Spec.DisplayName = "my-app"
	app.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	app.Spec.SubnetIds = []string{"ocid1.subnet.oc1..xxx"}

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(appId), app.Status.OsokStatus.Ocid)
}

// TestFunctionsApplication_CreateOrUpdate_Create_OciError verifies that a generic OCI error
// on CreateApplication propagates and returns IsSuccessful=false.
func TestFunctionsApplication_CreateOrUpdate_Create_OciError(t *testing.T) {
	ociErr := errors.New("OCI internal error")
	ociClient := &mockFunctionsClient{
		listApplicationsFn: func(_ context.Context, _ ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
			return ocifunctions.ListApplicationsResponse{Items: []ocifunctions.ApplicationSummary{}}, nil
		},
		createApplicationFn: func(_ context.Context, _ ocifunctions.CreateApplicationRequest) (ocifunctions.CreateApplicationResponse, error) {
			return ocifunctions.CreateApplicationResponse{}, ociErr
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Spec.DisplayName = "fail-app"
	app.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	app.Spec.SubnetIds = []string{"ocid1.subnet.oc1..xxx"}

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestFunctionsApplication_CreateOrUpdate_ExistingByName verifies that when an app with
// the same display name exists, it binds to it without creating a new one.
func TestFunctionsApplication_CreateOrUpdate_ExistingByName(t *testing.T) {
	appId := "ocid1.fnapp.oc1..existing"
	ociClient := &mockFunctionsClient{
		listApplicationsFn: func(_ context.Context, _ ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
			return ocifunctions.ListApplicationsResponse{
				Items: []ocifunctions.ApplicationSummary{
					{Id: common.String(appId), LifecycleState: ocifunctions.ApplicationLifecycleStateActive},
				},
			}, nil
		},
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			return ocifunctions.GetApplicationResponse{
				Application: makeActiveApplication(appId, "existing-app"),
			}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Spec.DisplayName = "existing-app"
	app.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	app.Spec.SubnetIds = []string{"ocid1.subnet.oc1..xxx"}

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(appId), app.Status.OsokStatus.Ocid)
}

// TestFunctionsApplication_CreateOrUpdate_Update_Success verifies the update path when
// FunctionsApplicationId is pre-set (bind to existing app and update it).
func TestFunctionsApplication_CreateOrUpdate_Update_Success(t *testing.T) {
	appId := "ocid1.fnapp.oc1..bound"
	ociClient := &mockFunctionsClient{
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			return ocifunctions.GetApplicationResponse{
				Application: makeActiveApplication(appId, "bound-app"),
			}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Spec.FunctionsApplicationId = ociv1beta1.OCID(appId)
	app.Spec.DisplayName = "bound-app"

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(appId), app.Status.OsokStatus.Ocid)
}

// TestFunctionsApplication_CreateOrUpdate_Update_GetError verifies that a GetApplication
// failure on the update path propagates correctly.
func TestFunctionsApplication_CreateOrUpdate_Update_GetError(t *testing.T) {
	ociClient := &mockFunctionsClient{
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			return ocifunctions.GetApplicationResponse{}, errors.New("not found")
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Spec.FunctionsApplicationId = "ocid1.fnapp.oc1..missing"

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestFunctionsApplication_CreateOrUpdate_FailedState verifies that a FAILED lifecycle
// state results in IsSuccessful=false with no error.
func TestFunctionsApplication_CreateOrUpdate_FailedState(t *testing.T) {
	appId := "ocid1.fnapp.oc1..failed"
	ociClient := &mockFunctionsClient{
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			app := makeActiveApplication(appId, "failed-app")
			app.LifecycleState = ocifunctions.ApplicationLifecycleStateFailed
			return ocifunctions.GetApplicationResponse{Application: app}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Spec.FunctionsApplicationId = ociv1beta1.OCID(appId)

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestFunctionsApplication_Delete_WithOcid verifies that Delete calls DeleteApplication
// when an OCID is present.
func TestFunctionsApplication_Delete_WithOcid(t *testing.T) {
	deleteCalled := false
	ociClient := &mockFunctionsClient{
		deleteApplicationFn: func(_ context.Context, _ ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error) {
			deleteCalled = true
			return ocifunctions.DeleteApplicationResponse{}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Status.OsokStatus.Ocid = "ocid1.fnapp.oc1..todelete"

	done, err := mgr.Delete(context.Background(), app)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// TestFunctionsApplication_Delete_OciError verifies that a DeleteApplication error
// is propagated correctly.
func TestFunctionsApplication_Delete_OciError(t *testing.T) {
	ociClient := &mockFunctionsClient{
		deleteApplicationFn: func(_ context.Context, _ ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error) {
			return ocifunctions.DeleteApplicationResponse{}, errors.New("delete failed")
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Status.OsokStatus.Ocid = "ocid1.fnapp.oc1..todelete"

	done, err := mgr.Delete(context.Background(), app)
	assert.Error(t, err)
	assert.False(t, done)
}

// TestFunctionsApplication_GetApplicationOcid_Found verifies that an ACTIVE application
// found by ListApplications returns its OCID.
func TestFunctionsApplication_GetApplicationOcid_Found(t *testing.T) {
	appId := "ocid1.fnapp.oc1..found"
	ociClient := &mockFunctionsClient{
		listApplicationsFn: func(_ context.Context, _ ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
			return ocifunctions.ListApplicationsResponse{
				Items: []ocifunctions.ApplicationSummary{
					{Id: common.String(appId), LifecycleState: ocifunctions.ApplicationLifecycleStateActive},
				},
			}, nil
		},
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			return ocifunctions.GetApplicationResponse{
				Application: makeActiveApplication(appId, "found-app"),
			}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Spec.DisplayName = "found-app"
	app.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	app.Spec.SubnetIds = []string{"ocid1.subnet.oc1..xxx"}

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(appId), app.Status.OsokStatus.Ocid)
}

// TestFunctionsApplication_GetApplicationOcid_ListError verifies that a ListApplications
// error propagates from CreateOrUpdate.
func TestFunctionsApplication_GetApplicationOcid_ListError(t *testing.T) {
	ociClient := &mockFunctionsClient{
		listApplicationsFn: func(_ context.Context, _ ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
			return ocifunctions.ListApplicationsResponse{}, errors.New("listing failed")
		},
	}

	mgr := newAppMgr(t, ociClient)

	app := &ociv1beta1.FunctionsApplication{}
	app.Spec.DisplayName = "my-app"
	app.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	app.Spec.SubnetIds = []string{"ocid1.subnet.oc1..xxx"}

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestFunctionsApplication_RetryPolicy_CREATING verifies that the retry policy
// returns true (should retry) when the application is in CREATING state.
func TestFunctionsApplication_RetryPolicy_CREATING(t *testing.T) {
	mgr := newAppMgr(t, nil)
	policy := ExportGetAppRetryPolicy(mgr, 5)

	creatingResp := common.OCIOperationResponse{
		Response: ocifunctions.GetApplicationResponse{
			Application: ocifunctions.Application{
				LifecycleState: ocifunctions.ApplicationLifecycleStateCreating,
			},
		},
	}
	assert.True(t, policy.ShouldRetryOperation(creatingResp), "should retry when CREATING")

	activeResp := common.OCIOperationResponse{
		Response: ocifunctions.GetApplicationResponse{
			Application: ocifunctions.Application{
				LifecycleState: ocifunctions.ApplicationLifecycleStateActive,
			},
		},
	}
	assert.False(t, policy.ShouldRetryOperation(activeResp), "should not retry when ACTIVE")
}

// TestFunctionsApplication_RetryPolicy_WrongResponseType verifies that the retry policy
// defaults to true when the response is not a GetApplicationResponse.
func TestFunctionsApplication_RetryPolicy_WrongResponseType(t *testing.T) {
	mgr := newAppMgr(t, nil)
	policy := ExportGetAppRetryPolicy(mgr, 5)

	wrongResp := common.OCIOperationResponse{
		Response: ocifunctions.GetFunctionResponse{},
	}
	assert.True(t, policy.ShouldRetryOperation(wrongResp), "should retry on unrecognized response type")
}

// --- FunctionsFunction mock-based tests ---

// TestFunctionsFunction_CreateOrUpdate_Create_Success verifies the full create path:
// no existing function → CreateFunction → GetFunction → ACTIVE → stores invoke secret.
func TestFunctionsFunction_CreateOrUpdate_Create_Success(t *testing.T) {
	fnId := "ocid1.fnfunc.oc1..aaa"
	invokeEndpoint := "https://xyz.functions.oci.example.com/20181201/functions/" + fnId + "/actions/invoke"
	cred := &fakeCredentialClient{}
	ociClient := &mockFunctionsClient{
		listFunctionsFn: func(_ context.Context, _ ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			return ocifunctions.ListFunctionsResponse{Items: []ocifunctions.FunctionSummary{}}, nil
		},
		createFunctionFn: func(_ context.Context, _ ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error) {
			return ocifunctions.CreateFunctionResponse{
				Function: ocifunctions.Function{Id: common.String(fnId)},
			}, nil
		},
		getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			return ocifunctions.GetFunctionResponse{
				Function: makeActiveFunction(fnId, "my-fn", invokeEndpoint),
			}, nil
		},
	}

	mgr := newFuncMgr(t, cred, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Name = "my-fn"
	fn.Namespace = "default"
	fn.Spec.DisplayName = "my-fn"
	fn.Spec.ApplicationId = "ocid1.fnapp.oc1..xxx"
	fn.Spec.Image = "phx.ocir.io/mytenancy/myrepo:latest"
	fn.Spec.MemoryInMBs = 256

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(fnId), fn.Status.OsokStatus.Ocid)
	assert.True(t, cred.createCalled, "invoke endpoint secret should be created")
}

// TestFunctionsFunction_CreateOrUpdate_Create_OciError verifies that a generic OCI error
// on CreateFunction propagates and returns IsSuccessful=false.
func TestFunctionsFunction_CreateOrUpdate_Create_OciError(t *testing.T) {
	ociClient := &mockFunctionsClient{
		listFunctionsFn: func(_ context.Context, _ ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			return ocifunctions.ListFunctionsResponse{Items: []ocifunctions.FunctionSummary{}}, nil
		},
		createFunctionFn: func(_ context.Context, _ ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error) {
			return ocifunctions.CreateFunctionResponse{}, errors.New("OCI internal error")
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Spec.DisplayName = "fail-fn"
	fn.Spec.ApplicationId = "ocid1.fnapp.oc1..xxx"
	fn.Spec.Image = "phx.ocir.io/mytenancy/myrepo:latest"
	fn.Spec.MemoryInMBs = 256

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestFunctionsFunction_CreateOrUpdate_ExistingByName verifies that when a function with
// the same display name exists, it binds to it without creating a new one.
func TestFunctionsFunction_CreateOrUpdate_ExistingByName(t *testing.T) {
	fnId := "ocid1.fnfunc.oc1..existing"
	ociClient := &mockFunctionsClient{
		listFunctionsFn: func(_ context.Context, _ ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			return ocifunctions.ListFunctionsResponse{
				Items: []ocifunctions.FunctionSummary{
					{Id: common.String(fnId), LifecycleState: ocifunctions.FunctionLifecycleStateActive},
				},
			}, nil
		},
		getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			return ocifunctions.GetFunctionResponse{
				Function: makeActiveFunction(fnId, "existing-fn", ""),
			}, nil
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Spec.DisplayName = "existing-fn"
	fn.Spec.ApplicationId = "ocid1.fnapp.oc1..xxx"
	fn.Spec.Image = "phx.ocir.io/mytenancy/myrepo:latest"
	fn.Spec.MemoryInMBs = 256

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(fnId), fn.Status.OsokStatus.Ocid)
}

// TestFunctionsFunction_CreateOrUpdate_Update_Success verifies the update path when
// FunctionsFunctionId is pre-set (bind to existing function and update it).
func TestFunctionsFunction_CreateOrUpdate_Update_Success(t *testing.T) {
	fnId := "ocid1.fnfunc.oc1..bound"
	ociClient := &mockFunctionsClient{
		getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			return ocifunctions.GetFunctionResponse{
				Function: makeActiveFunction(fnId, "bound-fn", ""),
			}, nil
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Spec.FunctionsFunctionId = ociv1beta1.OCID(fnId)
	fn.Spec.DisplayName = "bound-fn"
	fn.Spec.Image = "phx.ocir.io/mytenancy/myrepo:latest"
	fn.Spec.MemoryInMBs = 256

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(fnId), fn.Status.OsokStatus.Ocid)
}

// TestFunctionsFunction_CreateOrUpdate_Update_GetError verifies that a GetFunction
// failure on the update path propagates correctly.
func TestFunctionsFunction_CreateOrUpdate_Update_GetError(t *testing.T) {
	ociClient := &mockFunctionsClient{
		getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			return ocifunctions.GetFunctionResponse{}, errors.New("not found")
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Spec.FunctionsFunctionId = "ocid1.fnfunc.oc1..missing"

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestFunctionsFunction_CreateOrUpdate_FailedState verifies that FAILED lifecycle state
// results in IsSuccessful=false with no error.
func TestFunctionsFunction_CreateOrUpdate_FailedState(t *testing.T) {
	fnId := "ocid1.fnfunc.oc1..failed"
	ociClient := &mockFunctionsClient{
		getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			fn := makeActiveFunction(fnId, "failed-fn", "")
			fn.LifecycleState = ocifunctions.FunctionLifecycleStateFailed
			return ocifunctions.GetFunctionResponse{Function: fn}, nil
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Spec.FunctionsFunctionId = ociv1beta1.OCID(fnId)

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestFunctionsFunction_Delete_WithOcid verifies that Delete calls DeleteFunction and
// DeleteSecret when an OCID is present.
func TestFunctionsFunction_Delete_WithOcid(t *testing.T) {
	deleteFnCalled := false
	ociClient := &mockFunctionsClient{
		deleteFunctionFn: func(_ context.Context, _ ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
			deleteFnCalled = true
			return ocifunctions.DeleteFunctionResponse{}, nil
		},
	}
	cred := &fakeCredentialClient{}

	mgr := newFuncMgr(t, cred, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Name = "my-fn"
	fn.Namespace = "default"
	fn.Status.OsokStatus.Ocid = "ocid1.fnfunc.oc1..todelete"

	done, err := mgr.Delete(context.Background(), fn)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteFnCalled)
	assert.True(t, cred.deleteCalled, "DeleteSecret should be called when OCID is set")
}

// TestFunctionsFunction_Delete_OciError verifies that a DeleteFunction OCI error
// is propagated correctly.
func TestFunctionsFunction_Delete_OciError(t *testing.T) {
	ociClient := &mockFunctionsClient{
		deleteFunctionFn: func(_ context.Context, _ ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
			return ocifunctions.DeleteFunctionResponse{}, errors.New("delete failed")
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Name = "my-fn"
	fn.Namespace = "default"
	fn.Status.OsokStatus.Ocid = "ocid1.fnfunc.oc1..todelete"

	done, err := mgr.Delete(context.Background(), fn)
	assert.Error(t, err)
	assert.False(t, done)
}

// TestFunctionsFunction_InvokeEndpoint_StoredInSecret verifies the invoke endpoint is stored
// in the credential secret after a successful create.
func TestFunctionsFunction_InvokeEndpoint_StoredInSecret(t *testing.T) {
	fnId := "ocid1.fnfunc.oc1..ep"
	invokeEndpoint := "https://ep.functions.oci.example.com/20181201/functions/" + fnId + "/actions/invoke"
	var capturedData map[string][]byte
	cred := &fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, data map[string][]byte) (bool, error) {
			capturedData = data
			return true, nil
		},
	}
	ociClient := &mockFunctionsClient{
		listFunctionsFn: func(_ context.Context, _ ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			return ocifunctions.ListFunctionsResponse{Items: []ocifunctions.FunctionSummary{}}, nil
		},
		createFunctionFn: func(_ context.Context, _ ocifunctions.CreateFunctionRequest) (ocifunctions.CreateFunctionResponse, error) {
			return ocifunctions.CreateFunctionResponse{
				Function: ocifunctions.Function{Id: common.String(fnId)},
			}, nil
		},
		getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			return ocifunctions.GetFunctionResponse{
				Function: makeActiveFunction(fnId, "ep-fn", invokeEndpoint),
			}, nil
		},
	}

	mgr := newFuncMgr(t, cred, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Name = "ep-fn"
	fn.Namespace = "default"
	fn.Spec.DisplayName = "ep-fn"
	fn.Spec.ApplicationId = "ocid1.fnapp.oc1..xxx"
	fn.Spec.Image = "phx.ocir.io/mytenancy/myrepo:latest"
	fn.Spec.MemoryInMBs = 256

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, fnId, string(capturedData["functionId"]))
	assert.Equal(t, invokeEndpoint, string(capturedData["invokeEndpoint"]))
}

// TestFunctionsFunction_GetFunctionOcid_ListError verifies that a ListFunctions error
// propagates from CreateOrUpdate.
func TestFunctionsFunction_GetFunctionOcid_ListError(t *testing.T) {
	ociClient := &mockFunctionsClient{
		listFunctionsFn: func(_ context.Context, _ ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			return ocifunctions.ListFunctionsResponse{}, errors.New("listing failed")
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Spec.DisplayName = "my-fn"
	fn.Spec.ApplicationId = "ocid1.fnapp.oc1..xxx"
	fn.Spec.Image = "phx.ocir.io/mytenancy/myrepo:latest"
	fn.Spec.MemoryInMBs = 256

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestFunctionsFunction_RetryPolicy_CREATING verifies that the retry policy
// returns true (should retry) when the function is in CREATING state.
func TestFunctionsFunction_RetryPolicy_CREATING(t *testing.T) {
	mgr := newFuncMgr(t, nil, nil)
	policy := ExportGetFuncRetryPolicy(mgr, 5)

	creatingResp := common.OCIOperationResponse{
		Response: ocifunctions.GetFunctionResponse{
			Function: ocifunctions.Function{
				LifecycleState: ocifunctions.FunctionLifecycleStateCreating,
			},
		},
	}
	assert.True(t, policy.ShouldRetryOperation(creatingResp), "should retry when CREATING")

	activeResp := common.OCIOperationResponse{
		Response: ocifunctions.GetFunctionResponse{
			Function: ocifunctions.Function{
				LifecycleState: ocifunctions.FunctionLifecycleStateActive,
			},
		},
	}
	assert.False(t, policy.ShouldRetryOperation(activeResp), "should not retry when ACTIVE")
}

// TestFunctionsFunction_RetryPolicy_WrongResponseType verifies that the retry policy
// defaults to true when the response is not a GetFunctionResponse.
func TestFunctionsFunction_RetryPolicy_WrongResponseType(t *testing.T) {
	mgr := newFuncMgr(t, nil, nil)
	policy := ExportGetFuncRetryPolicy(mgr, 5)

	wrongResp := common.OCIOperationResponse{
		Response: ocifunctions.GetApplicationResponse{},
	}
	assert.True(t, policy.ShouldRetryOperation(wrongResp), "should retry on unrecognized response type")
}
