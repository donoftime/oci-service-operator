/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dataflow_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocidataflow "github.com/oracle/oci-go-sdk/v65/dataflow"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/dataflow"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ---------------------------------------------------------------------------
// fakeCredentialClient — implements credhelper.CredentialClient for testing.
// ---------------------------------------------------------------------------

type fakeCredentialClient struct{}

func (f *fakeCredentialClient) CreateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(_ context.Context, _, _ string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}

// ---------------------------------------------------------------------------
// fakeDataFlowClient — implements DataFlowClientInterface for testing.
// ---------------------------------------------------------------------------

type fakeDataFlowClient struct {
	createApplicationFn func(ctx context.Context, req ocidataflow.CreateApplicationRequest) (ocidataflow.CreateApplicationResponse, error)
	getApplicationFn    func(ctx context.Context, req ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error)
	listApplicationsFn  func(ctx context.Context, req ocidataflow.ListApplicationsRequest) (ocidataflow.ListApplicationsResponse, error)
	updateApplicationFn func(ctx context.Context, req ocidataflow.UpdateApplicationRequest) (ocidataflow.UpdateApplicationResponse, error)
	deleteApplicationFn func(ctx context.Context, req ocidataflow.DeleteApplicationRequest) (ocidataflow.DeleteApplicationResponse, error)
}

func (f *fakeDataFlowClient) CreateApplication(ctx context.Context, req ocidataflow.CreateApplicationRequest) (ocidataflow.CreateApplicationResponse, error) {
	if f.createApplicationFn != nil {
		return f.createApplicationFn(ctx, req)
	}
	return ocidataflow.CreateApplicationResponse{
		Application: ocidataflow.Application{
			Id:             common.String("ocid1.dataflowapplication.oc1..new"),
			DisplayName:    common.String("test-app"),
			LifecycleState: ocidataflow.ApplicationLifecycleStateActive,
		},
	}, nil
}

func (f *fakeDataFlowClient) GetApplication(ctx context.Context, req ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
	if f.getApplicationFn != nil {
		return f.getApplicationFn(ctx, req)
	}
	return ocidataflow.GetApplicationResponse{}, nil
}

func (f *fakeDataFlowClient) ListApplications(ctx context.Context, req ocidataflow.ListApplicationsRequest) (ocidataflow.ListApplicationsResponse, error) {
	if f.listApplicationsFn != nil {
		return f.listApplicationsFn(ctx, req)
	}
	return ocidataflow.ListApplicationsResponse{}, nil
}

func (f *fakeDataFlowClient) UpdateApplication(ctx context.Context, req ocidataflow.UpdateApplicationRequest) (ocidataflow.UpdateApplicationResponse, error) {
	if f.updateApplicationFn != nil {
		return f.updateApplicationFn(ctx, req)
	}
	return ocidataflow.UpdateApplicationResponse{}, nil
}

func (f *fakeDataFlowClient) DeleteApplication(ctx context.Context, req ocidataflow.DeleteApplicationRequest) (ocidataflow.DeleteApplicationResponse, error) {
	if f.deleteApplicationFn != nil {
		return f.deleteApplicationFn(ctx, req)
	}
	return ocidataflow.DeleteApplicationResponse{}, nil
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

func mgrWithFake(fake *fakeDataFlowClient) *DataFlowApplicationServiceManager {
	mgr := NewDataFlowApplicationServiceManager(emptyProvider(), &fakeCredentialClient{}, nil, defaultLog())
	ExportSetClientForTest(mgr, fake)
	return mgr
}

func makeApp(name, ocid string) *ociv1beta1.DataFlowApplication {
	app := &ociv1beta1.DataFlowApplication{}
	app.Name = name
	app.Namespace = "default"
	app.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	app.Spec.DisplayName = name
	app.Spec.Language = "PYTHON"
	app.Spec.DriverShape = "VM.Standard2.1"
	app.Spec.ExecutorShape = "VM.Standard2.1"
	app.Spec.NumExecutors = 1
	app.Spec.SparkVersion = "3.2.1"
	app.Spec.FileUri = "oci://bucket@ns/app.py"
	if ocid != "" {
		app.Status.OsokStatus.Ocid = ociv1beta1.OCID(ocid)
	}
	return app
}

// ---------------------------------------------------------------------------
// TestGetCrdStatus
// ---------------------------------------------------------------------------

func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := mgrWithFake(&fakeDataFlowClient{})

	app := makeApp("test-app", "ocid1.dataflowapplication.oc1..xxx")
	status, err := mgr.GetCrdStatus(app)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.dataflowapplication.oc1..xxx"), status.Ocid)
}

