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

// Compile-time check that OciInternetGatewayServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciInternetGatewayServiceManager{}

// OciInternetGatewayServiceManager implements OSOKServiceManager for OCI Internet Gateway.
type OciInternetGatewayServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciInternetGatewayServiceManager creates a new OciInternetGatewayServiceManager.
func NewOciInternetGatewayServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciInternetGatewayServiceManager {
	return &OciInternetGatewayServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciInternetGateway resource against OCI.
func (c *OciInternetGatewayServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	igw, err := c.convertIGW(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	igwInstance, err := reconcileNetworkingResource(networkingCreateOrUpdateOps[ocicore.InternetGateway]{
		SpecID: igw.Spec.InternetGatewayId,
		Status: &igw.Status.OsokStatus,
		Get: func(id ociv1beta1.OCID) (*ocicore.InternetGateway, error) {
			return c.GetInternetGateway(ctx, id)
		},
		Update: func() error {
			return c.UpdateInternetGateway(ctx, igw)
		},
		Lookup: func() (*ociv1beta1.OCID, error) {
			return c.GetInternetGatewayOcid(ctx, *igw)
		},
		Create: func() (*ocicore.InternetGateway, error) {
			return c.CreateInternetGateway(ctx, *igw)
		},
		OnCreateError: func(err error) {
			igw.Status.OsokStatus = util.UpdateOSOKStatusCondition(igw.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
			c.Log.ErrorLog(err, "Create OciInternetGateway failed")
		},
		Log:            c.Log,
		GetExistingMsg: "Error while getting existing OciInternetGateway",
		GetStatusMsg:   "Error while getting existing OciInternetGateway from status OCID",
		GetByOCIDMsg:   "Error while getting OciInternetGateway by OCID",
		UpdateMsg:      "Error while updating OciInternetGateway",
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return reconcileLifecycleStatus(&igw.Status.OsokStatus, "OciInternetGateway", safeString(igwInstance.DisplayName),
		string(igwInstance.LifecycleState), ociv1beta1.OCID(*igwInstance.Id), c.Log), nil
}

// Delete handles deletion of the Internet Gateway (called by the finalizer).
func (c *OciInternetGatewayServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	igw, err := c.convertIGW(obj)
	if err != nil {
		return false, err
	}

	resourceID := igw.Status.OsokStatus.Ocid
	if resourceID == "" {
		resourceID = igw.Spec.InternetGatewayId
	}
	if resourceID == "" {
		c.Log.InfoLog("OciInternetGateway has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciInternetGateway %s", resourceID))
	done, err := deleteResourceAndWait(
		func() error { return c.DeleteInternetGateway(ctx, resourceID) },
		func() error {
			_, getErr := c.GetInternetGateway(ctx, resourceID)
			return getErr
		},
	)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciInternetGateway")
		return false, err
	}

	return done, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciInternetGatewayServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertIGW(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciInternetGatewayServiceManager) convertIGW(obj runtime.Object) (*ociv1beta1.OciInternetGateway, error) {
	igw, ok := obj.(*ociv1beta1.OciInternetGateway)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciInternetGateway")
	}
	return igw, nil
}
