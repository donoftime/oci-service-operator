/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package devops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/devops"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/util"
)

// Compile-time check that DevopsProjectServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &DevopsProjectServiceManager{}

// DevopsProjectServiceManager implements OSOKServiceManager for OCI DevOps projects.
type DevopsProjectServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
}

// NewDevopsProjectServiceManager creates a new DevopsProjectServiceManager.
func NewDevopsProjectServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *DevopsProjectServiceManager {
	return &DevopsProjectServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the DevopsProject resource against OCI.
func (c *DevopsProjectServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	project, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var projectInstance *devops.Project

	if strings.TrimSpace(string(project.Spec.DevopsProjectId)) == "" {
		// No ID provided — check by name or create
		projectOcid, err := c.GetDevopsProjectOcid(ctx, *project)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if projectOcid == nil {
			// Create a new DevOps project
			resp, err := c.CreateDevopsProject(ctx, *project)
			if err != nil {
				project.Status.OsokStatus = util.UpdateOSOKStatusCondition(project.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create DevopsProject failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			c.Log.InfoLog(fmt.Sprintf("DevopsProject %s is Provisioning", project.Spec.Name))
			project.Status.OsokStatus = util.UpdateOSOKStatusCondition(project.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "DevopsProject Provisioning", c.Log)
			project.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)

			retryPolicy := c.getRetryPolicy(30)
			projectInstance, err = c.GetDevopsProject(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting DevopsProject after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting existing DevopsProject %s", *projectOcid))
			projectInstance, err = c.GetDevopsProject(ctx, *projectOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting DevopsProject by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
	} else {
		// Bind to an existing project by ID
		projectInstance, err = c.GetDevopsProject(ctx, project.Spec.DevopsProjectId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing DevopsProject")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateDevopsProject(ctx, project); err != nil {
			c.Log.ErrorLog(err, "Error while updating DevopsProject")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		project.Status.OsokStatus = util.UpdateOSOKStatusCondition(project.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "DevopsProject Bound/Updated", c.Log)
		c.Log.InfoLog(fmt.Sprintf("DevopsProject %s is bound/updated", *projectInstance.Name))
	}

	project.Status.OsokStatus.Ocid = ociv1beta1.OCID(*projectInstance.Id)
	if project.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		project.Status.OsokStatus.CreatedAt = &now
	}

	if projectInstance.LifecycleState == devops.ProjectLifecycleStateFailed {
		project.Status.OsokStatus = util.UpdateOSOKStatusCondition(project.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("DevopsProject %s creation Failed", *projectInstance.Name), c.Log)
		c.Log.InfoLog(fmt.Sprintf("DevopsProject %s creation Failed", *projectInstance.Name))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	project.Status.OsokStatus = util.UpdateOSOKStatusCondition(project.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("DevopsProject %s is %s", *projectInstance.Name, projectInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("DevopsProject %s is %s", *projectInstance.Name, projectInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the DevOps project (called by the finalizer).
func (c *DevopsProjectServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	project, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if project.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("DevopsProject has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting DevopsProject %s", project.Status.OsokStatus.Ocid))
	if err := c.DeleteDevopsProject(ctx, project.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting DevopsProject")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *DevopsProjectServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *DevopsProjectServiceManager) convert(obj runtime.Object) (*ociv1beta1.DevopsProject, error) {
	project, ok := obj.(*ociv1beta1.DevopsProject)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for DevopsProject")
	}
	return project, nil
}
