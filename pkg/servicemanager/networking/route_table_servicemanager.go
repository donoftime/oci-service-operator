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

// Compile-time check that OciRouteTableServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciRouteTableServiceManager{}

// OciRouteTableServiceManager implements OSOKServiceManager for OCI Route Table.
type OciRouteTableServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        VirtualNetworkClientInterface
}

// NewOciRouteTableServiceManager creates a new OciRouteTableServiceManager.
func NewOciRouteTableServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciRouteTableServiceManager {
	return &OciRouteTableServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciRouteTable resource against OCI.
func (c *OciRouteTableServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	rt, err := c.convertRouteTable(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var rtInstance *ocicore.RouteTable

	if strings.TrimSpace(string(rt.Spec.RouteTableId)) == "" {
		rtOcid, err := c.GetRouteTableOcid(ctx, *rt)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if rtOcid == nil {
			rtInstance, err = c.CreateRouteTable(ctx, *rt)
			if err != nil {
				rt.Status.OsokStatus = util.UpdateOSOKStatusCondition(rt.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciRouteTable failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if rtInstance.LifecycleState == ocicore.RouteTableLifecycleStateProvisioning {
				rt.Status.OsokStatus = util.UpdateOSOKStatusCondition(rt.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciRouteTable Provisioning", c.Log)
				rt.Status.OsokStatus.Ocid = ociv1beta1.OCID(*rtInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		} else {
			rtInstance, err = c.GetRouteTable(ctx, *rtOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciRouteTable by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			if rtInstance.LifecycleState == ocicore.RouteTableLifecycleStateProvisioning {
				c.Log.InfoLog(fmt.Sprintf("OciRouteTable %s is still PROVISIONING", safeString(rtInstance.DisplayName)))
				rt.Status.OsokStatus = util.UpdateOSOKStatusCondition(rt.Status.OsokStatus,
					ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciRouteTable Provisioning", c.Log)
				rt.Status.OsokStatus.Ocid = ociv1beta1.OCID(*rtInstance.Id)
				return servicemanager.OSOKResponse{IsSuccessful: false}, nil
			}
		}
	} else {
		rtInstance, err = c.GetRouteTable(ctx, rt.Spec.RouteTableId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciRouteTable")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateRouteTable(ctx, rt); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciRouteTable")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	rt.Status.OsokStatus.Ocid = ociv1beta1.OCID(*rtInstance.Id)
	if rt.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		rt.Status.OsokStatus.CreatedAt = &now
	}

	rt.Status.OsokStatus = util.UpdateOSOKStatusCondition(rt.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("OciRouteTable %s is %s", safeString(rtInstance.DisplayName), rtInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("OciRouteTable %s is %s", safeString(rtInstance.DisplayName), rtInstance.LifecycleState))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the Route Table (called by the finalizer).
func (c *OciRouteTableServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	rt, err := c.convertRouteTable(obj)
	if err != nil {
		return false, err
	}

	if rt.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciRouteTable has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciRouteTable %s", rt.Status.OsokStatus.Ocid))
	if err := c.DeleteRouteTable(ctx, rt.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciRouteTable")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciRouteTableServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convertRouteTable(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciRouteTableServiceManager) convertRouteTable(obj runtime.Object) (*ociv1beta1.OciRouteTable, error) {
	rt, ok := obj.(*ociv1beta1.OciRouteTable)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciRouteTable")
	}
	return rt, nil
}
