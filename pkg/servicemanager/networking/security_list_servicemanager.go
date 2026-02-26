/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that OciSecurityListServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciSecurityListServiceManager{}

// OciSecurityListServiceManager implements OSOKServiceManager for OCI Security List.
type OciSecurityListServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciSecurityListServiceManager creates a new OciSecurityListServiceManager.
func NewOciSecurityListServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciSecurityListServiceManager {
	return &OciSecurityListServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciSecurityList resource against OCI.
func (c *OciSecurityListServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	sl, err := c.convertSecurityList(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var slInstance *ocicore.SecurityList

	if strings.TrimSpace(string(sl.Spec.SecurityListId)) == "" {
		slOcid, err := c.GetSecurityListOcid(ctx, *sl)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if slOcid == nil {
			slInstance, err = c.CreateSecurityList(ctx, *sl)
			if err != nil {
				sl.Status.OsokStatus = util.UpdateOSOKStatusCondition(sl.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciSecurityList failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if slInstance.LifecycleState == ocicore.SecurityListLifecycleStateProvisioning {
				sl.Status.OsokStatus = util.UpdateOSOKStatusCondition(sl.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciSecurityList Provisioning", c.Log)
				sl.Status.OsokStatus.Ocid = ociv1beta1.OCID(*slInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		} else {
			slInstance, err = c.GetSecurityList(ctx, *slOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciSecurityList by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if slInstance.LifecycleState == ocicore.SecurityListLifecycleStateProvisioning {
				c.Log.InfoLog(fmt.Sprintf("OciSecurityList %s is still PROVISIONING", safeString(slInstance.DisplayName)))
				sl.Status.OsokStatus = util.UpdateOSOKStatusCondition(sl.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciSecurityList Provisioning", c.Log)
				sl.Status.OsokStatus.Ocid = ociv1beta1.OCID(*slInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		}
	} else {
		slInstance, err = c.GetSecurityList(ctx, sl.Spec.SecurityListId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciSecurityList")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateSecurityList(ctx, sl); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciSecurityList")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	sl.Status.OsokStatus.Ocid = ociv1beta1.OCID(*slInstance.Id)
	if sl.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		sl.Status.OsokStatus.CreatedAt = &now
	}

	sl.Status.OsokStatus = util.UpdateOSOKStatusCondition(sl.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("OciSecurityList %s is %s", safeString(slInstance.DisplayName), slInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("OciSecurityList %s is %s", safeString(slInstance.DisplayName), slInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the Security List (called by the finalizer).
func (c *OciSecurityListServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	sl, err := c.convertSecurityList(obj)
	if err != nil {
		return false, err
	}

	if sl.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciSecurityList has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciSecurityList %s", sl.Status.OsokStatus.Ocid))
	if err := c.DeleteSecurityList(ctx, sl.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciSecurityList")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciSecurityListServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertSecurityList(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciSecurityListServiceManager) convertSecurityList(obj runtime.Object) (*ociv1beta1.OciSecurityList, error) {
	sl, ok := obj.(*ociv1beta1.OciSecurityList)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciSecurityList")
	}
	return sl, nil
}
