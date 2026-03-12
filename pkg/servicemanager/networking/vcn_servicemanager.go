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

// Compile-time check that OciVcnServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciVcnServiceManager{}

// OciVcnServiceManager implements OSOKServiceManager for OCI VCN.
type OciVcnServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciVcnServiceManager creates a new OciVcnServiceManager.
func NewOciVcnServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciVcnServiceManager {
	return &OciVcnServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciVcn resource against OCI.
func (c *OciVcnServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	vcn, err := c.convertVcn(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	vcnInstance, err := reconcileNetworkingResource(networkingCreateOrUpdateOps[ocicore.Vcn]{
		SpecID: vcn.Spec.VcnId,
		Status: &vcn.Status.OsokStatus,
		Get: func(id ociv1beta1.OCID) (*ocicore.Vcn, error) {
			return c.GetVcn(ctx, id)
		},
		Update: func() error {
			return c.UpdateVcn(ctx, vcn)
		},
		Lookup: func() (*ociv1beta1.OCID, error) {
			return c.GetVcnOcid(ctx, *vcn)
		},
		Create: func() (*ocicore.Vcn, error) {
			return c.CreateVcn(ctx, *vcn)
		},
		OnCreateError: func(err error) {
			vcn.Status.OsokStatus = util.UpdateOSOKStatusCondition(vcn.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
			c.Log.ErrorLog(err, "Create OciVcn failed")
		},
		Log:            c.Log,
		GetExistingMsg: "Error while getting existing OciVcn",
		GetStatusMsg:   "Error while getting existing OciVcn from status OCID",
		GetByOCIDMsg:   "Error while getting OciVcn by OCID",
		UpdateMsg:      "Error while updating OciVcn",
	})
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return reconcileLifecycleStatus(&vcn.Status.OsokStatus, "OciVcn", safeString(vcnInstance.DisplayName),
		string(vcnInstance.LifecycleState), ociv1beta1.OCID(*vcnInstance.Id), c.Log), nil
}

// Delete handles deletion of the VCN (called by the finalizer).
func (c *OciVcnServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	vcn, err := c.convertVcn(obj)
	if err != nil {
		return false, err
	}

	resourceID := vcn.Status.OsokStatus.Ocid
	if resourceID == "" {
		resourceID = vcn.Spec.VcnId
	}
	if resourceID == "" {
		c.Log.InfoLog("OciVcn has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciVcn %s", resourceID))
	done, err := deleteResourceAndWait(
		func() error { return c.DeleteVcn(ctx, resourceID) },
		func() error {
			_, getErr := c.GetVcn(ctx, resourceID)
			return getErr
		},
	)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciVcn")
		return false, err
	}

	return done, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciVcnServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertVcn(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciVcnServiceManager) convertVcn(obj runtime.Object) (*ociv1beta1.OciVcn, error) {
	vcn, ok := obj.(*ociv1beta1.OciVcn)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciVcn")
	}
	return vcn, nil
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
