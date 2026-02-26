/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// DeploymentClientInterface is the subset of apigateway.DeploymentClient methods used by
// DeploymentServiceManager. It allows injection of a mock in tests.
type DeploymentClientInterface interface {
	CreateDeployment(ctx context.Context, request apigateway.CreateDeploymentRequest) (apigateway.CreateDeploymentResponse, error)
	GetDeployment(ctx context.Context, request apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error)
	ListDeployments(ctx context.Context, request apigateway.ListDeploymentsRequest) (apigateway.ListDeploymentsResponse, error)
	UpdateDeployment(ctx context.Context, request apigateway.UpdateDeploymentRequest) (apigateway.UpdateDeploymentResponse, error)
	DeleteDeployment(ctx context.Context, request apigateway.DeleteDeploymentRequest) (apigateway.DeleteDeploymentResponse, error)
}

// getDeploymentClientOrCreate returns the injected client when set; otherwise creates one from the provider.
func (c *DeploymentServiceManager) getDeploymentClientOrCreate() (DeploymentClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return apigateway.NewDeploymentClientWithConfigurationProvider(c.Provider)
}

// buildApiSpecification converts CRD route specs into the OCI SDK ApiSpecification type.
func buildApiSpecification(routes []ociv1beta1.ApiGatewayRoute) *apigateway.ApiSpecification {
	sdkRoutes := make([]apigateway.ApiSpecificationRoute, 0, len(routes))
	for _, r := range routes {
		var backend apigateway.ApiSpecificationRouteBackend
		switch r.Backend.Type {
		case "HTTP_BACKEND":
			backend = apigateway.HttpBackend{
				Url: common.String(r.Backend.Url),
			}
		case "ORACLE_FUNCTIONS_BACKEND":
			backend = apigateway.OracleFunctionBackend{
				FunctionId: common.String(r.Backend.FunctionId),
			}
		case "STOCK_RESPONSE_BACKEND":
			backend = apigateway.StockResponseBackend{
				Status: common.Int(r.Backend.Status),
				Body:   common.String(r.Backend.Body),
			}
		default:
			backend = apigateway.HttpBackend{
				Url: common.String(r.Backend.Url),
			}
		}
		sdkRoute := apigateway.ApiSpecificationRoute{
			Path:    common.String(r.Path),
			Backend: backend,
		}
		if len(r.Methods) > 0 {
			methods := make([]apigateway.ApiSpecificationRouteMethodsEnum, 0, len(r.Methods))
			for _, m := range r.Methods {
				methods = append(methods, apigateway.ApiSpecificationRouteMethodsEnum(m))
			}
			sdkRoute.Methods = methods
		}
		sdkRoutes = append(sdkRoutes, sdkRoute)
	}
	return &apigateway.ApiSpecification{
		Routes: sdkRoutes,
	}
}

// CreateDeployment calls the OCI API to create a new API Gateway Deployment.
func (c *DeploymentServiceManager) CreateDeployment(ctx context.Context, dep ociv1beta1.ApiGatewayDeployment) (apigateway.CreateDeploymentResponse, error) {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return apigateway.CreateDeploymentResponse{}, err
	}

	c.Log.DebugLog("Creating ApiGatewayDeployment", "displayName", dep.Spec.DisplayName)

	details := apigateway.CreateDeploymentDetails{
		GatewayId:     common.String(string(dep.Spec.GatewayId)),
		CompartmentId: common.String(string(dep.Spec.CompartmentId)),
		PathPrefix:    common.String(dep.Spec.PathPrefix),
		Specification: buildApiSpecification(dep.Spec.Routes),
	}

	if dep.Spec.DisplayName != "" {
		details.DisplayName = common.String(dep.Spec.DisplayName)
	}

	if dep.Spec.FreeFormTags != nil {
		details.FreeformTags = dep.Spec.FreeFormTags
	}

	if dep.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&dep.Spec.DefinedTags)
	}

	req := apigateway.CreateDeploymentRequest{
		CreateDeploymentDetails: details,
	}

	return client.CreateDeployment(ctx, req)
}

// GetDeployment retrieves an API Gateway Deployment by OCID.
func (c *DeploymentServiceManager) GetDeployment(ctx context.Context, deploymentId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*apigateway.Deployment, error) {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return nil, err
	}

	req := apigateway.GetDeploymentRequest{
		DeploymentId: common.String(string(deploymentId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetDeployment(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Deployment, nil
}

// GetDeploymentOcid looks up an existing deployment by display name, gateway and compartment.
func (c *DeploymentServiceManager) GetDeploymentOcid(ctx context.Context, dep ociv1beta1.ApiGatewayDeployment) (*ociv1beta1.OCID, error) {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return nil, err
	}

	req := apigateway.ListDeploymentsRequest{
		CompartmentId: common.String(string(dep.Spec.CompartmentId)),
		GatewayId:     common.String(string(dep.Spec.GatewayId)),
		DisplayName:   common.String(dep.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListDeployments(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing ApiGatewayDeployments")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("ApiGatewayDeployment %s exists with OCID %s", dep.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("ApiGatewayDeployment %s does not exist", dep.Spec.DisplayName))
	return nil, nil
}

// UpdateDeployment updates an existing API Gateway Deployment.
func (c *DeploymentServiceManager) UpdateDeployment(ctx context.Context, dep *ociv1beta1.ApiGatewayDeployment) error {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return err
	}

	updateDetails := apigateway.UpdateDeploymentDetails{
		Specification: buildApiSpecification(dep.Spec.Routes),
	}

	if dep.Spec.DisplayName != "" {
		updateDetails.DisplayName = common.String(dep.Spec.DisplayName)
	}

	if dep.Spec.FreeFormTags != nil {
		updateDetails.FreeformTags = dep.Spec.FreeFormTags
	}

	if dep.Spec.DefinedTags != nil {
		updateDetails.DefinedTags = *util.ConvertToOciDefinedTags(&dep.Spec.DefinedTags)
	}

	req := apigateway.UpdateDeploymentRequest{
		DeploymentId:            common.String(string(dep.Status.OsokStatus.Ocid)),
		UpdateDeploymentDetails: updateDetails,
	}

	_, err = client.UpdateDeployment(ctx, req)
	return err
}

// DeleteDeployment deletes the API Gateway Deployment for the given OCID.
func (c *DeploymentServiceManager) DeleteDeployment(ctx context.Context, deploymentId ociv1beta1.OCID) error {
	client, err := c.getDeploymentClientOrCreate()
	if err != nil {
		return err
	}

	req := apigateway.DeleteDeploymentRequest{
		DeploymentId: common.String(string(deploymentId)),
	}

	_, err = client.DeleteDeployment(ctx, req)
	return err
}

// getDeploymentRetryPolicy returns a retry policy that waits while the deployment is CREATING.
func (c *DeploymentServiceManager) getDeploymentRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(apigateway.GetDeploymentResponse); ok {
			return resp.LifecycleState == apigateway.DeploymentLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
