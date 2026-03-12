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

	appInstance, err := m.resolveApplicationInstance(ctx, app)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if appInstance.Id != nil {
		app.Status.OsokStatus.Ocid = ociv1beta1.OCID(*appInstance.Id)
	}
	servicemanager.SetCreatedAtIfUnset(&app.Status.OsokStatus)

	return reconcileFunctionsApplicationLifecycle(&app.Status.OsokStatus, appInstance, m.Log), nil
}

// Delete handles deletion of the FunctionsApplication (called by the finalizer).
func (m *FunctionsApplicationServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	app, err := m.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := servicemanager.ResolveResourceID(app.Status.OsokStatus.Ocid, app.Spec.FunctionsApplicationId)
	if err != nil {
		m.Log.InfoLog("FunctionsApplication has no OCID, nothing to delete")
		return true, nil
	}

	m.Log.InfoLog(fmt.Sprintf("Deleting FunctionsApplication %s", targetID))
	if err := m.DeleteApplication(ctx, targetID); err != nil {
		if isFunctionsNotFound(err) {
			return true, nil
		}
		m.Log.ErrorLog(err, "Error while deleting FunctionsApplication")
		return false, err
	}

	appInstance, err := m.GetApplication(ctx, targetID, nil)
	if err != nil {
		if isFunctionsNotFound(err) {
			return true, nil
		}
		m.Log.ErrorLog(err, "Error while checking FunctionsApplication deletion")
		return false, err
	}

	if appInstance.LifecycleState == ocifunctions.ApplicationLifecycleStateDeleted {
		return true, nil
	}
	return false, nil
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

func (m *FunctionsApplicationServiceManager) resolveApplicationInstance(ctx context.Context,
	app *ociv1beta1.FunctionsApplication) (*ocifunctions.Application, error) {
	if strings.TrimSpace(string(app.Spec.FunctionsApplicationId)) != "" {
		return m.bindApplication(ctx, app)
	}
	return m.lookupOrCreateApplication(ctx, app)
}

func (m *FunctionsApplicationServiceManager) lookupOrCreateApplication(ctx context.Context,
	app *ociv1beta1.FunctionsApplication) (*ocifunctions.Application, error) {
	appOcid, err := m.GetApplicationOcid(ctx, *app)
	if err != nil {
		return nil, err
	}
	if appOcid == nil {
		return m.createApplicationInstance(ctx, app)
	}
	return m.loadResolvedApplication(ctx, app, *appOcid)
}

func (m *FunctionsApplicationServiceManager) bindApplication(ctx context.Context,
	app *ociv1beta1.FunctionsApplication) (*ocifunctions.Application, error) {
	appInstance, err := m.GetApplication(ctx, app.Spec.FunctionsApplicationId, nil)
	if err != nil {
		m.Log.ErrorLog(err, "Error while getting existing FunctionsApplication")
		return nil, err
	}
	app.Status.OsokStatus.Ocid = app.Spec.FunctionsApplicationId
	if err := m.UpdateApplication(ctx, app); err != nil {
		m.Log.ErrorLog(err, "Error while updating FunctionsApplication")
		return nil, err
	}
	m.Log.InfoLog(fmt.Sprintf("FunctionsApplication %s is bound/updated", safeFunctionsString(appInstance.DisplayName)))
	return appInstance, nil
}

func (m *FunctionsApplicationServiceManager) createApplicationInstance(ctx context.Context,
	app *ociv1beta1.FunctionsApplication) (*ocifunctions.Application, error) {
	resp, err := m.CreateApplication(ctx, *app)
	if err != nil {
		applyFunctionsCreateFailure(&app.Status.OsokStatus, err, m.Log, "FunctionsApplication")
		return nil, err
	}

	m.Log.InfoLog(fmt.Sprintf("FunctionsApplication %s is Provisioning", app.Spec.DisplayName))
	setFunctionsProvisioning(&app.Status.OsokStatus, "FunctionsApplication", app.Spec.DisplayName, ociv1beta1.OCID(*resp.Id), m.Log)
	retryPolicy := m.getRetryPolicy(30)
	appInstance, err := m.GetApplication(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
	if err != nil {
		m.Log.ErrorLog(err, "Error while getting FunctionsApplication after create")
		return nil, err
	}
	return appInstance, nil
}

func (m *FunctionsApplicationServiceManager) loadResolvedApplication(ctx context.Context,
	app *ociv1beta1.FunctionsApplication, appOcid ociv1beta1.OCID) (*ocifunctions.Application, error) {
	m.Log.InfoLog(fmt.Sprintf("Getting existing FunctionsApplication %s", appOcid))
	appInstance, err := m.GetApplication(ctx, appOcid, nil)
	if err != nil {
		m.Log.ErrorLog(err, "Error while getting FunctionsApplication by OCID")
		return nil, err
	}
	app.Status.OsokStatus.Ocid = ociv1beta1.OCID(*appInstance.Id)
	if err := m.UpdateApplication(ctx, app); err != nil {
		m.Log.ErrorLog(err, "Error while updating FunctionsApplication by resolved OCID")
		return nil, err
	}
	m.Log.InfoLog(fmt.Sprintf("FunctionsApplication %s is %s", safeFunctionsString(appInstance.DisplayName), appInstance.LifecycleState))
	return appInstance, nil
}
