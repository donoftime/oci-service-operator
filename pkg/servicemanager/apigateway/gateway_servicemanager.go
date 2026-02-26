/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that GatewayServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &GatewayServiceManager{}

// GatewayServiceManager implements OSOKServiceManager for OCI API Gateway.
type GatewayServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        GatewayClientInterface
}

// NewGatewayServiceManager creates a new GatewayServiceManager.
func NewGatewayServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *GatewayServiceManager {
	return &GatewayServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the ApiGateway resource against OCI.
func (c *GatewayServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	gw, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var gwInstance *apigateway.Gateway

	if strings.TrimSpace(string(gw.Spec.ApiGatewayId)) == "" {
		// No ID provided — look up by display name or create
		gwOcid, err := c.GetGatewayOcid(ctx, *gw)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if gwOcid == nil {
			// Create a new API Gateway
			resp, err := c.CreateGateway(ctx, *gw)
			if err != nil {
				gw.Status.OsokStatus = util.UpdateOSOKStatusCondition(gw.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Create ApiGateway failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				gw.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
				c.Log.ErrorLog(err, "Create ApiGateway bad request")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			c.Log.InfoLog(fmt.Sprintf("ApiGateway %s is Provisioning", gw.Spec.DisplayName))
			gw.Status.OsokStatus = util.UpdateOSOKStatusCondition(gw.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "ApiGateway Provisioning", c.Log)
			gw.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)

			retryPolicy := c.getGatewayRetryPolicy(30)
			gwInstance, err = c.GetGateway(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting ApiGateway after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting existing ApiGateway %s", *gwOcid))
			gwInstance, err = c.GetGateway(ctx, *gwOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting ApiGateway by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			if err = c.UpdateGateway(ctx, gw); err != nil {
				c.Log.ErrorLog(err, "Error while updating ApiGateway")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
	} else {
		// Bind to an existing gateway by ID
		gwInstance, err = c.GetGateway(ctx, gw.Spec.ApiGatewayId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing ApiGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		if err = c.UpdateGateway(ctx, gw); err != nil {
			c.Log.ErrorLog(err, "Error while updating ApiGateway")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	gw.Status.OsokStatus.Ocid = ociv1beta1.OCID(*gwInstance.Id)
	if gw.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		gw.Status.OsokStatus.CreatedAt = &now
	}

	if gwInstance.LifecycleState == apigateway.GatewayLifecycleStateCreating {
		gw.Status.OsokStatus = util.UpdateOSOKStatusCondition(gw.Status.OsokStatus,
			ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGateway %s is still Creating", *gwInstance.DisplayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("ApiGateway %s is still Creating, requeueing", *gwInstance.DisplayName))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}, nil
	}

	if gwInstance.LifecycleState == apigateway.GatewayLifecycleStateFailed {
		gw.Status.OsokStatus = util.UpdateOSOKStatusCondition(gw.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ApiGateway %s creation Failed", *gwInstance.DisplayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("ApiGateway %s creation Failed", *gwInstance.DisplayName))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	gw.Status.OsokStatus = util.UpdateOSOKStatusCondition(gw.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("ApiGateway %s is %s", *gwInstance.DisplayName, gwInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("ApiGateway %s is Active", *gwInstance.DisplayName))

	if _, err := c.addToSecret(ctx, gw.Namespace, gw.Name, *gwInstance); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			c.Log.InfoLog("ApiGateway secret creation failed")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the API Gateway (called by the finalizer).
func (c *GatewayServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	gw, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if gw.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("ApiGateway has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting ApiGateway %s", gw.Status.OsokStatus.Ocid))
	if err := c.DeleteGateway(ctx, gw.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting ApiGateway")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *GatewayServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *GatewayServiceManager) convert(obj runtime.Object) (*ociv1beta1.ApiGateway, error) {
	gw, ok := obj.(*ociv1beta1.ApiGateway)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for ApiGateway")
	}
	return gw, nil
}
