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

// Compile-time check that OciSubnetServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciSubnetServiceManager{}

// OciSubnetServiceManager implements OSOKServiceManager for OCI Subnet.
type OciSubnetServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciSubnetServiceManager creates a new OciSubnetServiceManager.
func NewOciSubnetServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciSubnetServiceManager {
	return &OciSubnetServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciSubnet resource against OCI.
func (c *OciSubnetServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	subnet, err := c.convertSubnet(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var subnetInstance *ocicore.Subnet

	if strings.TrimSpace(string(subnet.Spec.SubnetId)) == "" {
		// No explicit ID — look up by display name or create.
		subnetOcid, err := c.GetSubnetOcid(ctx, *subnet)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if subnetOcid == nil {
			subnetInstance, err = c.CreateSubnet(ctx, *subnet)
			if err != nil {
				subnet.Status.OsokStatus = util.UpdateOSOKStatusCondition(subnet.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciSubnet failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if subnetInstance.LifecycleState == ocicore.SubnetLifecycleStateProvisioning {
				subnet.Status.OsokStatus = util.UpdateOSOKStatusCondition(subnet.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciSubnet Provisioning", c.Log)
				subnet.Status.OsokStatus.Ocid = ociv1beta1.OCID(*subnetInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		} else {
			subnetInstance, err = c.GetSubnet(ctx, *subnetOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciSubnet by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if subnetInstance.LifecycleState == ocicore.SubnetLifecycleStateProvisioning {
				c.Log.InfoLog(fmt.Sprintf("OciSubnet %s is still PROVISIONING", safeString(subnetInstance.DisplayName)))
				subnet.Status.OsokStatus = util.UpdateOSOKStatusCondition(subnet.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciSubnet Provisioning", c.Log)
				subnet.Status.OsokStatus.Ocid = ociv1beta1.OCID(*subnetInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		}
	} else {
		// Bind to an existing subnet by ID.
		subnetInstance, err = c.GetSubnet(ctx, subnet.Spec.SubnetId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciSubnet")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateSubnet(ctx, subnet); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciSubnet")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	subnet.Status.OsokStatus.Ocid = ociv1beta1.OCID(*subnetInstance.Id)
	if subnet.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		subnet.Status.OsokStatus.CreatedAt = &now
	}

	subnet.Status.OsokStatus = util.UpdateOSOKStatusCondition(subnet.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("OciSubnet %s is %s", safeString(subnetInstance.DisplayName), subnetInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("OciSubnet %s is %s", safeString(subnetInstance.DisplayName), subnetInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the Subnet (called by the finalizer).
func (c *OciSubnetServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	subnet, err := c.convertSubnet(obj)
	if err != nil {
		return false, err
	}

	if subnet.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciSubnet has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciSubnet %s", subnet.Status.OsokStatus.Ocid))
	if err := c.DeleteSubnet(ctx, subnet.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciSubnet")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciSubnetServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertSubnet(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciSubnetServiceManager) convertSubnet(obj runtime.Object) (*ociv1beta1.OciSubnet, error) {
	subnet, ok := obj.(*ociv1beta1.OciSubnet)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciSubnet")
	}
	return subnet, nil
}
