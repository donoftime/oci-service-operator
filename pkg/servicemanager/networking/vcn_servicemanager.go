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

	var vcnInstance *ocicore.Vcn

	if strings.TrimSpace(string(vcn.Spec.VcnId)) == "" {
		// No explicit ID — look up by display name or create.
		vcnOcid, err := c.GetVcnOcid(ctx, *vcn)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if vcnOcid == nil {
			vcnInstance, err = c.CreateVcn(ctx, *vcn)
			if err != nil {
				vcn.Status.OsokStatus = util.UpdateOSOKStatusCondition(vcn.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciVcn failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if vcnInstance.LifecycleState == ocicore.VcnLifecycleStateProvisioning {
				vcn.Status.OsokStatus = util.UpdateOSOKStatusCondition(vcn.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciVcn Provisioning", c.Log)
				vcn.Status.OsokStatus.Ocid = ociv1beta1.OCID(*vcnInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		} else {
			vcnInstance, err = c.GetVcn(ctx, *vcnOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciVcn by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if vcnInstance.LifecycleState == ocicore.VcnLifecycleStateProvisioning {
				c.Log.InfoLog(fmt.Sprintf("OciVcn %s is still PROVISIONING", safeString(vcnInstance.DisplayName)))
				vcn.Status.OsokStatus = util.UpdateOSOKStatusCondition(vcn.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciVcn Provisioning", c.Log)
				vcn.Status.OsokStatus.Ocid = ociv1beta1.OCID(*vcnInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		}
	} else {
		// Bind to an existing VCN by ID.
		vcnInstance, err = c.GetVcn(ctx, vcn.Spec.VcnId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciVcn")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateVcn(ctx, vcn); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciVcn")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	vcn.Status.OsokStatus.Ocid = ociv1beta1.OCID(*vcnInstance.Id)
	if vcn.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		vcn.Status.OsokStatus.CreatedAt = &now
	}

	vcn.Status.OsokStatus = util.UpdateOSOKStatusCondition(vcn.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("OciVcn %s is %s", safeString(vcnInstance.DisplayName), vcnInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("OciVcn %s is %s", safeString(vcnInstance.DisplayName), vcnInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the VCN (called by the finalizer).
func (c *OciVcnServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	vcn, err := c.convertVcn(obj)
	if err != nil {
		return false, err
	}

	if vcn.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciVcn has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciVcn %s", vcn.Status.OsokStatus.Ocid))
	if err := c.DeleteVcn(ctx, vcn.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciVcn")
		return false, err
	}

	return true, nil
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
