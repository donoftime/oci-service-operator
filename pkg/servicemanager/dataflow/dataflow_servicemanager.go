/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dataflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocidataflow "github.com/oracle/oci-go-sdk/v65/dataflow"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that DataFlowApplicationServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &DataFlowApplicationServiceManager{}

// DataFlowApplicationServiceManager implements OSOKServiceManager for OCI Data Flow Application.
type DataFlowApplicationServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        DataFlowClientInterface
}

// NewDataFlowApplicationServiceManager creates a new DataFlowApplicationServiceManager.
func NewDataFlowApplicationServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *DataFlowApplicationServiceManager {
	return &DataFlowApplicationServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the DataFlowApplication resource against OCI.
func (c *DataFlowApplicationServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	app, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	// Path 1: bind to existing application by spec ID
	if strings.TrimSpace(string(app.Spec.DataFlowApplicationId)) != "" {
		appInstance, err := c.GetDataFlowApplication(ctx, app.Spec.DataFlowApplicationId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting DataFlowApplication by spec ID")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		if appInstance.LifecycleState == ocidataflow.ApplicationLifecycleStateDeleted {
			return markDeletedStatus(app, appInstance, c.Log), nil
		}

		response := reconcileLifecycleStatus(app, appInstance, c.Log)
		if !response.IsSuccessful {
			return response, nil
		}
		if err := c.UpdateDataFlowApplication(ctx, app); err != nil {
			c.Log.ErrorLog(err, "Error while updating DataFlowApplication by spec ID")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		c.Log.InfoLog(fmt.Sprintf("DataFlowApplication %s is bound to existing application", safeString(appInstance.DisplayName)))
		return response, nil
	}

	// Path 2: update existing application by status OCID
	if strings.TrimSpace(string(app.Status.OsokStatus.Ocid)) != "" {
		appInstance, err := c.GetDataFlowApplication(ctx, app.Status.OsokStatus.Ocid)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting DataFlowApplication by status OCID")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if appInstance.LifecycleState == ocidataflow.ApplicationLifecycleStateDeleted {
			return markDeletedStatus(app, appInstance, c.Log), nil
		}

		response := reconcileLifecycleStatus(app, appInstance, c.Log)
		if !response.IsSuccessful {
			return response, nil
		}
		if err := c.UpdateDataFlowApplication(ctx, app); err != nil {
			c.Log.ErrorLog(err, "Error while updating DataFlowApplication")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		c.Log.InfoLog(fmt.Sprintf("DataFlowApplication %s updated", safeString(appInstance.DisplayName)))
		return response, nil
	}

	// Path 3: look up by name or create new
	existingOcid, err := c.GetDataFlowApplicationByName(ctx, *app)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	if existingOcid != nil {
		// Application exists — use it
		app.Status.OsokStatus.Ocid = *existingOcid
		appInstance, err := c.GetDataFlowApplication(ctx, *existingOcid)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		response := reconcileLifecycleStatus(app, appInstance, c.Log)
		if response.IsSuccessful {
			if err := c.UpdateDataFlowApplication(ctx, app); err != nil {
				c.Log.ErrorLog(err, "Error while updating existing DataFlowApplication")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
		c.Log.InfoLog(fmt.Sprintf("DataFlowApplication %s found existing", app.Spec.DisplayName))
		return response, nil
	}

	// Create new application
	appInstance, err := c.CreateDataFlowApplication(ctx, *app)
	if err != nil {
		app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
		c.Log.ErrorLog(err, "Create DataFlowApplication failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	c.Log.InfoLog(fmt.Sprintf("DataFlowApplication %s created successfully", app.Spec.DisplayName))

	return reconcileLifecycleStatus(app, appInstance, c.Log), nil
}

// Delete handles deletion of the DataFlowApplication (called by the finalizer).
func (c *DataFlowApplicationServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	app, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := servicemanager.ResolveResourceID(app.Status.OsokStatus.Ocid, app.Spec.DataFlowApplicationId)
	if err != nil {
		c.Log.InfoLog("DataFlowApplication has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting DataFlowApplication %s", targetID))
	if err := c.DeleteDataFlowApplication(ctx, targetID); err != nil {
		c.Log.ErrorLog(err, "Error while deleting DataFlowApplication")
		return false, err
	}

	appInstance, err := c.GetDataFlowApplication(ctx, targetID)
	if err != nil {
		if isNotFoundServiceError(err) || servicemanager.IsNotFoundErrorString(err) {
			return true, nil
		}
		return false, err
	}
	if appInstance.LifecycleState == ocidataflow.ApplicationLifecycleStateDeleted {
		return true, nil
	}

	return false, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *DataFlowApplicationServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *DataFlowApplicationServiceManager) convert(obj runtime.Object) (*ociv1beta1.DataFlowApplication, error) {
	app, ok := obj.(*ociv1beta1.DataFlowApplication)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for DataFlowApplication")
	}
	return app, nil
}