func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := mgrWithFake(&fakeDataFlowClient{})

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// ---------------------------------------------------------------------------
// TestDelete
// ---------------------------------------------------------------------------

func TestDelete_NoOcid(t *testing.T) {
	mgr := mgrWithFake(&fakeDataFlowClient{})

	app := makeApp("test-app", "")
	done, err := mgr.Delete(context.Background(), app)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestDelete_Success(t *testing.T) {
	var deleteCalled bool
	fake := &fakeDataFlowClient{
		deleteApplicationFn: func(_ context.Context, req ocidataflow.DeleteApplicationRequest) (ocidataflow.DeleteApplicationResponse, error) {
			deleteCalled = true
			assert.Equal(t, "ocid1.dataflowapplication.oc1..xxx", *req.ApplicationId)
			return ocidataflow.DeleteApplicationResponse{}, nil
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "ocid1.dataflowapplication.oc1..xxx")
	done, err := mgr.Delete(context.Background(), app)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

func TestDelete_NotFound(t *testing.T) {
	fake := &fakeDataFlowClient{
		deleteApplicationFn: func(_ context.Context, _ ocidataflow.DeleteApplicationRequest) (ocidataflow.DeleteApplicationResponse, error) {
			return ocidataflow.DeleteApplicationResponse{}, errors.New("404 NotFound")
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "ocid1.dataflowapplication.oc1..xxx")
	done, err := mgr.Delete(context.Background(), app)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestDelete_Error(t *testing.T) {
	fake := &fakeDataFlowClient{
		deleteApplicationFn: func(_ context.Context, _ ocidataflow.DeleteApplicationRequest) (ocidataflow.DeleteApplicationResponse, error) {
			return ocidataflow.DeleteApplicationResponse{}, errors.New("internal server error")
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "ocid1.dataflowapplication.oc1..xxx")
	done, err := mgr.Delete(context.Background(), app)
	assert.Error(t, err)
	assert.False(t, done)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — type assertion
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := mgrWithFake(&fakeDataFlowClient{})

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — create new
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_CreateNew(t *testing.T) {
	appID := "ocid1.dataflowapplication.oc1..new"
	var createCalled bool

	fake := &fakeDataFlowClient{
		listApplicationsFn: func(_ context.Context, _ ocidataflow.ListApplicationsRequest) (ocidataflow.ListApplicationsResponse, error) {
			return ocidataflow.ListApplicationsResponse{
				Items: []ocidataflow.ApplicationSummary{},
			}, nil
		},
		createApplicationFn: func(_ context.Context, req ocidataflow.CreateApplicationRequest) (ocidataflow.CreateApplicationResponse, error) {
			createCalled = true
			assert.Equal(t, "test-app", *req.CreateApplicationDetails.DisplayName)
			assert.Equal(t, ocidataflow.ApplicationLanguagePython, req.CreateApplicationDetails.Language)
			return ocidataflow.CreateApplicationResponse{
				Application: ocidataflow.Application{
					Id:             common.String(appID),
					DisplayName:    common.String("test-app"),
					LifecycleState: ocidataflow.ApplicationLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "")
	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, createCalled)
	assert.Equal(t, ociv1beta1.OCID(appID), app.Status.OsokStatus.Ocid)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — bind to existing via spec ID
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_Bind(t *testing.T) {
	appID := "ocid1.dataflowapplication.oc1..existing"

	fake := &fakeDataFlowClient{
		getApplicationFn: func(_ context.Context, req ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
			assert.Equal(t, appID, *req.ApplicationId)
			return ocidataflow.GetApplicationResponse{
				Application: ocidataflow.Application{
					Id:             common.String(appID),
					DisplayName:    common.String("existing-app"),
					LifecycleState: ocidataflow.ApplicationLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "")
	app.Spec.DataFlowApplicationId = ociv1beta1.OCID(appID)
	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(appID), app.Status.OsokStatus.Ocid)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — update existing via status OCID
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_Existing(t *testing.T) {
	appID := "ocid1.dataflowapplication.oc1..existing"
	var updateCalled bool

	fake := &fakeDataFlowClient{
		getApplicationFn: func(_ context.Context, _ ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
			return ocidataflow.GetApplicationResponse{
				Application: ocidataflow.Application{
					Id:             common.String(appID),
					DisplayName:    common.String("old-name"),
					LifecycleState: ocidataflow.ApplicationLifecycleStateActive,
				},
			}, nil
		},
		updateApplicationFn: func(_ context.Context, _ ocidataflow.UpdateApplicationRequest) (ocidataflow.UpdateApplicationResponse, error) {
			updateCalled = true
			return ocidataflow.UpdateApplicationResponse{}, nil
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("new-name", appID)
	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — list existing by name
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_ListFindsExisting(t *testing.T) {
	appID := "ocid1.dataflowapplication.oc1..found"

	fake := &fakeDataFlowClient{
		listApplicationsFn: func(_ context.Context, _ ocidataflow.ListApplicationsRequest) (ocidataflow.ListApplicationsResponse, error) {
			return ocidataflow.ListApplicationsResponse{
				Items: []ocidataflow.ApplicationSummary{
					{
						Id:             common.String(appID),
						DisplayName:    common.String("test-app"),
						LifecycleState: ocidataflow.ApplicationLifecycleStateActive,
					},
				},
			}, nil
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "")
	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(appID), app.Status.OsokStatus.Ocid)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — error paths
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_ListError(t *testing.T) {
	fake := &fakeDataFlowClient{
		listApplicationsFn: func(_ context.Context, _ ocidataflow.ListApplicationsRequest) (ocidataflow.ListApplicationsResponse, error) {
			return ocidataflow.ListApplicationsResponse{}, errors.New("network error")
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "")
	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_CreateError(t *testing.T) {
	fake := &fakeDataFlowClient{
		listApplicationsFn: func(_ context.Context, _ ocidataflow.ListApplicationsRequest) (ocidataflow.ListApplicationsResponse, error) {
			return ocidataflow.ListApplicationsResponse{Items: []ocidataflow.ApplicationSummary{}}, nil
		},
		createApplicationFn: func(_ context.Context, _ ocidataflow.CreateApplicationRequest) (ocidataflow.CreateApplicationResponse, error) {
			return ocidataflow.CreateApplicationResponse{}, errors.New("create failed")
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "")
	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestCreateOrUpdate_InvalidLanguage(t *testing.T) {
	fake := &fakeDataFlowClient{
		listApplicationsFn: func(_ context.Context, _ ocidataflow.ListApplicationsRequest) (ocidataflow.ListApplicationsResponse, error) {
			return ocidataflow.ListApplicationsResponse{Items: []ocidataflow.ApplicationSummary{}}, nil
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", "")
	app.Spec.Language = "INVALID"
	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — deleted state
// ---------------------------------------------------------------------------

func TestCreateOrUpdate_DeletedState(t *testing.T) {
	appID := "ocid1.dataflowapplication.oc1..deleted"

	fake := &fakeDataFlowClient{
		getApplicationFn: func(_ context.Context, _ ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
			return ocidataflow.GetApplicationResponse{
				Application: ocidataflow.Application{
					Id:             common.String(appID),
					DisplayName:    common.String("deleted-app"),
					LifecycleState: ocidataflow.ApplicationLifecycleStateDeleted,
				},
			}, nil
		},
	}
	mgr := mgrWithFake(fake)

	app := makeApp("test-app", appID)
	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
}
