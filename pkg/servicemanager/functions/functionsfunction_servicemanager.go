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
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	var fnInstance *ocifunctions.Function

	if strings.TrimSpace(string(fn.Spec.FunctionsFunctionId)) == "" {
		// No ID provided — check by display name or create
		fnOcid, err := m.GetFunctionOcid(ctx, *fn)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if fnOcid == nil {
			// Create a new function
			resp, err := m.CreateFunction(ctx, *fn)
			if err != nil {
				fn.Status.OsokStatus = util.UpdateOSOKStatusCondition(fn.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), m.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					m.Log.ErrorLog(err, "Create FunctionsFunction failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				fn.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
				m.Log.ErrorLog(err, "Create FunctionsFunction bad request")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			m.Log.InfoLog(fmt.Sprintf("FunctionsFunction %s is Provisioning", fn.Spec.DisplayName))
			fn.Status.OsokStatus = util.UpdateOSOKStatusCondition(fn.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "FunctionsFunction Provisioning", m.Log)
			fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)

			retryPolicy := m.getRetryPolicy(30)
			fnInstance, err = m.GetFunction(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				m.Log.ErrorLog(err, "Error while getting FunctionsFunction after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			m.Log.InfoLog(fmt.Sprintf("Getting existing FunctionsFunction %s", *fnOcid))
			fnInstance, err = m.GetFunction(ctx, *fnOcid, nil)
			if err != nil {
				m.Log.ErrorLog(err, "Error while getting FunctionsFunction by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}

		fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(*fnInstance.Id)
		fn.Status.OsokStatus = util.UpdateOSOKStatusCondition(fn.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("FunctionsFunction %s is %s", *fnInstance.DisplayName, fnInstance.LifecycleState), m.Log)
		m.Log.InfoLog(fmt.Sprintf("FunctionsFunction %s is %s", *fnInstance.DisplayName, fnInstance.LifecycleState))

	} else {
		// Bind to an existing function by ID
		fnInstance, err = m.GetFunction(ctx, fn.Spec.FunctionsFunctionId, nil)
		if err != nil {
			m.Log.ErrorLog(err, "Error while getting existing FunctionsFunction")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = m.UpdateFunction(ctx, fn); err != nil {
			m.Log.ErrorLog(err, "Error while updating FunctionsFunction")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		fn.Status.OsokStatus = util.UpdateOSOKStatusCondition(fn.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "FunctionsFunction Bound/Updated", m.Log)
		m.Log.InfoLog(fmt.Sprintf("FunctionsFunction %s is bound/updated", *fnInstance.DisplayName))
	}

	fn.Status.OsokStatus.Ocid = ociv1beta1.OCID(*fnInstance.Id)
	if fn.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		fn.Status.OsokStatus.CreatedAt = &now
	}

	if fnInstance.LifecycleState == ocifunctions.FunctionLifecycleStateFailed {
		fn.Status.OsokStatus = util.UpdateOSOKStatusCondition(fn.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("FunctionsFunction %s creation Failed", *fnInstance.DisplayName), m.Log)
		m.Log.InfoLog(fmt.Sprintf("FunctionsFunction %s creation Failed", *fnInstance.DisplayName))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	// Store invoke endpoint in a secret for easy access
	if fnInstance.InvokeEndpoint != nil {
		if _, err = m.addToSecret(ctx, fn.Namespace, fn.Name, *fnInstance); err != nil {
			m.Log.InfoLog("Secret creation for FunctionsFunction endpoint failed")
		}
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the FunctionsFunction (called by the finalizer).
func (m *FunctionsFunctionServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	fn, err := m.convert(obj)
	if err != nil {
		return false, err
	}

	if fn.Status.OsokStatus.Ocid == "" {
		m.Log.InfoLog("FunctionsFunction has no OCID, nothing to delete")
		return true, nil
	}

	m.Log.InfoLog(fmt.Sprintf("Deleting FunctionsFunction %s", fn.Status.OsokStatus.Ocid))
	if err := m.DeleteFunction(ctx, fn.Status.OsokStatus.Ocid); err != nil {
		m.Log.ErrorLog(err, "Error while deleting FunctionsFunction")
		return false, err
	}

	if _, err := m.CredentialClient.DeleteSecret(ctx, fn.Name, fn.Namespace); err != nil {
		m.Log.ErrorLog(err, "Error while deleting FunctionsFunction secret")
	}

	return true, nil
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

// addToSecret stores the function invoke endpoint in a Kubernetes secret.
func (m *FunctionsFunctionServiceManager) addToSecret(ctx context.Context, namespace string, fnName string,
	fn ocifunctions.Function) (bool, error) {
	m.Log.InfoLog("Creating the FunctionsFunction endpoint secret")
	credMap := getFunctionCredentialMap(fn)
	m.Log.InfoLog(fmt.Sprintf("Creating secret for FunctionsFunction %s in namespace %s", fnName, namespace))
	return m.CredentialClient.CreateSecret(ctx, fnName, namespace, nil, credMap)
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
