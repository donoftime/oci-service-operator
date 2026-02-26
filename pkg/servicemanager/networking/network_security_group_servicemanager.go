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

	var nsgInstance *ocicore.NetworkSecurityGroup

	if strings.TrimSpace(string(nsg.Spec.NetworkSecurityGroupId)) == "" {
		nsgOcid, err := c.GetNetworkSecurityGroupOcid(ctx, *nsg)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if nsgOcid == nil {
			nsgInstance, err = c.CreateNetworkSecurityGroup(ctx, *nsg)
			if err != nil {
				nsg.Status.OsokStatus = util.UpdateOSOKStatusCondition(nsg.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciNetworkSecurityGroup failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if nsgInstance.LifecycleState == ocicore.NetworkSecurityGroupLifecycleStateProvisioning {
				nsg.Status.OsokStatus = util.UpdateOSOKStatusCondition(nsg.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciNetworkSecurityGroup Provisioning", c.Log)
				nsg.Status.OsokStatus.Ocid = ociv1beta1.OCID(*nsgInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		} else {
			nsgInstance, err = c.GetNetworkSecurityGroup(ctx, *nsgOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciNetworkSecurityGroup by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if nsgInstance.LifecycleState == ocicore.NetworkSecurityGroupLifecycleStateProvisioning {
				c.Log.InfoLog(fmt.Sprintf("OciNetworkSecurityGroup %s is still PROVISIONING", safeString(nsgInstance.DisplayName)))
				nsg.Status.OsokStatus = util.UpdateOSOKStatusCondition(nsg.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciNetworkSecurityGroup Provisioning", c.Log)
				nsg.Status.OsokStatus.Ocid = ociv1beta1.OCID(*nsgInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		}
	} else {
		nsgInstance, err = c.GetNetworkSecurityGroup(ctx, nsg.Spec.NetworkSecurityGroupId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciNetworkSecurityGroup")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateNetworkSecurityGroup(ctx, nsg); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciNetworkSecurityGroup")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	nsg.Status.OsokStatus.Ocid = ociv1beta1.OCID(*nsgInstance.Id)
	if nsg.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		nsg.Status.OsokStatus.CreatedAt = &now
	}

	nsg.Status.OsokStatus = util.UpdateOSOKStatusCondition(nsg.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("OciNetworkSecurityGroup %s is %s", safeString(nsgInstance.DisplayName), nsgInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("OciNetworkSecurityGroup %s is %s", safeString(nsgInstance.DisplayName), nsgInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the NSG (called by the finalizer).
func (c *OciNetworkSecurityGroupServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	nsg, err := c.convertNSG(obj)
	if err != nil {
		return false, err
	}

	if nsg.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciNetworkSecurityGroup has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciNetworkSecurityGroup %s", nsg.Status.OsokStatus.Ocid))
	if err := c.DeleteNetworkSecurityGroup(ctx, nsg.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciNetworkSecurityGroup")
		return false, err
	}

	return true, nil
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
