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

			if sgwInstance.LifecycleState == ocicore.ServiceGatewayLifecycleStateProvisioning {
				sgw.Status.OsokStatus = util.UpdateOSOKStatusCondition(sgw.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciServiceGateway Provisioning", c.Log)
				sgw.Status.OsokStatus.Ocid = ociv1beta1.OCID(*sgwInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		} else {
			sgwInstance, err = c.GetServiceGateway(ctx, *sgwOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciServiceGateway by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if sgwInstance.LifecycleState == ocicore.ServiceGatewayLifecycleStateProvisioning {
				c.Log.InfoLog(fmt.Sprintf("OciServiceGateway %s is still PROVISIONING", safeString(sgwInstance.DisplayName)))
				sgw.Status.OsokStatus = util.UpdateOSOKStatusCondition(sgw.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciServiceGateway Provisioning", c.Log)
				sgw.Status.OsokStatus.Ocid = ociv1beta1.OCID(*sgwInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		}
	} else {
		// Bind to an existing Service Gateway by ID.
		sgwInstance, err = c.GetServiceGateway(ctx, sgw.Spec.ServiceGatewayId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciServiceGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateServiceGateway(ctx, sgw); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciServiceGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	sgw.Status.OsokStatus.Ocid = ociv1beta1.OCID(*sgwInstance.Id)
	if sgw.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		sgw.Status.OsokStatus.CreatedAt = &now
	}

	sgw.Status.OsokStatus = util.UpdateOSOKStatusCondition(sgw.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("OciServiceGateway %s is %s", safeString(sgwInstance.DisplayName), sgwInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("OciServiceGateway %s is %s", safeString(sgwInstance.DisplayName), sgwInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the Service Gateway (called by the finalizer).
func (c *OciServiceGatewayServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	sgw, err := c.convertSGW(obj)
	if err != nil {
		return false, err
	}

	if sgw.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciServiceGateway has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciServiceGateway %s", sgw.Status.OsokStatus.Ocid))
	if err := c.DeleteServiceGateway(ctx, sgw.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciServiceGateway")
		return false, err
	}

	return true, nil
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
