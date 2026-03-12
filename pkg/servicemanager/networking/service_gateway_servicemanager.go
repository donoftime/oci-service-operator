/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking

import (
	"context"
	"fmt"
	"strings"

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

// Compile-time check that OciServiceGatewayServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciServiceGatewayServiceManager{}

// OciServiceGatewayServiceManager implements OSOKServiceManager for OCI Service Gateway.
type OciServiceGatewayServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciServiceGatewayServiceManager creates a new OciServiceGatewayServiceManager.
func NewOciServiceGatewayServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciServiceGatewayServiceManager {
	return &OciServiceGatewayServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciServiceGateway resource against OCI.
func (c *OciServiceGatewayServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	sgw, err := c.convertSGW(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var sgwInstance *ocicore.ServiceGateway

	if strings.TrimSpace(string(sgw.Spec.ServiceGatewayId)) == "" {
		// No explicit ID — look up by display name or create.
		sgwOcid, err := c.GetServiceGatewayOcid(ctx, *sgw)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if sgwOcid == nil {
			sgwInstance, err = c.CreateServiceGateway(ctx, *sgw)
			if err != nil {
				sgw.Status.OsokStatus = util.UpdateOSOKStatusCondition(sgw.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciServiceGateway failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			sgwInstance, err = c.GetServiceGateway(ctx, *sgwOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciServiceGateway by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
	} else {
		// Bind to an existing Service Gateway by ID.
		sgwInstance, err = c.GetServiceGateway(ctx, sgw.Spec.ServiceGatewayId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciServiceGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		sgw.Status.OsokStatus.Ocid = sgw.Spec.ServiceGatewayId
		if err = c.UpdateServiceGateway(ctx, sgw); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciServiceGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return reconcileLifecycleStatus(&sgw.Status.OsokStatus, "OciServiceGateway", safeString(sgwInstance.DisplayName),
		string(sgwInstance.LifecycleState), ociv1beta1.OCID(*sgwInstance.Id), c.Log), nil
}

// Delete handles deletion of the Service Gateway (called by the finalizer).
func (c *OciServiceGatewayServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	sgw, err := c.convertSGW(obj)
	if err != nil {
		return false, err
	}

	resourceID := sgw.Status.OsokStatus.Ocid
	if resourceID == "" {
		resourceID = sgw.Spec.ServiceGatewayId
	}
	if resourceID == "" {
		c.Log.InfoLog("OciServiceGateway has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciServiceGateway %s", resourceID))
	done, err := deleteResourceAndWait(
		func() error { return c.DeleteServiceGateway(ctx, resourceID) },
		func() error {
			_, getErr := c.GetServiceGateway(ctx, resourceID)
			return getErr
		},
	)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciServiceGateway")
		return false, err
	}

	return done, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciServiceGatewayServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertSGW(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciServiceGatewayServiceManager) convertSGW(obj runtime.Object) (*ociv1beta1.OciServiceGateway, error) {
	sgw, ok := obj.(*ociv1beta1.OciServiceGateway)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciServiceGateway")
	}
	return sgw, nil
}
