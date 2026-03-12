/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestFunctionsApplication_PropertyBindByIDUsesSpecIDForUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		appID := fmt.Sprintf("ocid1.fnapp.oc1..%d", seed)
		var updatedID string
		ociClient := &mockFunctionsClient{
			getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
				return ocifunctions.GetApplicationResponse{
					Application: makeActiveApplication(appID, "prop-app"),
				}, nil
			},
			updateApplicationFn: func(_ context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
				updatedID = *req.ApplicationId
				return ocifunctions.UpdateApplicationResponse{}, nil
			},
		}

		mgr := newAppMgr(t, ociClient)
		app := &ociv1beta1.FunctionsApplication{}
		app.Spec.FunctionsApplicationId = ociv1beta1.OCID(appID)
		app.Spec.Config = map[string]string{"k": "v"}

		resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
		return err == nil && resp.IsSuccessful && updatedID == appID
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsFunction_PropertyBindByIDUsesSpecIDForUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		functionID := fmt.Sprintf("ocid1.fnfunc.oc1..%d", seed)
		var updatedID string
		ociClient := &mockFunctionsClient{
			getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
				return ocifunctions.GetFunctionResponse{
					Function: makeActiveFunction(functionID, "prop-fn", ""),
				}, nil
			},
			updateFunctionFn: func(_ context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
				updatedID = *req.FunctionId
				return ocifunctions.UpdateFunctionResponse{}, nil
			},
		}

		mgr := newFuncMgr(t, nil, ociClient)
		fn := &ociv1beta1.FunctionsFunction{}
		fn.Spec.FunctionsFunctionId = ociv1beta1.OCID(functionID)
		fn.Spec.Image = "phx.ocir.io/mytenancy/repo:latest"

		resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
		return err == nil && resp.IsSuccessful && updatedID == functionID
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyRetryableStatesRequeue(t *testing.T) {
	states := []ocifunctions.ApplicationLifecycleStateEnum{
		ocifunctions.ApplicationLifecycleStateCreating,
		ocifunctions.ApplicationLifecycleStateUpdating,
		ocifunctions.ApplicationLifecycleStateDeleting,
	}

	property := func(seed uint8) bool {
		state := states[int(seed)%len(states)]
		appID := fmt.Sprintf("ocid1.fnapp.oc1..retry-%d", seed)
		ociClient := &mockFunctionsClient{
			getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
				app := makeActiveApplication(appID, "retry-app")
				app.LifecycleState = state
				return ocifunctions.GetApplicationResponse{Application: app}, nil
			},
		}

		mgr := newAppMgr(t, ociClient)
		app := &ociv1beta1.FunctionsApplication{}
		app.Spec.FunctionsApplicationId = ociv1beta1.OCID(appID)

		resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
		return err == nil && !resp.IsSuccessful && resp.ShouldRequeue
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyExistingByNameUsesResolvedIDForUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		appID := fmt.Sprintf("ocid1.fnapp.oc1..existing-%d", seed)
		var updatedReq ocifunctions.UpdateApplicationRequest
		ociClient := &mockFunctionsClient{
			listApplicationsFn: func(_ context.Context, _ ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
				return ocifunctions.ListApplicationsResponse{
					Items: []ocifunctions.ApplicationSummary{
						{Id: common.String(appID), LifecycleState: ocifunctions.ApplicationLifecycleStateActive},
					},
				}, nil
			},
			getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
				app := makeActiveApplication(appID, "prop-app")
				app.Config = map[string]string{"mode": "old"}
				app.NetworkSecurityGroupIds = []string{"ocid1.nsg.oc1..old"}
				app.SyslogUrl = common.String("tcp://old.example.com")
				app.FreeformTags = map[string]string{"team": "old"}
				app.DefinedTags = map[string]map[string]interface{}{"ops": {"env": "dev"}}
				return ocifunctions.GetApplicationResponse{Application: app}, nil
			},
			updateApplicationFn: func(_ context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
				updatedReq = req
				return ocifunctions.UpdateApplicationResponse{}, nil
			},
		}

		mgr := newAppMgr(t, ociClient)
		app := &ociv1beta1.FunctionsApplication{}
		app.Spec.DisplayName = "prop-app"
		app.Spec.CompartmentId = ociv1beta1.OCID(fmt.Sprintf("ocid1.compartment.oc1..%d", seed))
		app.Spec.Config = map[string]string{"mode": "new"}
		app.Spec.NetworkSecurityGroupIds = []string{"ocid1.nsg.oc1..new"}
		app.Spec.SyslogUrl = "tcp://new.example.com"
		app.Spec.FreeFormTags = map[string]string{"team": "platform"}
		app.Spec.DefinedTags = map[string]ociv1beta1.MapValue{"ops": {"env": "prod"}}

		resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
		return err == nil &&
			resp.IsSuccessful &&
			updatedReq.ApplicationId != nil &&
			*updatedReq.ApplicationId == appID &&
			updatedReq.Config["mode"] == "new" &&
			len(updatedReq.NetworkSecurityGroupIds) == 1 &&
			updatedReq.NetworkSecurityGroupIds[0] == "ocid1.nsg.oc1..new" &&
			updatedReq.SyslogUrl != nil &&
			*updatedReq.SyslogUrl == "tcp://new.example.com" &&
			updatedReq.FreeformTags["team"] == "platform" &&
			updatedReq.DefinedTags["ops"]["env"] == "prod"
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyExistingByNameSkipsNoOpUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		appID := fmt.Sprintf("ocid1.fnapp.oc1..noop-%d", seed)
		updateCalled := false
		ociClient := &mockFunctionsClient{
			listApplicationsFn: func(_ context.Context, _ ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
				return ocifunctions.ListApplicationsResponse{
					Items: []ocifunctions.ApplicationSummary{
						{Id: common.String(appID), LifecycleState: ocifunctions.ApplicationLifecycleStateActive},
					},
				}, nil
			},
			getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
				app := makeActiveApplication(appID, "prop-app")
				app.Config = map[string]string{"mode": "steady"}
				app.NetworkSecurityGroupIds = []string{"ocid1.nsg.oc1..steady"}
				app.SyslogUrl = common.String("tcp://steady.example.com")
				app.FreeformTags = map[string]string{"team": "platform"}
				app.DefinedTags = map[string]map[string]interface{}{"ops": {"env": "prod"}}
				return ocifunctions.GetApplicationResponse{Application: app}, nil
			},
			updateApplicationFn: func(_ context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
				updateCalled = true
				return ocifunctions.UpdateApplicationResponse{}, nil
			},
		}

		mgr := newAppMgr(t, ociClient)
		app := &ociv1beta1.FunctionsApplication{}
		app.Spec.DisplayName = "prop-app"
		app.Spec.CompartmentId = ociv1beta1.OCID(fmt.Sprintf("ocid1.compartment.oc1..%d", seed))
		app.Spec.Config = map[string]string{"mode": "steady"}
		app.Spec.NetworkSecurityGroupIds = []string{"ocid1.nsg.oc1..steady"}
		app.Spec.SyslogUrl = "tcp://steady.example.com"
		app.Spec.FreeFormTags = map[string]string{"team": "platform"}
		app.Spec.DefinedTags = map[string]ociv1beta1.MapValue{"ops": {"env": "prod"}}

		resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
		return err == nil && resp.IsSuccessful && !updateCalled
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyStatusIDUsesTrackedResourceForUpdate(t *testing.T) {
	appID := "ocid1.fnapp.oc1..tracked"
	var updatedID string
	ociClient := &mockFunctionsClient{
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			app := makeActiveApplication(appID, "tracked-app")
			app.Config = map[string]string{"mode": "old"}
			return ocifunctions.GetApplicationResponse{Application: app}, nil
		},
		updateApplicationFn: func(_ context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
			updatedID = *req.ApplicationId
			return ocifunctions.UpdateApplicationResponse{}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)
	app := &ociv1beta1.FunctionsApplication{}
	app.Status.OsokStatus.Ocid = ociv1beta1.OCID(appID)
	app.Spec.Config = map[string]string{"mode": "new"}

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, appID, updatedID)
}

