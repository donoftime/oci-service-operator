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

	var drgInstance *ocicore.Drg

	if strings.TrimSpace(string(drg.Spec.DrgId)) == "" {
		// No explicit ID — look up by display name or create.
		drgOcid, err := c.GetDrgOcid(ctx, *drg)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if drgOcid == nil {
			drgInstance, err = c.CreateDrg(ctx, *drg)
			if err != nil {
				drg.Status.OsokStatus = util.UpdateOSOKStatusCondition(drg.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciDrg failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if drgInstance.LifecycleState == ocicore.DrgLifecycleStateProvisioning {
				drg.Status.OsokStatus = util.UpdateOSOKStatusCondition(drg.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciDrg Provisioning", c.Log)
				drg.Status.OsokStatus.Ocid = ociv1beta1.OCID(*drgInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		} else {
			drgInstance, err = c.GetDrg(ctx, *drgOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciDrg by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if drgInstance.LifecycleState == ocicore.DrgLifecycleStateProvisioning {
				c.Log.InfoLog(fmt.Sprintf("OciDrg %s is still PROVISIONING", safeString(drgInstance.DisplayName)))
				drg.Status.OsokStatus = util.UpdateOSOKStatusCondition(drg.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciDrg Provisioning", c.Log)
				drg.Status.OsokStatus.Ocid = ociv1beta1.OCID(*drgInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		}
	} else {
		// Bind to an existing DRG by ID.
		drgInstance, err = c.GetDrg(ctx, drg.Spec.DrgId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciDrg")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateDrg(ctx, drg); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciDrg")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	drg.Status.OsokStatus.Ocid = ociv1beta1.OCID(*drgInstance.Id)
	if drg.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		drg.Status.OsokStatus.CreatedAt = &now
	}

	drg.Status.OsokStatus = util.UpdateOSOKStatusCondition(drg.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("OciDrg %s is %s", safeString(drgInstance.DisplayName), drgInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("OciDrg %s is %s", safeString(drgInstance.DisplayName), drgInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the DRG (called by the finalizer).
func (c *OciDrgServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	drg, err := c.convertDRG(obj)
	if err != nil {
		return false, err
	}

	if drg.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciDrg has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciDrg %s", drg.Status.OsokStatus.Ocid))
	if err := c.DeleteDrg(ctx, drg.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciDrg")
		return false, err
	}

	return true, nil
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
