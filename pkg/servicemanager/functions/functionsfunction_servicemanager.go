/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that FunctionsFunctionServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &FunctionsFunctionServiceManager{}

// FunctionsFunctionServiceManager implements OSOKServiceManager for OCI Functions Functions.
type FunctionsFunctionServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        FunctionsManagementClientInterface
}

// NewFunctionsFunctionServiceManager creates a new FunctionsFunctionServiceManager.
func NewFunctionsFunctionServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *FunctionsFunctionServiceManager {
	return &FunctionsFunctionServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the FunctionsFunction resource against OCI.
func (m *FunctionsFunctionServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	fn, err := m.convert(obj)
	if err != nil {
		m.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	fnInstance, err := m.resolveFunctionInstance(ctx, fn)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if fnInstance.Id != nil {
		fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(*fnInstance.Id)
	}
	servicemanager.SetCreatedAtIfUnset(&fn.Status.OsokStatus)

	response := reconcileFunctionsFunctionLifecycle(&fn.Status.OsokStatus, fnInstance, m.Log)
	if !response.IsSuccessful {
		return response, nil
	}
	if fnInstance.InvokeEndpoint != nil {
		if _, err = m.addToSecret(ctx, fn.Namespace, fn.Name, *fnInstance); err != nil {
			m.Log.InfoLog("Secret creation for FunctionsFunction endpoint failed")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return response, nil
}

// Delete handles deletion of the FunctionsFunction (called by the finalizer).
func (m *FunctionsFunctionServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	fn, err := m.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := servicemanager.ResolveResourceID(fn.Status.OsokStatus.Ocid, fn.Spec.FunctionsFunctionId)
	if err != nil {
		m.Log.InfoLog("FunctionsFunction has no OCID, nothing to delete")
		return true, nil
	}

	m.Log.InfoLog(fmt.Sprintf("Deleting FunctionsFunction %s", targetID))
	if err := m.DeleteFunction(ctx, targetID); err != nil {
		if isFunctionsNotFound(err) {
			return m.deleteFunctionSecret(ctx, fn)
		}
		m.Log.ErrorLog(err, "Error while deleting FunctionsFunction")
		return false, err
	}

	fnInstance, err := m.GetFunction(ctx, targetID, nil)
	if err != nil {
		if isFunctionsNotFound(err) {
			return m.deleteFunctionSecret(ctx, fn)
		}
		m.Log.ErrorLog(err, "Error while checking FunctionsFunction deletion")
		return false, err
	}

	if fnInstance.LifecycleState == ocifunctions.FunctionLifecycleStateDeleted {
		return m.deleteFunctionSecret(ctx, fn)
	}

	return false, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (m *FunctionsFunctionServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := m.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (m *FunctionsFunctionServiceManager) convert(obj runtime.Object) (*ociv1beta1.FunctionsFunction, error) {
	fn, ok := obj.(*ociv1beta1.FunctionsFunction)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for FunctionsFunction")
	}
	return fn, nil
}

func (m *FunctionsFunctionServiceManager) deleteFunctionSecret(ctx context.Context, fn *ociv1beta1.FunctionsFunction) (bool, error) {
	done, err := servicemanager.DeleteOwnedSecretIfPresent(ctx, m.CredentialClient, fn.Name, fn.Namespace, "FunctionsFunction", fn.Name)
	if err != nil {
		m.Log.ErrorLog(err, "Error while deleting FunctionsFunction secret")
	}
	return done, err
}

// addToSecret stores the function invoke endpoint in a Kubernetes secret.
func (m *FunctionsFunctionServiceManager) addToSecret(ctx context.Context, namespace string, fnName string,
	fn ocifunctions.Function) (bool, error) {
	m.Log.InfoLog("Creating the FunctionsFunction endpoint secret")
	credMap := getFunctionCredentialMap(fn)
	m.Log.InfoLog(fmt.Sprintf("Creating secret for FunctionsFunction %s in namespace %s", fnName, namespace))
	return servicemanager.EnsureOwnedSecret(ctx, m.CredentialClient, fnName, namespace, "FunctionsFunction", fnName, credMap)
}

func getFunctionCredentialMap(fn ocifunctions.Function) map[string][]byte {
	credMap := make(map[string][]byte)
	if fn.InvokeEndpoint != nil {
		credMap["invokeEndpoint"] = []byte(*fn.InvokeEndpoint)
	}
	if fn.Id != nil {
		credMap["functionId"] = []byte(*fn.Id)
	}
	return credMap
}

// getRetryPolicy returns a retry policy that waits while a function is in CREATING state.
func (m *FunctionsFunctionServiceManager) getRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(ocifunctions.GetFunctionResponse); ok {
			return resp.LifecycleState == ocifunctions.FunctionLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

func (m *FunctionsFunctionServiceManager) resolveFunctionInstance(ctx context.Context,
	fn *ociv1beta1.FunctionsFunction) (*ocifunctions.Function, error) {
	if strings.TrimSpace(string(fn.Spec.FunctionsFunctionId)) != "" {
		return m.bindFunction(ctx, fn)
	}
	return m.lookupOrCreateFunction(ctx, fn)
}

func (m *FunctionsFunctionServiceManager) lookupOrCreateFunction(ctx context.Context,
	fn *ociv1beta1.FunctionsFunction) (*ocifunctions.Function, error) {
	fnOcid, err := m.GetFunctionOcid(ctx, *fn)
	if err != nil {
		return nil, err
	}
	if fnOcid == nil {
		return m.createFunctionInstance(ctx, fn)
	}
	return m.loadResolvedFunction(ctx, fn, *fnOcid)
}

func (m *FunctionsFunctionServiceManager) bindFunction(ctx context.Context,
	fn *ociv1beta1.FunctionsFunction) (*ocifunctions.Function, error) {
	fnInstance, err := m.GetFunction(ctx, fn.Spec.FunctionsFunctionId, nil)
	if err != nil {
		m.Log.ErrorLog(err, "Error while getting existing FunctionsFunction")
		return nil, err
	}
	fn.Status.OsokStatus.Ocid = fn.Spec.FunctionsFunctionId
	if err := m.UpdateFunction(ctx, fn); err != nil {
		m.Log.ErrorLog(err, "Error while updating FunctionsFunction")
		return nil, err
	}
	m.Log.InfoLog(fmt.Sprintf("FunctionsFunction %s is bound/updated", safeFunctionsString(fnInstance.DisplayName)))
	return fnInstance, nil
}

func (m *FunctionsFunctionServiceManager) createFunctionInstance(ctx context.Context,
	fn *ociv1beta1.FunctionsFunction) (*ocifunctions.Function, error) {
	resp, err := m.CreateFunction(ctx, *fn)
	if err != nil {
		applyFunctionsCreateFailure(&fn.Status.OsokStatus, err, m.Log, "FunctionsFunction")
		return nil, err
	}

	m.Log.InfoLog(fmt.Sprintf("FunctionsFunction %s is Provisioning", fn.Spec.DisplayName))
	setFunctionsProvisioning(&fn.Status.OsokStatus, "FunctionsFunction", fn.Spec.DisplayName, ociv1beta1.OCID(*resp.Id), m.Log)
	retryPolicy := m.getRetryPolicy(30)
	fnInstance, err := m.GetFunction(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
	if err != nil {
		m.Log.ErrorLog(err, "Error while getting FunctionsFunction after create")
		return nil, err
	}
	return fnInstance, nil
}

func (m *FunctionsFunctionServiceManager) loadResolvedFunction(ctx context.Context,
	fn *ociv1beta1.FunctionsFunction, fnOcid ociv1beta1.OCID) (*ocifunctions.Function, error) {
	m.Log.InfoLog(fmt.Sprintf("Getting existing FunctionsFunction %s", fnOcid))
	fnInstance, err := m.GetFunction(ctx, fnOcid, nil)
	if err != nil {
		m.Log.ErrorLog(err, "Error while getting FunctionsFunction by OCID")
		return nil, err
	}
	fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(*fnInstance.Id)
	m.Log.InfoLog(fmt.Sprintf("FunctionsFunction %s is %s", safeFunctionsString(fnInstance.DisplayName), fnInstance.LifecycleState))
	return fnInstance, nil
}
