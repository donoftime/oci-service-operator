/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/quick"

	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
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
