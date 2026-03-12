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

// Compile-time check that OciNetworkSecurityGroupServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciNetworkSecurityGroupServiceManager{}

// OciNetworkSecurityGroupServiceManager implements OSOKServiceManager for OCI NSG.
type OciNetworkSecurityGroupServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciNetworkSecurityGroupServiceManager creates a new OciNetworkSecurityGroupServiceManager.
func NewOciNetworkSecurityGroupServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciNetworkSecurityGroupServiceManager {
	return &OciNetworkSecurityGroupServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciNetworkSecurityGroup resource against OCI.
func (c *OciNetworkSecurityGroupServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	nsg, err := c.convertNSG(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	nsgInstance, err := reconcileNetworkingResource(networkingCreateOrUpdateOps[ocicore.NetworkSecurityGroup]{
		SpecID: nsg.Spec.NetworkSecurityGroupId,
		Status: &nsg.Status.OsokStatus,
		Get: func(id ociv1beta1.OCID) (*ocicore.NetworkSecurityGroup, error) {
			return c.GetNetworkSecurityGroup(ctx, id)
		},
		Update: func() error {
			return c.UpdateNetworkSecurityGroup(ctx, nsg)
		},
		Lookup: func() (*ociv1beta1.OCID, error) {
			return c.GetNetworkSecurityGroupOcid(ctx, *nsg)
		},
		Create: func() (*ocicore.NetworkSecurityGroup, error) {
			return c.CreateNetworkSecurityGroup(ctx, *nsg)
		},
		OnCreateError: func(err error) {
			nsg.Status.OsokStatus = util.UpdateOSOKStatusCondition(nsg.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
			c.Log.ErrorLog(err, "Create OciNetworkSecurityGroup failed")
		},
		Log:            c.Log,
		GetExistingMsg: "Error while getting existing OciNetworkSecurityGroup",
		GetStatusMsg:   "Error while getting existing OciNetworkSecurityGroup from status OCID",
		GetByOCIDMsg:   "Error while getting OciNetworkSecurityGroup by OCID",
		UpdateMsg:      "Error while updating OciNetworkSecurityGroup",
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return reconcileLifecycleStatus(&nsg.Status.OsokStatus, "OciNetworkSecurityGroup", safeString(nsgInstance.DisplayName),
		string(nsgInstance.LifecycleState), ociv1beta1.OCID(*nsgInstance.Id), c.Log), nil
}

// Delete handles deletion of the NSG (called by the finalizer).
func (c *OciNetworkSecurityGroupServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	nsg, err := c.convertNSG(obj)
	if err != nil {
		return false, err
	}

	resourceID := nsg.Status.OsokStatus.Ocid
	if resourceID == "" {
		resourceID = nsg.Spec.NetworkSecurityGroupId
	}
	if resourceID == "" {
		c.Log.InfoLog("OciNetworkSecurityGroup has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciNetworkSecurityGroup %s", resourceID))
	done, err := deleteResourceAndWait(
		func() error { return c.DeleteNetworkSecurityGroup(ctx, resourceID) },
		func() error {
			_, getErr := c.GetNetworkSecurityGroup(ctx, resourceID)
			return getErr
		},
	)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciNetworkSecurityGroup")
		return false, err
	}

	return done, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciNetworkSecurityGroupServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertNSG(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciNetworkSecurityGroupServiceManager) convertNSG(obj runtime.Object) (*ociv1beta1.OciNetworkSecurityGroup, error) {
	nsg, ok := obj.(*ociv1beta1.OciNetworkSecurityGroup)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciNetworkSecurityGroup")
	}
	return nsg, nil
}
