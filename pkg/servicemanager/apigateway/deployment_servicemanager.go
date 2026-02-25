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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that DeploymentServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &DeploymentServiceManager{}

// DeploymentServiceManager implements OSOKServiceManager for OCI API Gateway Deployments.
type DeploymentServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
}

// NewDeploymentServiceManager creates a new DeploymentServiceManager.
func NewDeploymentServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *DeploymentServiceManager {
	return &DeploymentServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the ApiGatewayDeployment resource against OCI.
func (c *DeploymentServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	dep, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var depInstance *apigateway.Deployment

	if strings.TrimSpace(string(dep.Spec.DeploymentId)) == "" {
		// No ID provided — look up by display name or create
		depOcid, err := c.GetDeploymentOcid(ctx, *dep)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if depOcid == nil {
			// Create a new deployment
			resp, err := c.CreateDeployment(ctx, *dep)
			if err != nil {
				dep.Status.OsokStatus = util.UpdateOSOKStatusCondition(dep.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Create ApiGatewayDeployment failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				dep.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
				c.Log.ErrorLog(err, "Create ApiGatewayDeployment bad request")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			c.Log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is Provisioning", dep.Spec.DisplayName))
			dep.Status.OsokStatus = util.UpdateOSOKStatusCondition(dep.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "ApiGatewayDeployment Provisioning", c.Log)
			dep.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)

			retryPolicy := c.getDeploymentRetryPolicy(30)
			depInstance, err = c.GetDeployment(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting ApiGatewayDeployment after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting existing ApiGatewayDeployment %s", *depOcid))
			depInstance, err = c.GetDeployment(ctx, *depOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting ApiGatewayDeployment by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			dep.Status.OsokStatus.Ocid = *depOcid
			if err = c.UpdateDeployment(ctx, dep); err != nil {
				c.Log.ErrorLog(err, "Error while updating ApiGatewayDeployment")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
	} else {
		// Bind to an existing deployment by ID
		depInstance, err = c.GetDeployment(ctx, dep.Spec.DeploymentId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing ApiGatewayDeployment")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		dep.Status.OsokStatus.Ocid = dep.Spec.DeploymentId
		if err = c.UpdateDeployment(ctx, dep); err != nil {
			c.Log.ErrorLog(err, "Error while updating ApiGatewayDeployment")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	dep.Status.OsokStatus.Ocid = ociv1beta1.OCID(*depInstance.Id)
	if dep.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		dep.Status.OsokStatus.CreatedAt = &now
	}

	if depInstance.LifecycleState == apigateway.DeploymentLifecycleStateFailed {
		dep.Status.OsokStatus = util.UpdateOSOKStatusCondition(dep.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ApiGatewayDeployment %s creation Failed", *depInstance.DisplayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s creation Failed", *depInstance.DisplayName))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	dep.Status.OsokStatus = util.UpdateOSOKStatusCondition(dep.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("ApiGatewayDeployment %s is %s", *depInstance.DisplayName, depInstance.LifecycleState), c.Log)
	c.Log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is Active", *depInstance.DisplayName))

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the API Gateway Deployment (called by the finalizer).
func (c *DeploymentServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	dep, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if dep.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("ApiGatewayDeployment has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting ApiGatewayDeployment %s", dep.Status.OsokStatus.Ocid))
	if err := c.DeleteDeployment(ctx, dep.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting ApiGatewayDeployment")
		return false, err
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *DeploymentServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *DeploymentServiceManager) convert(obj runtime.Object) (*ociv1beta1.ApiGatewayDeployment, error) {
	dep, ok := obj.(*ociv1beta1.ApiGatewayDeployment)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for ApiGatewayDeployment")
	}
	return dep, nil
}