func TestFunctionsApplication_PropertyCompartmentDriftTriggersMove(t *testing.T) {
	appID := "ocid1.fnapp.oc1..move"
	var moved ocifunctions.ChangeApplicationCompartmentRequest
	ociClient := &mockFunctionsClient{
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			app := makeActiveApplication(appID, "move-app")
			app.CompartmentId = common.String("ocid1.compartment.oc1..old")
			return ocifunctions.GetApplicationResponse{Application: app}, nil
		},
		changeApplicationCompartmentFn: func(_ context.Context, req ocifunctions.ChangeApplicationCompartmentRequest) (ocifunctions.ChangeApplicationCompartmentResponse, error) {
			moved = req
			return ocifunctions.ChangeApplicationCompartmentResponse{}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)
	app := &ociv1beta1.FunctionsApplication{}
	app.Status.OsokStatus.Ocid = ociv1beta1.OCID(appID)
	app.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	assert.NoError(t, mgr.UpdateApplication(context.Background(), app))
	assert.Equal(t, appID, *moved.ApplicationId)
	assert.Equal(t, string(app.Spec.CompartmentId), *moved.CompartmentId)
}

func TestFunctionsFunction_PropertyRetryableStatesRequeue(t *testing.T) {
	states := []ocifunctions.FunctionLifecycleStateEnum{
		ocifunctions.FunctionLifecycleStateCreating,
		ocifunctions.FunctionLifecycleStateUpdating,
		ocifunctions.FunctionLifecycleStateDeleting,
	}

	property := func(seed uint8) bool {
		state := states[int(seed)%len(states)]
		functionID := fmt.Sprintf("ocid1.fnfunc.oc1..retry-%d", seed)
		credClient := &fakeCredentialClient{}
		ociClient := &mockFunctionsClient{
			getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
				fn := makeActiveFunction(functionID, "retry-fn", "https://invoke.example.com")
				fn.LifecycleState = state
				return ocifunctions.GetFunctionResponse{Function: fn}, nil
			},
		}

		mgr := newFuncMgr(t, credClient, ociClient)
		fn := &ociv1beta1.FunctionsFunction{}
		fn.Spec.FunctionsFunctionId = ociv1beta1.OCID(functionID)

		resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
		return err == nil && !resp.IsSuccessful && resp.ShouldRequeue && !credClient.createCalled
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsFunction_PropertyDeleteSecretErrorsBlockCompletion(t *testing.T) {
	property := func(seed uint16) bool {
		functionID := fmt.Sprintf("ocid1.fnfunc.oc1..delete-%d", seed)
		ociClient := &mockFunctionsClient{
			deleteFunctionFn: func(_ context.Context, _ ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
				return ocifunctions.DeleteFunctionResponse{}, nil
			},
			getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
				fn := makeActiveFunction(functionID, "delete-fn", "")
				fn.LifecycleState = ocifunctions.FunctionLifecycleStateDeleted
				return ocifunctions.GetFunctionResponse{Function: fn}, nil
			},
		}
		credClient := &fakeCredentialClient{
			deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, errors.New("secret delete failed")
			},
		}

		mgr := newFuncMgr(t, credClient, ociClient)
		fn := &ociv1beta1.FunctionsFunction{}
		fn.Name = "delete-fn"
		fn.Namespace = "default"
		fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(functionID)

		done, err := mgr.Delete(context.Background(), fn)
		return err != nil && !done
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsFunction_PropertyDeleteIgnoresMissingSecret(t *testing.T) {
	property := func(seed uint16) bool {
		functionID := fmt.Sprintf("ocid1.fnfunc.oc1..missing-%d", seed)
		ociClient := &mockFunctionsClient{
			deleteFunctionFn: func(_ context.Context, _ ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
				return ocifunctions.DeleteFunctionResponse{}, nil
			},
			getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
				fn := makeActiveFunction(functionID, "missing-fn", "")
				fn.LifecycleState = ocifunctions.FunctionLifecycleStateDeleted
				return ocifunctions.GetFunctionResponse{Function: fn}, nil
			},
		}
		credClient := &fakeCredentialClient{
			deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, apierrors.NewNotFound(corev1.Resource("secret"), "missing-fn")
			},
		}

		mgr := newFuncMgr(t, credClient, ociClient)
		fn := &ociv1beta1.FunctionsFunction{}
		fn.Name = "missing-fn"
		fn.Namespace = "default"
		fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(functionID)

		done, err := mgr.Delete(context.Background(), fn)
		return err == nil && done
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyDeleteFallsBackToSpecID(t *testing.T) {
	property := func(seed uint16) bool {
		appID := fmt.Sprintf("ocid1.fnapp.oc1..delete-%d", seed)
		var deletedID string
		ociClient := &mockFunctionsClient{
			deleteApplicationFn: func(_ context.Context, req ocifunctions.DeleteApplicationRequest) (ocifunctions.DeleteApplicationResponse, error) {
				deletedID = *req.ApplicationId
				return ocifunctions.DeleteApplicationResponse{}, nil
			},
			getApplicationFn: func(_ context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
				app := makeActiveApplication(*req.ApplicationId, "delete-app")
				app.LifecycleState = ocifunctions.ApplicationLifecycleStateDeleted
				return ocifunctions.GetApplicationResponse{Application: app}, nil
			},
		}

		mgr := newAppMgr(t, ociClient)
		app := &ociv1beta1.FunctionsApplication{}
		app.Spec.FunctionsApplicationId = ociv1beta1.OCID(appID)

		done, err := mgr.Delete(context.Background(), app)
		return err == nil && done && deletedID == appID
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsFunction_PropertyDeleteFallsBackToSpecID(t *testing.T) {
	property := func(seed uint16) bool {
		functionID := fmt.Sprintf("ocid1.fnfunc.oc1..delete-bind-%d", seed)
		var deletedID string
		ociClient := &mockFunctionsClient{
			deleteFunctionFn: func(_ context.Context, req ocifunctions.DeleteFunctionRequest) (ocifunctions.DeleteFunctionResponse, error) {
				deletedID = *req.FunctionId
				return ocifunctions.DeleteFunctionResponse{}, nil
			},
			getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
				fn := makeActiveFunction(*req.FunctionId, "delete-fn", "")
				fn.LifecycleState = ocifunctions.FunctionLifecycleStateDeleted
				return ocifunctions.GetFunctionResponse{Function: fn}, nil
			},
		}
		credClient := &fakeCredentialClient{
			deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
				return false, apierrors.NewNotFound(corev1.Resource("secret"), "delete-fn")
			},
		}

		mgr := newFuncMgr(t, credClient, ociClient)
		fn := &ociv1beta1.FunctionsFunction{}
		fn.Name = "delete-fn"
		fn.Namespace = "default"
		fn.Spec.FunctionsFunctionId = ociv1beta1.OCID(functionID)

		done, err := mgr.Delete(context.Background(), fn)
		return err == nil && done && deletedID == functionID
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyConfigAndTagDriftTriggersUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		appID := fmt.Sprintf("ocid1.fnapp.oc1..drift-%d", seed)
		expectedDefinedTags := map[string]map[string]interface{}{
			"ops": {"env": "prod"},
		}
		var captured ocifunctions.UpdateApplicationRequest
		ociClient := &mockFunctionsClient{
			getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
				app := makeActiveApplication(appID, "drift-app")
				app.Config = map[string]string{"OLD": "value"}
				app.NetworkSecurityGroupIds = []string{"ocid1.nsg.oc1..old"}
				app.SyslogUrl = common.String("tcp://old.example.com:514")
				app.FreeformTags = map[string]string{"team": "old"}
				app.DefinedTags = map[string]map[string]interface{}{
					"ops": {"env": "dev"},
				}
				return ocifunctions.GetApplicationResponse{Application: app}, nil
			},
			updateApplicationFn: func(_ context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
				captured = req
				return ocifunctions.UpdateApplicationResponse{}, nil
			},
		}

		mgr := newAppMgr(t, ociClient)
		app := &ociv1beta1.FunctionsApplication{}
		app.Spec.FunctionsApplicationId = ociv1beta1.OCID(appID)
		app.Spec.Config = map[string]string{"NEW": "value"}
		app.Spec.NetworkSecurityGroupIds = []string{"ocid1.nsg.oc1..new"}
		app.Spec.SyslogUrl = "tcp://new.example.com:514"
		app.Spec.FreeFormTags = map[string]string{"team": "platform"}
		app.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
			"ops": {"env": "prod"},
		}

		resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
		return err == nil &&
			resp.IsSuccessful &&
			reflect.DeepEqual(captured.Config, app.Spec.Config) &&
			reflect.DeepEqual(captured.NetworkSecurityGroupIds, app.Spec.NetworkSecurityGroupIds) &&
			captured.SyslogUrl != nil &&
			*captured.SyslogUrl == app.Spec.SyslogUrl &&
			reflect.DeepEqual(captured.FreeformTags, app.Spec.FreeFormTags) &&
			reflect.DeepEqual(captured.DefinedTags, expectedDefinedTags)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyMatchingConfigSkipsUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		appID := fmt.Sprintf("ocid1.fnapp.oc1..same-%d", seed)
		updateCalled := false
		ociClient := &mockFunctionsClient{
			getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
				app := makeActiveApplication(appID, "same-app")
				app.Config = map[string]string{"APP_MODE": "prod"}
				app.NetworkSecurityGroupIds = []string{"ocid1.nsg.oc1..same"}
				app.SyslogUrl = common.String("tcp://same.example.com:514")
				app.FreeformTags = map[string]string{"team": "platform"}
				app.DefinedTags = map[string]map[string]interface{}{
					"ops": {"env": "prod"},
				}
				return ocifunctions.GetApplicationResponse{Application: app}, nil
			},
			updateApplicationFn: func(_ context.Context, _ ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
				updateCalled = true
				return ocifunctions.UpdateApplicationResponse{}, nil
			},
		}

		mgr := newAppMgr(t, ociClient)
		app := &ociv1beta1.FunctionsApplication{}
		app.Spec.FunctionsApplicationId = ociv1beta1.OCID(appID)
		app.Spec.Config = map[string]string{"APP_MODE": "prod"}
		app.Spec.NetworkSecurityGroupIds = []string{"ocid1.nsg.oc1..same"}
		app.Spec.SyslogUrl = "tcp://same.example.com:514"
		app.Spec.FreeFormTags = map[string]string{"team": "platform"}
		app.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
			"ops": {"env": "prod"},
		}

		resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
		return err == nil && resp.IsSuccessful && !updateCalled
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyResolvedExistingApplicationAppliesUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		appID := fmt.Sprintf("ocid1.fnapp.oc1..resolved-%d", seed)
		var updatedID string
		var updatedConfig map[string]string
		ociClient := &mockFunctionsClient{
			listApplicationsFn: func(_ context.Context, _ ocifunctions.ListApplicationsRequest) (ocifunctions.ListApplicationsResponse, error) {
				return ocifunctions.ListApplicationsResponse{
					Items: []ocifunctions.ApplicationSummary{{
						Id:             common.String(appID),
						DisplayName:    common.String("resolved-app"),
						LifecycleState: ocifunctions.ApplicationLifecycleStateActive,
					}},
				}, nil
			},
			getApplicationFn: func(_ context.Context, req ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
				app := makeActiveApplication(*req.ApplicationId, "resolved-app")
				app.Config = map[string]string{"OLD": "value"}
				return ocifunctions.GetApplicationResponse{Application: app}, nil
			},
			updateApplicationFn: func(_ context.Context, req ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
				updatedID = *req.ApplicationId
				updatedConfig = req.Config
				return ocifunctions.UpdateApplicationResponse{}, nil
			},
		}

		mgr := newAppMgr(t, ociClient)
		app := &ociv1beta1.FunctionsApplication{}
		app.Spec.DisplayName = "resolved-app"
		app.Spec.CompartmentId = "ocid1.compartment.oc1..x"
		app.Spec.Config = map[string]string{"NEW": "value"}

		resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
		return err == nil && resp.IsSuccessful && updatedID == appID && reflect.DeepEqual(updatedConfig, app.Spec.Config)
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsApplication_PropertyImmutableSubnetChangeFailsBeforeMutation(t *testing.T) {
	updateCalled := false
	ociClient := &mockFunctionsClient{
		getApplicationFn: func(_ context.Context, _ ocifunctions.GetApplicationRequest) (ocifunctions.GetApplicationResponse, error) {
			app := makeActiveApplication("ocid1.fnapp.oc1..immutable", "immutable-app")
			app.SubnetIds = []string{"ocid1.subnet.oc1..old"}
			return ocifunctions.GetApplicationResponse{Application: app}, nil
		},
		updateApplicationFn: func(_ context.Context, _ ocifunctions.UpdateApplicationRequest) (ocifunctions.UpdateApplicationResponse, error) {
			updateCalled = true
			return ocifunctions.UpdateApplicationResponse{}, nil
		},
	}

	mgr := newAppMgr(t, ociClient)
	app := &ociv1beta1.FunctionsApplication{}
	app.Spec.FunctionsApplicationId = "ocid1.fnapp.oc1..immutable"
	app.Spec.SubnetIds = []string{"ocid1.subnet.oc1..new"}

	resp, err := mgr.CreateOrUpdate(context.Background(), app, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "subnetIds cannot be updated in place")
	assert.False(t, updateCalled)
}

