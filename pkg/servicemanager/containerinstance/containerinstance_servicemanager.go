/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
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

	ciInstance, response, err := c.resolveContainerInstance(ctx, ci)
	if err != nil || ciInstance == nil {
		return response, err
	}

	return c.finalizeCreateOrUpdate(ctx, ci, ciInstance), nil
}

func (c *ContainerInstanceServiceManager) resolveContainerInstance(ctx context.Context, ci *ociv1beta1.ContainerInstance) (*containerinstances.ContainerInstance, servicemanager.OSOKResponse, error) {
	if hasContainerInstanceID(ci) {
		return c.bindContainerInstance(ctx, ci)
	}
	if strings.TrimSpace(string(ci.Status.OsokStatus.Ocid)) != "" {
		ciInstance, err := c.GetContainerInstance(ctx, ci.Status.OsokStatus.Ocid, nil)
		if err != nil {
			if !isNotFoundServiceError(err) {
				c.Log.ErrorLog(err, "Error while getting existing ContainerInstance from status OCID")
				return nil, servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			ci.Status.OsokStatus.Ocid = ""
		} else {
			if err := c.UpdateContainerInstance(ctx, ci); err != nil {
				c.Log.ErrorLog(err, "Error while updating ContainerInstance from status OCID")
				return nil, servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			return ciInstance, servicemanager.OSOKResponse{}, nil
		}
	}
	return c.lookupOrCreateContainerInstance(ctx, ci)
}

func hasContainerInstanceID(ci *ociv1beta1.ContainerInstance) bool {
	return strings.TrimSpace(string(ci.Spec.ContainerInstanceId)) != ""
}

func (c *ContainerInstanceServiceManager) lookupOrCreateContainerInstance(ctx context.Context, ci *ociv1beta1.ContainerInstance) (*containerinstances.ContainerInstance, servicemanager.OSOKResponse, error) {
	ciOcid, err := c.GetContainerInstanceOcid(ctx, *ci)
	if err != nil {
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if ciOcid == nil {
		return c.createNewContainerInstance(ctx, ci)
	}

	c.Log.InfoLog(fmt.Sprintf("Getting existing ContainerInstance %s", *ciOcid))
	ciInstance, err := c.GetContainerInstance(ctx, *ciOcid, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting ContainerInstance by OCID")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	ci.Status.OsokStatus.Ocid = *ciOcid
	if err := c.UpdateContainerInstance(ctx, ci); err != nil {
		c.Log.ErrorLog(err, "Error while updating ContainerInstance by resolved OCID")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return ciInstance, servicemanager.OSOKResponse{}, nil
}

func (c *ContainerInstanceServiceManager) createNewContainerInstance(ctx context.Context, ci *ociv1beta1.ContainerInstance) (*containerinstances.ContainerInstance, servicemanager.OSOKResponse, error) {
	resp, err := c.CreateContainerInstance(ctx, *ci)
	if err != nil {
		response, handleErr := c.handleCreateError(ctx, ci, err)
		return nil, response, handleErr
	}

	containerInstanceID := ociv1beta1.OCID(*resp.Id)
	c.Log.InfoLog(fmt.Sprintf("ContainerInstance %s is Provisioning", safeString(ci.Spec.DisplayName)))
	ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
		ociv1beta1.Provisioning, v1.ConditionTrue, "", "ContainerInstance Provisioning", c.Log)
	ci.Status.OsokStatus.Ocid = containerInstanceID

	retryPolicy := c.getRetryPolicy(30)
	ciInstance, getErr := c.GetContainerInstance(ctx, containerInstanceID, &retryPolicy)
	if getErr != nil {
		c.Log.ErrorLog(getErr, "Error while getting ContainerInstance after create")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, getErr
	}
	return ciInstance, servicemanager.OSOKResponse{}, nil
}

func (c *ContainerInstanceServiceManager) handleCreateError(ctx context.Context, ci *ociv1beta1.ContainerInstance, err error) (servicemanager.OSOKResponse, error) {
	c.runGarbageCollect(ctx, *ci)
	ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
		ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)

	if serviceErr, ok := common.IsServiceError(err); ok && serviceErr.GetHTTPStatusCode() == 400 {
		ci.Status.OsokStatus.Message = serviceErr.GetCode()
		c.Log.ErrorLog(err, "Create ContainerInstance bad request")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	c.Log.ErrorLog(err, "Create ContainerInstance failed")
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (c *ContainerInstanceServiceManager) bindContainerInstance(ctx context.Context, ci *ociv1beta1.ContainerInstance) (*containerinstances.ContainerInstance, servicemanager.OSOKResponse, error) {
	ciInstance, err := c.GetContainerInstance(ctx, ci.Spec.ContainerInstanceId, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting existing ContainerInstance")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	ci.Status.OsokStatus.Ocid = ci.Spec.ContainerInstanceId

	if err = c.UpdateContainerInstance(ctx, ci); err != nil {
		c.Log.ErrorLog(err, "Error while updating ContainerInstance")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	return ciInstance, servicemanager.OSOKResponse{}, nil
}

func (c *ContainerInstanceServiceManager) finalizeCreateOrUpdate(ctx context.Context, ci *ociv1beta1.ContainerInstance, ciInstance *containerinstances.ContainerInstance) servicemanager.OSOKResponse {
	response := reconcileLifecycleStatus(&ci.Status.OsokStatus, ciInstance, c.Log)
	c.runGarbageCollect(ctx, *ci)
	return response
}

func (c *ContainerInstanceServiceManager) runGarbageCollect(ctx context.Context, ci ociv1beta1.ContainerInstance) {
	if err := c.GarbageCollect(ctx, ci); err != nil {
		c.Log.ErrorLog(err, "ContainerInstance GC failed (non-fatal)")
	}
}

// Delete handles deletion of the container instance (called by the finalizer).
func (c *ContainerInstanceServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	ci, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := resolveContainerInstanceID(ci.Status.OsokStatus.Ocid, ci.Spec.ContainerInstanceId)
	if err != nil {
		c.Log.InfoLog("ContainerInstance has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting ContainerInstance %s", targetID))
	if err := c.DeleteContainerInstance(ctx, targetID); err != nil {
		if isNotFoundServiceError(err) {
			return true, nil
		}
		c.Log.ErrorLog(err, "Error while deleting ContainerInstance")
		return false, err
	}

	instance, err := c.GetContainerInstance(ctx, targetID, nil)
	if err != nil {
		if isNotFoundServiceError(err) {
			return true, nil
		}
		return false, err
	}
	if instance.LifecycleState == containerinstances.ContainerInstanceLifecycleStateDeleted {
		return true, nil
	}

	return false, nil
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
