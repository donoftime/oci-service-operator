/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compute

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
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

// Compile-time check that ComputeInstanceServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &ComputeInstanceServiceManager{}

// ComputeInstanceServiceManager implements OSOKServiceManager for OCI Compute Instances.
type ComputeInstanceServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        ComputeInstanceClientInterface
}

// NewComputeInstanceServiceManager creates a new ComputeInstanceServiceManager.
func NewComputeInstanceServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *ComputeInstanceServiceManager {
	return &ComputeInstanceServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the ComputeInstance resource against OCI.
func (c *ComputeInstanceServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	ci, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var instance *core.Instance

	if strings.TrimSpace(string(ci.Spec.ComputeInstanceId)) == "" {
		// No ID provided — check by display name or create
		instanceOcid, err := c.GetInstanceOcid(ctx, *ci)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if instanceOcid == nil {
			// Launch a new compute instance
			resp, err := c.LaunchInstance(ctx, *ci)
			if err != nil {
				ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Launch ComputeInstance failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				ci.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
				c.Log.ErrorLog(err, "Launch ComputeInstance bad request")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			displayName := ""
			if ci.Spec.DisplayName != nil {
				displayName = *ci.Spec.DisplayName
			}
			c.Log.InfoLog(fmt.Sprintf("ComputeInstance %s is Provisioning", displayName))
			ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "ComputeInstance Provisioning", c.Log)
			ci.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)

			retryPolicy := c.getRetryPolicy(30)
			instance, err = c.GetInstance(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting ComputeInstance after launch")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting existing ComputeInstance %s", *instanceOcid))
			instance, err = c.GetInstance(ctx, *instanceOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting ComputeInstance by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}

		ci.Status.OsokStatus.Ocid = ociv1beta1.OCID(*instance.Id)
		displayName := ""
		if instance.DisplayName != nil {
			displayName = *instance.DisplayName
		}
		ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("ComputeInstance %s is %s", displayName, instance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("ComputeInstance %s is %s", displayName, instance.LifecycleState))

	} else {
		// Bind to an existing compute instance by ID
		instance, err = c.GetInstance(ctx, ci.Spec.ComputeInstanceId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing ComputeInstance")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateInstance(ctx, ci); err != nil {
			c.Log.ErrorLog(err, "Error while updating ComputeInstance")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "ComputeInstance Bound/Updated", c.Log)
		displayName := ""
		if instance.DisplayName != nil {
			displayName = *instance.DisplayName
		}
		c.Log.InfoLog(fmt.Sprintf("ComputeInstance %s is bound/updated", displayName))
	}

	ci.Status.OsokStatus.Ocid = ociv1beta1.OCID(*instance.Id)
	if ci.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		ci.Status.OsokStatus.CreatedAt = &now
	}

	if instance.LifecycleState == core.InstanceLifecycleStateTerminated {
		displayName := ""
		if instance.DisplayName != nil {
			displayName = *instance.DisplayName
		}
		ci.Status.OsokStatus = util.UpdateOSOKStatusCondition(ci.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ComputeInstance %s is Terminated", displayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("ComputeInstance %s is Terminated", displayName))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the compute instance (called by the finalizer).
func (c *ComputeInstanceServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	ci, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if ci.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("ComputeInstance has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Terminating ComputeInstance %s", ci.Status.OsokStatus.Ocid))
	if err := c.TerminateInstance(ctx, ci.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while terminating ComputeInstance")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *ComputeInstanceServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *ComputeInstanceServiceManager) convert(obj runtime.Object) (*ociv1beta1.ComputeInstance, error) {
	ci, ok := obj.(*ociv1beta1.ComputeInstance)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for ComputeInstance")
	}
	return ci, nil
}
