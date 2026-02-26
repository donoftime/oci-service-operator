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

// Compile-time check that FunctionsApplicationServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &FunctionsApplicationServiceManager{}

// FunctionsApplicationServiceManager implements OSOKServiceManager for OCI Functions Applications.
type FunctionsApplicationServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        FunctionsManagementClientInterface
}

// NewFunctionsApplicationServiceManager creates a new FunctionsApplicationServiceManager.
func NewFunctionsApplicationServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *FunctionsApplicationServiceManager {
	return &FunctionsApplicationServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the FunctionsApplication resource against OCI.
func (m *FunctionsApplicationServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	app, err := m.convert(obj)
	if err != nil {
		m.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var appInstance *ocifunctions.Application

	if strings.TrimSpace(string(app.Spec.FunctionsApplicationId)) == "" {
		// No ID provided — check by display name or create
		appOcid, err := m.GetApplicationOcid(ctx, *app)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if appOcid == nil {
			// Create a new application
			resp, err := m.CreateApplication(ctx, *app)
			if err != nil {
				app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), m.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					m.Log.ErrorLog(err, "Create FunctionsApplication failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				app.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
				m.Log.ErrorLog(err, "Create FunctionsApplication bad request")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			m.Log.InfoLog(fmt.Sprintf("FunctionsApplication %s is Provisioning", app.Spec.DisplayName))
			app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "FunctionsApplication Provisioning", m.Log)
			app.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)

			retryPolicy := m.getRetryPolicy(30)
			appInstance, err = m.GetApplication(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				m.Log.ErrorLog(err, "Error while getting FunctionsApplication after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			m.Log.InfoLog(fmt.Sprintf("Getting existing FunctionsApplication %s", *appOcid))
			appInstance, err = m.GetApplication(ctx, *appOcid, nil)
			if err != nil {
				m.Log.ErrorLog(err, "Error while getting FunctionsApplication by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}

		app.Status.OsokStatus.Ocid = ociv1beta1.OCID(*appInstance.Id)
		app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("FunctionsApplication %s is %s", *appInstance.DisplayName, appInstance.LifecycleState), m.Log)
		m.Log.InfoLog(fmt.Sprintf("FunctionsApplication %s is %s", *appInstance.DisplayName, appInstance.LifecycleState))

	} else {
		// Bind to an existing application by ID
		appInstance, err = m.GetApplication(ctx, app.Spec.FunctionsApplicationId, nil)
		if err != nil {
			m.Log.ErrorLog(err, "Error while getting existing FunctionsApplication")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = m.UpdateApplication(ctx, app); err != nil {
			m.Log.ErrorLog(err, "Error while updating FunctionsApplication")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "FunctionsApplication Bound/Updated", m.Log)
		m.Log.InfoLog(fmt.Sprintf("FunctionsApplication %s is bound/updated", *appInstance.DisplayName))
	}

	app.Status.OsokStatus.Ocid = ociv1beta1.OCID(*appInstance.Id)
	if app.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		app.Status.OsokStatus.CreatedAt = &now
	}

	if appInstance.LifecycleState == ocifunctions.ApplicationLifecycleStateFailed {
		app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("FunctionsApplication %s creation Failed", *appInstance.DisplayName), m.Log)
		m.Log.InfoLog(fmt.Sprintf("FunctionsApplication %s creation Failed", *appInstance.DisplayName))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the FunctionsApplication (called by the finalizer).
func (m *FunctionsApplicationServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	app, err := m.convert(obj)
	if err != nil {
		return false, err
	}

	if app.Status.OsokStatus.Ocid == "" {
		m.Log.InfoLog("FunctionsApplication has no OCID, nothing to delete")
		return true, nil
	}

	m.Log.InfoLog(fmt.Sprintf("Deleting FunctionsApplication %s", app.Status.OsokStatus.Ocid))
	if err := m.DeleteApplication(ctx, app.Status.OsokStatus.Ocid); err != nil {
		m.Log.ErrorLog(err, "Error while deleting FunctionsApplication")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (m *FunctionsApplicationServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := m.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (m *FunctionsApplicationServiceManager) convert(obj runtime.Object) (*ociv1beta1.FunctionsApplication, error) {
	app, ok := obj.(*ociv1beta1.FunctionsApplication)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for FunctionsApplication")
	}
	return app, nil
}

// getRetryPolicy returns a retry policy that waits while an application is in CREATING state.
func (m *FunctionsApplicationServiceManager) getRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(ocifunctions.GetApplicationResponse); ok {
			return resp.LifecycleState == ocifunctions.ApplicationLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
