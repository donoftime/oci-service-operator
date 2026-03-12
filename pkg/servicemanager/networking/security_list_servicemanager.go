/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicore "github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
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

	slInstance, err := reconcileNetworkingResource(networkingCreateOrUpdateOps[ocicore.SecurityList]{
		SpecID: sl.Spec.SecurityListId,
		Status: &sl.Status.OsokStatus,
		Get: func(id ociv1beta1.OCID) (*ocicore.SecurityList, error) {
			return c.GetSecurityList(ctx, id)
		},
		Update: func() error {
			return c.UpdateSecurityList(ctx, sl)
		},
		Lookup: func() (*ociv1beta1.OCID, error) {
			return c.GetSecurityListOcid(ctx, *sl)
		},
		Create: func() (*ocicore.SecurityList, error) {
			return c.CreateSecurityList(ctx, *sl)
		},
		OnCreateError: func(err error) {
			sl.Status.OsokStatus = util.UpdateOSOKStatusCondition(sl.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
			c.Log.ErrorLog(err, "Create OciSecurityList failed")
		},
		Log:            c.Log,
		GetExistingMsg: "Error while getting existing OciSecurityList",
		GetStatusMsg:   "Error while getting existing OciSecurityList from status OCID",
		GetByOCIDMsg:   "Error while getting OciSecurityList by OCID",
		UpdateMsg:      "Error while updating OciSecurityList",
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return reconcileLifecycleStatus(&sl.Status.OsokStatus, "OciSecurityList", safeString(slInstance.DisplayName),
		string(slInstance.LifecycleState), ociv1beta1.OCID(*slInstance.Id), c.Log), nil
}

// Delete handles deletion of the Security List (called by the finalizer).
func (c *OciSecurityListServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	sl, err := c.convertSecurityList(obj)
	if err != nil {
		return false, err
	}

	resourceID := sl.Status.OsokStatus.Ocid
	if resourceID == "" {
		resourceID = sl.Spec.SecurityListId
	}
	if resourceID == "" {
		c.Log.InfoLog("OciSecurityList has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciSecurityList %s", resourceID))
	done, err := deleteResourceAndWait(
		func() error { return c.DeleteSecurityList(ctx, resourceID) },
		func() error {
			_, getErr := c.GetSecurityList(ctx, resourceID)
			return getErr
		},
	)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciSecurityList")
		return false, err
	}

	return done, nil
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
