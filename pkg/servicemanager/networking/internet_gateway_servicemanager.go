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

	var igwInstance *ocicore.InternetGateway

	if strings.TrimSpace(string(igw.Spec.InternetGatewayId)) == "" {
		// No explicit ID — look up by display name or create.
		igwOcid, err := c.GetInternetGatewayOcid(ctx, *igw)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if igwOcid == nil {
			igwInstance, err = c.CreateInternetGateway(ctx, *igw)
			if err != nil {
				igw.Status.OsokStatus = util.UpdateOSOKStatusCondition(igw.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciInternetGateway failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if igwInstance.LifecycleState == ocicore.InternetGatewayLifecycleStateProvisioning {
				igw.Status.OsokStatus = util.UpdateOSOKStatusCondition(igw.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciInternetGateway Provisioning", c.Log)
				igw.Status.OsokStatus.Ocid = ociv1beta1.OCID(*igwInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		} else {
			igwInstance, err = c.GetInternetGateway(ctx, *igwOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciInternetGateway by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if igwInstance.LifecycleState == ocicore.InternetGatewayLifecycleStateProvisioning {
				c.Log.InfoLog(fmt.Sprintf("OciInternetGateway %s is still PROVISIONING", safeString(igwInstance.DisplayName)))
				igw.Status.OsokStatus = util.UpdateOSOKStatusCondition(igw.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciInternetGateway Provisioning", c.Log)
				igw.Status.OsokStatus.Ocid = ociv1beta1.OCID(*igwInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		}
	} else {
		// Bind to an existing Internet Gateway by ID.
		igwInstance, err = c.GetInternetGateway(ctx, igw.Spec.InternetGatewayId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciInternetGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateInternetGateway(ctx, igw); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciInternetGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	igw.Status.OsokStatus.Ocid = ociv1beta1.OCID(*igwInstance.Id)
	if igw.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		igw.Status.OsokStatus.CreatedAt = &now
	}

	igw.Status.OsokStatus = util.UpdateOSOKStatusCondition(igw.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("OciInternetGateway %s is %s", safeString(igwInstance.DisplayName), igwInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("OciInternetGateway %s is %s", safeString(igwInstance.DisplayName), igwInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the Internet Gateway (called by the finalizer).
func (c *OciInternetGatewayServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	igw, err := c.convertIGW(obj)
	if err != nil {
		return false, err
	}

	if igw.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciInternetGateway has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciInternetGateway %s", igw.Status.OsokStatus.Ocid))
	if err := c.DeleteInternetGateway(ctx, igw.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciInternetGateway")
		return false, err
	}

	return true, nil
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
