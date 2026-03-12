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

// Compile-time check that OciDrgServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciDrgServiceManager{}

// OciDrgServiceManager implements OSOKServiceManager for OCI DRG.
type OciDrgServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciDrgServiceManager creates a new OciDrgServiceManager.
func NewOciDrgServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciDrgServiceManager {
	return &OciDrgServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciDrg resource against OCI.
func (c *OciDrgServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	drg, err := c.convertDRG(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	drgInstance, err := reconcileNetworkingResource(networkingCreateOrUpdateOps[ocicore.Drg]{
		SpecID: drg.Spec.DrgId,
		Status: &drg.Status.OsokStatus,
		Get: func(id ociv1beta1.OCID) (*ocicore.Drg, error) {
			return c.GetDrg(ctx, id)
		},
		Update: func() error {
			return c.UpdateDrg(ctx, drg)
		},
		Lookup: func() (*ociv1beta1.OCID, error) {
			return c.GetDrgOcid(ctx, *drg)
		},
		Create: func() (*ocicore.Drg, error) {
			return c.CreateDrg(ctx, *drg)
		},
		OnCreateError: func(err error) {
			drg.Status.OsokStatus = util.UpdateOSOKStatusCondition(drg.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
			c.Log.ErrorLog(err, "Create OciDrg failed")
		},
		Log:            c.Log,
		GetExistingMsg: "Error while getting existing OciDrg",
		GetStatusMsg:   "Error while getting existing OciDrg from status OCID",
		GetByOCIDMsg:   "Error while getting OciDrg by OCID",
		UpdateMsg:      "Error while updating OciDrg",
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return reconcileLifecycleStatus(&drg.Status.OsokStatus, "OciDrg", safeString(drgInstance.DisplayName),
		string(drgInstance.LifecycleState), ociv1beta1.OCID(*drgInstance.Id), c.Log), nil
}

// Delete handles deletion of the DRG (called by the finalizer).
func (c *OciDrgServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	drg, err := c.convertDRG(obj)
	if err != nil {
		return false, err
	}

	resourceID := drg.Status.OsokStatus.Ocid
	if resourceID == "" {
		resourceID = drg.Spec.DrgId
	}
	if resourceID == "" {
		c.Log.InfoLog("OciDrg has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciDrg %s", resourceID))
	done, err := deleteResourceAndWait(
		func() error { return c.DeleteDrg(ctx, resourceID) },
		func() error {
			_, getErr := c.GetDrg(ctx, resourceID)
			return getErr
		},
	)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciDrg")
		return false, err
	}

	return done, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciDrgServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertDRG(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciDrgServiceManager) convertDRG(obj runtime.Object) (*ociv1beta1.OciDrg, error) {
	drg, ok := obj.(*ociv1beta1.OciDrg)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciDrg")
	}
	return drg, nil
}