func TestFunctionsFunction_PropertyTagDriftTriggersUpdate(t *testing.T) {
	functionID := "ocid1.fnfunc.oc1..tags"
	var captured ocifunctions.UpdateFunctionRequest
	ociClient := &mockFunctionsClient{
		getFunctionFn: func(_ context.Context, _ ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			fn := makeActiveFunction(functionID, "tagged-fn", "")
			fn.ApplicationId = common.String("ocid1.fnapp.oc1..same")
			fn.FreeformTags = map[string]string{"team": "old"}
			fn.DefinedTags = map[string]map[string]interface{}{"ops": {"env": "dev"}}
			return ocifunctions.GetFunctionResponse{Function: fn}, nil
		},
		updateFunctionFn: func(_ context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			captured = req
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)
	fn := &ociv1beta1.FunctionsFunction{}
	fn.Spec.FunctionsFunctionId = ociv1beta1.OCID(functionID)
	fn.Spec.ApplicationId = "ocid1.fnapp.oc1..same"
	fn.Spec.FreeFormTags = map[string]string{"team": "platform"}
	fn.Spec.DefinedTags = map[string]ociv1beta1.MapValue{"ops": {"env": "prod"}}

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, functionID, *captured.FunctionId)
	assert.Equal(t, map[string]string{"team": "platform"}, captured.FreeformTags)
	assert.Equal(t, map[string]map[string]interface{}{"ops": {"env": "prod"}}, captured.DefinedTags)
}

