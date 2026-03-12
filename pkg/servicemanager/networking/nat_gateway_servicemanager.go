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

// Compile-time check that OciNatGatewayServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciNatGatewayServiceManager{}

// OciNatGatewayServiceManager implements OSOKServiceManager for OCI NAT Gateway.
type OciNatGatewayServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciNatGatewayServiceManager creates a new OciNatGatewayServiceManager.
func NewOciNatGatewayServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciNatGatewayServiceManager {
	return &OciNatGatewayServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciNatGateway resource against OCI.
func (c *OciNatGatewayServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	nat, err := c.convertNAT(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var natInstance *ocicore.NatGateway

	if strings.TrimSpace(string(nat.Spec.NatGatewayId)) == "" {
		// No explicit ID — look up by display name or create.
		natOcid, err := c.GetNatGatewayOcid(ctx, *nat)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if natOcid == nil {
			natInstance, err = c.CreateNatGateway(ctx, *nat)
			if err != nil {
				nat.Status.OsokStatus = util.UpdateOSOKStatusCondition(nat.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciNatGateway failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			natInstance, err = c.GetNatGateway(ctx, *natOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciNatGateway by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
	} else {
		// Bind to an existing NAT Gateway by ID.
		natInstance, err = c.GetNatGateway(ctx, nat.Spec.NatGatewayId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciNatGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		nat.Status.OsokStatus.Ocid = nat.Spec.NatGatewayId
		if err = c.UpdateNatGateway(ctx, nat); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciNatGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return reconcileLifecycleStatus(&nat.Status.OsokStatus, "OciNatGateway", safeString(natInstance.DisplayName),
		string(natInstance.LifecycleState), ociv1beta1.OCID(*natInstance.Id), c.Log), nil
}

// Delete handles deletion of the NAT Gateway (called by the finalizer).
func (c *OciNatGatewayServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	nat, err := c.convertNAT(obj)
	if err != nil {
		return false, err
	}

	resourceID := nat.Status.OsokStatus.Ocid
	if resourceID == "" {
		resourceID = nat.Spec.NatGatewayId
	}
	if resourceID == "" {
		c.Log.InfoLog("OciNatGateway has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciNatGateway %s", resourceID))
	done, err := deleteResourceAndWait(
		func() error { return c.DeleteNatGateway(ctx, resourceID) },
		func() error {
			_, getErr := c.GetNatGateway(ctx, resourceID)
			return getErr
		},
	)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciNatGateway")
		return false, err
	}

	return done, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciNatGatewayServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertNAT(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciNatGatewayServiceManager) convertNAT(obj runtime.Object) (*ociv1beta1.OciNatGateway, error) {
	nat, ok := obj.(*ociv1beta1.OciNatGateway)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciNatGateway")
	}
	return nat, nil
}
