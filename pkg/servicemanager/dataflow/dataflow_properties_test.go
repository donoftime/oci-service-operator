/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dataflow_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocidataflow "github.com/oracle/oci-go-sdk/v65/dataflow"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func makeExistingApplication(id, displayName string, state ocidataflow.ApplicationLifecycleStateEnum) ocidataflow.Application {
	return ocidataflow.Application{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		Language:       ocidataflow.ApplicationLanguagePython,
		DriverShape:    common.String("VM.Standard2.1"),
		ExecutorShape:  common.String("VM.Standard2.1"),
		NumExecutors:   common.Int(1),
		SparkVersion:   common.String("3.2.1"),
		FileUri:        common.String("oci://bucket@ns/app.py"),
		LifecycleState: state,
	}
}

func TestPropertyDataFlowSkipsUpdateWhenSpecMatchesExistingState(t *testing.T) {
	var updateCalled bool
	fake := &fakeDataFlowClient{
		getApplicationFn: func(_ context.Context, _ ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
			return ocidataflow.GetApplicationResponse{Application: makeExistingApplication("ocid1.dataflowapplication.oc1..x", "test-app", ocidataflow.ApplicationLifecycleStateActive)}, nil
		},
		updateApplicationFn: func(_ context.Context, _ ocidataflow.UpdateApplicationRequest) (ocidataflow.UpdateApplicationResponse, error) {
			updateCalled = true
			return ocidataflow.UpdateApplicationResponse{}, nil
		},
	}
	mgr := mgrWithFake(fake)
	app := makeApp("test-app", "ocid1.dataflowapplication.oc1..x")

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, updateCalled)
}

func TestPropertyDataFlowExplicitDeletedApplicationFails(t *testing.T) {
	fake := &fakeDataFlowClient{
		getApplicationFn: func(_ context.Context, _ ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
			return ocidataflow.GetApplicationResponse{Application: makeExistingApplication("ocid1.dataflowapplication.oc1..deleted", "deleted-app", ocidataflow.ApplicationLifecycleStateDeleted)}, nil
		},
	}
	mgr := mgrWithFake(fake)
	app := makeApp("deleted-app", "")
	app.Spec.DataFlowApplicationId = "ocid1.dataflowapplication.oc1..deleted"

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
}

func TestPropertyDataFlowInactiveStateIsSuccessful(t *testing.T) {
	fake := &fakeDataFlowClient{
		getApplicationFn: func(_ context.Context, _ ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
			return ocidataflow.GetApplicationResponse{
				Application: makeExistingApplication("ocid1.dataflowapplication.oc1..inactive", "inactive-app", ocidataflow.ApplicationLifecycleStateInactive),
			}, nil
		},
	}
	mgr := mgrWithFake(fake)
	app := makeApp("inactive-app", "ocid1.dataflowapplication.oc1..inactive")

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

func TestPropertyDataFlowDeleteWaitsForConfirmedDisappearance(t *testing.T) {
	fake := &fakeDataFlowClient{
		deleteApplicationFn: func(_ context.Context, _ ocidataflow.DeleteApplicationRequest) (ocidataflow.DeleteApplicationResponse, error) {
			return ocidataflow.DeleteApplicationResponse{}, nil
		},
		getApplicationFn: func(_ context.Context, req ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
			return ocidataflow.GetApplicationResponse{Application: makeExistingApplication(*req.ApplicationId, "still-there", ocidataflow.ApplicationLifecycleStateActive)}, nil
		},
	}
	mgr := mgrWithFake(fake)
	app := makeApp("still-there", "ocid1.dataflowapplication.oc1..delete")

	done, err := mgr.Delete(context.Background(), app)
	assert.NoError(t, err)
	assert.False(t, done)
}

func TestPropertyDataFlowDeleteFallsBackToSpecID(t *testing.T) {
	appID := "ocid1.dataflowapplication.oc1..bind-delete"
	var deletedID string
	fake := &fakeDataFlowClient{
		deleteApplicationFn: func(_ context.Context, req ocidataflow.DeleteApplicationRequest) (ocidataflow.DeleteApplicationResponse, error) {
			deletedID = *req.ApplicationId
			return ocidataflow.DeleteApplicationResponse{}, nil
		},
		getApplicationFn: func(_ context.Context, req ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error) {
			return ocidataflow.GetApplicationResponse{
				Application: makeExistingApplication(*req.ApplicationId, "bind-delete", ocidataflow.ApplicationLifecycleStateDeleted),
			}, nil
		},
	}
	mgr := mgrWithFake(fake)
	app := makeApp("bind-delete", "")
	app.Spec.DataFlowApplicationId = ociv1beta1.OCID(appID)

	done, err := mgr.Delete(context.Background(), app)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, appID, deletedID)
}