func TestFunctionsFunction_PropertyResolvedExistingFunctionAppliesUpdate(t *testing.T) {
	functionID := "ocid1.fnfunc.oc1..resolved"
	var updatedID string
	ociClient := &mockFunctionsClient{
		listFunctionsFn: func(_ context.Context, _ ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
			return ocifunctions.ListFunctionsResponse{
				Items: []ocifunctions.FunctionSummary{{
					Id:             common.String(functionID),
					DisplayName:    common.String("resolved-fn"),
					LifecycleState: ocifunctions.FunctionLifecycleStateActive,
				}},
			}, nil
		},
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			fn := makeActiveFunction(*req.FunctionId, "resolved-fn", "")
			fn.ApplicationId = common.String("ocid1.fnapp.oc1..same")
			fn.FreeformTags = map[string]string{"team": "old"}
			return ocifunctions.GetFunctionResponse{Function: fn}, nil
		},
		updateFunctionFn: func(_ context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			updatedID = *req.FunctionId
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)
	fn := &ociv1beta1.FunctionsFunction{}
	fn.Spec.DisplayName = "resolved-fn"
	fn.Spec.ApplicationId = "ocid1.fnapp.oc1..same"
	fn.Spec.FreeFormTags = map[string]string{"team": "platform"}

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, functionID, updatedID)
}

