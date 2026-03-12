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
		} else {
			subnetInstance, err = c.GetSubnet(ctx, *subnetOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciSubnet by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
	} else {
		// Bind to an existing subnet by ID.
		subnetInstance, err = c.GetSubnet(ctx, subnet.Spec.SubnetId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciSubnet")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		subnet.Status.OsokStatus.Ocid = subnet.Spec.SubnetId
		if err = c.UpdateSubnet(ctx, subnet); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciSubnet")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return reconcileLifecycleStatus(&subnet.Status.OsokStatus, "OciSubnet", safeString(subnetInstance.DisplayName),
		string(subnetInstance.LifecycleState), ociv1beta1.OCID(*subnetInstance.Id), c.Log), nil
}

// Delete handles deletion of the Subnet (called by the finalizer).
func (c *OciSubnetServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	subnet, err := c.convertSubnet(obj)
	if err != nil {
		return false, err
	}

	resourceID := subnet.Status.OsokStatus.Ocid
	if resourceID == "" {
		resourceID = subnet.Spec.SubnetId
	}
	if resourceID == "" {
		c.Log.InfoLog("OciSubnet has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciSubnet %s", resourceID))
	done, err := deleteResourceAndWait(
		func() error { return c.DeleteSubnet(ctx, resourceID) },
		func() error {
			_, getErr := c.GetSubnet(ctx, resourceID)
			return getErr
		},
	)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciSubnet")
		return false, err
	}

	return done, nil
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
