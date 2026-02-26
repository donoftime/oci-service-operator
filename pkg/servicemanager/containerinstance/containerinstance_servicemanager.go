/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/util"
)

// Compile-time check that ContainerInstanceServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &ContainerInstanceServiceManager{}

// ContainerInstanceServiceManager implements OSOKServiceManager for OCI Container Instances.
type ContainerInstanceServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        ContainerInstanceClientInterface
}

// NewContainerInstanceServiceManager creates a new ContainerInstanceServiceManager.
func NewContainerInstanceServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *ContainerInstanceServiceManager {
	return &ContainerInstanceServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the ContainerInstance resource against OCI.
func (c *ContainerInstanceServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	ci, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var ciInstance *containerinstances.ContainerInstance

	if strings.TrimSpace(string(ci.Spec.ContainerInstanceId)) == "" {
		// No ID provided — check by display name or create
		ciOcid, err := c.GetContainerInstanceOcid(ctx, *ci)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if ciOcid == nil {
			// Create a new container instance
			resp, err := c.CreateContainerInstance(ctx, *ci)
			if err != nil {
				ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Create ContainerInstance failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				ci.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
				c.Log.ErrorLog(err, "Create ContainerInstance bad request")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			displayName := ""
			if ci.Spec.DisplayName != nil {
				displayName = *ci.Spec.DisplayName
			}
			c.Log.InfoLog(fmt.Sprintf("ContainerInstance %s is Provisioning", displayName))
			ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "ContainerInstance Provisioning", c.Log)
			ci.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)

			retryPolicy := c.getRetryPolicy(30)
			ciInstance, err = c.GetContainerInstance(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting ContainerInstance after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting existing ContainerInstance %s", *ciOcid))
			ciInstance, err = c.GetContainerInstance(ctx, *ciOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting ContainerInstance by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}

		ci.Status.OsokStatus.Ocid = ociv1beta1.OCID(*ciInstance.Id)
		displayName := ""
		if ciInstance.DisplayName != nil {
			displayName = *ciInstance.DisplayName
		}
		ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("ContainerInstance %s is %s", displayName, ciInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("ContainerInstance %s is %s", displayName, ciInstance.LifecycleState))

	} else {
		// Bind to an existing container instance by ID
		ciInstance, err = c.GetContainerInstance(ctx, ci.Spec.ContainerInstanceId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing ContainerInstance")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateContainerInstance(ctx, ci); err != nil {
			c.Log.ErrorLog(err, "Error while updating ContainerInstance")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "ContainerInstance Bound/Updated", c.Log)
		displayName := ""
		if ciInstance.DisplayName != nil {
			displayName = *ciInstance.DisplayName
		}
		c.Log.InfoLog(fmt.Sprintf("ContainerInstance %s is bound/updated", displayName))
	}

	ci.Status.OsokStatus.Ocid = ociv1beta1.OCID(*ciInstance.Id)
	if ci.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		ci.Status.OsokStatus.CreatedAt = &now
	}

	if ciInstance.LifecycleState == containerinstances.ContainerInstanceLifecycleStateFailed {
		displayName := ""
		if ciInstance.DisplayName != nil {
			displayName = *ciInstance.DisplayName
		}
		ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ContainerInstance %s creation Failed", displayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("ContainerInstance %s creation Failed", displayName))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the container instance (called by the finalizer).
func (c *ContainerInstanceServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	ci, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if ci.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("ContainerInstance has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting ContainerInstance %s", ci.Status.OsokStatus.Ocid))
	if err := c.DeleteContainerInstance(ctx, ci.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting ContainerInstance")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *ContainerInstanceServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *ContainerInstanceServiceManager) convert(obj runtime.Object) (*ociv1beta1.ContainerInstance, error) {
	ci, ok := obj.(*ociv1beta1.ContainerInstance)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for ContainerInstance")
	}
	return ci, nil
}