func TestFunctionsFunction_PropertyStatusIDUsesTrackedResourceForUpdate(t *testing.T) {
	functionID := "ocid1.fnfunc.oc1..tracked"
	var updatedID string
	ociClient := &mockFunctionsClient{
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			fn := makeActiveFunction(*req.FunctionId, "tracked-fn", "https://invoke.example.com")
			fn.Image = common.String("phx.ocir.io/mytenancy/repo:old")
			return ocifunctions.GetFunctionResponse{Function: fn}, nil
		},
		updateFunctionFn: func(_ context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			updatedID = *req.FunctionId
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)
	fn := &ociv1beta1.FunctionsFunction{}
	fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(functionID)
	fn.Spec.Image = "phx.ocir.io/mytenancy/repo:new"

	resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, functionID, updatedID)
}

func TestFunctionsFunction_PropertyResolvedExistingFunctionAppliesTagUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		functionID := fmt.Sprintf("ocid1.fnfunc.oc1..resolved-%d", seed)
		var updatedReq ocifunctions.UpdateFunctionRequest
		ociClient := &mockFunctionsClient{
			listFunctionsFn: func(_ context.Context, _ ocifunctions.ListFunctionsRequest) (ocifunctions.ListFunctionsResponse, error) {
				return ocifunctions.ListFunctionsResponse{
					Items: []ocifunctions.FunctionSummary{{
						Id:             common.String(functionID),
						DisplayName:    common.String("resolved-fn"),
						LifecycleState: ocifunctions.FunctionLifecycleStateActive,
					}},
				}, nil
			},
			getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
				fn := makeActiveFunction(*req.FunctionId, "resolved-fn", "https://invoke.example.com")
				fn.ApplicationId = common.String("ocid1.fnapp.oc1..same")
				fn.FreeformTags = map[string]string{"team": "old"}
				fn.DefinedTags = map[string]map[string]interface{}{"ops": {"env": "dev"}}
				return ocifunctions.GetFunctionResponse{Function: fn}, nil
			},
			updateFunctionFn: func(_ context.Context, req ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
				updatedReq = req
				return ocifunctions.UpdateFunctionResponse{}, nil
			},
		}

		mgr := newFuncMgr(t, nil, ociClient)
		fn := &ociv1beta1.FunctionsFunction{}
		fn.Spec.ApplicationId = "ocid1.fnapp.oc1..same"
		fn.Spec.DisplayName = "resolved-fn"
		fn.Spec.FreeFormTags = map[string]string{"team": "platform"}
		fn.Spec.DefinedTags = map[string]ociv1beta1.MapValue{"ops": {"env": "prod"}}

		resp, err := mgr.CreateOrUpdate(context.Background(), fn, ctrl.Request{})
		return err == nil &&
			resp.IsSuccessful &&
			updatedReq.FunctionId != nil &&
			*updatedReq.FunctionId == functionID &&
			updatedReq.FreeformTags["team"] == "platform" &&
			updatedReq.DefinedTags["ops"]["env"] == "prod"
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFunctionsFunction_PropertyImmutableDriftFailsBeforeMutation(t *testing.T) {
	functionID := "ocid1.fnfunc.oc1..immutable"
	updateCalled := false
	ociClient := &mockFunctionsClient{
		getFunctionFn: func(_ context.Context, req ocifunctions.GetFunctionRequest) (ocifunctions.GetFunctionResponse, error) {
			fn := makeActiveFunction(*req.FunctionId, "existing-fn", "https://invoke.example.com")
			fn.ApplicationId = common.String("ocid1.fnapp.oc1..old")
			return ocifunctions.GetFunctionResponse{Function: fn}, nil
		},
		updateFunctionFn: func(_ context.Context, _ ocifunctions.UpdateFunctionRequest) (ocifunctions.UpdateFunctionResponse, error) {
			updateCalled = true
			return ocifunctions.UpdateFunctionResponse{}, nil
		},
	}

	mgr := newFuncMgr(t, nil, ociClient)
	fn := &ociv1beta1.FunctionsFunction{}
	fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(functionID)
	fn.Spec.ApplicationId = "ocid1.fnapp.oc1..new"

	err := mgr.UpdateFunction(context.Background(), fn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "applicationId cannot be updated in place")
	assert.False(t, updateCalled)
}
