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

// GatewayClientInterface defines the OCI operations used by GatewayServiceManager.
type GatewayClientInterface interface {
	CreateGateway(ctx context.Context, request apigateway.CreateGatewayRequest) (apigateway.CreateGatewayResponse, error)
	GetGateway(ctx context.Context, request apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error)
	ListGateways(ctx context.Context, request apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error)
	UpdateGateway(ctx context.Context, request apigateway.UpdateGatewayRequest) (apigateway.UpdateGatewayResponse, error)
	DeleteGateway(ctx context.Context, request apigateway.DeleteGatewayRequest) (apigateway.DeleteGatewayResponse, error)
}

func getGatewayClient(provider common.ConfigurationProvider) (apigateway.GatewayClient, error) {
	return apigateway.NewGatewayClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *GatewayServiceManager) getOCIClient() (GatewayClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getGatewayClient(c.Provider)
}

// CreateGateway calls the OCI API to create a new API Gateway.
func (c *GatewayServiceManager) CreateGateway(ctx context.Context, gw ociv1beta1.ApiGateway) (apigateway.CreateGatewayResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return apigateway.CreateGatewayResponse{}, err
	}

	c.Log.DebugLog("Creating ApiGateway", "displayName", gw.Spec.DisplayName)

	details := apigateway.CreateGatewayDetails{
		CompartmentId: common.String(string(gw.Spec.CompartmentId)),
		EndpointType:  apigateway.GatewayEndpointTypeEnum(gw.Spec.EndpointType),
		SubnetId:      common.String(string(gw.Spec.SubnetId)),
	}

	if gw.Spec.DisplayName != "" {
		details.DisplayName = common.String(gw.Spec.DisplayName)
	}

	if gw.Spec.CertificateId != "" {
		details.CertificateId = common.String(string(gw.Spec.CertificateId))
	}

	if len(gw.Spec.NetworkSecurityGroupIds) > 0 {
		details.NetworkSecurityGroupIds = gw.Spec.NetworkSecurityGroupIds
	}

	if gw.Spec.FreeFormTags != nil {
		details.FreeformTags = gw.Spec.FreeFormTags
	}

	if gw.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&gw.Spec.DefinedTags)
	}

	req := apigateway.CreateGatewayRequest{
		CreateGatewayDetails: details,
	}

	return client.CreateGateway(ctx, req)
}

// GetGateway retrieves an API Gateway by OCID.
func (c *GatewayServiceManager) GetGateway(ctx context.Context, gatewayId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*apigateway.Gateway, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := apigateway.GetGatewayRequest{
		GatewayId: common.String(string(gatewayId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetGateway(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Gateway, nil
}

// GetGatewayOcid looks up an existing gateway by display name and compartment.
func (c *GatewayServiceManager) GetGatewayOcid(ctx context.Context, gw ociv1beta1.ApiGateway) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := apigateway.ListGatewaysRequest{
		CompartmentId: common.String(string(gw.Spec.CompartmentId)),
		DisplayName:   common.String(gw.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListGateways(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing ApiGateways")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("ApiGateway %s exists with OCID %s", gw.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("ApiGateway %s does not exist", gw.Spec.DisplayName))
	return nil, nil
}

// UpdateGateway updates an existing API Gateway.
func (c *GatewayServiceManager) UpdateGateway(ctx context.Context, gw *ociv1beta1.ApiGateway) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	updateDetails := apigateway.UpdateGatewayDetails{}
	updateNeeded := false

	if gw.Spec.DisplayName != "" {
		updateDetails.DisplayName = common.String(gw.Spec.DisplayName)
		updateNeeded = true
	}

	if len(gw.Spec.NetworkSecurityGroupIds) > 0 {
		updateDetails.NetworkSecurityGroupIds = gw.Spec.NetworkSecurityGroupIds
		updateNeeded = true
	}

	if gw.Spec.CertificateId != "" {
		updateDetails.CertificateId = common.String(string(gw.Spec.CertificateId))
		updateNeeded = true
	}

	if gw.Spec.FreeFormTags != nil {
		updateDetails.FreeformTags = gw.Spec.FreeFormTags
		updateNeeded = true
	}

	if gw.Spec.DefinedTags != nil {
		updateDetails.DefinedTags = *util.ConvertToOciDefinedTags(&gw.Spec.DefinedTags)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := apigateway.UpdateGatewayRequest{
		GatewayId:            common.String(string(gw.Status.OsokStatus.Ocid)),
		UpdateGatewayDetails: updateDetails,
	}

	_, err = client.UpdateGateway(ctx, req)
	return err
}

// DeleteGateway deletes the API Gateway for the given OCID.
func (c *GatewayServiceManager) DeleteGateway(ctx context.Context, gatewayId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := apigateway.DeleteGatewayRequest{
		GatewayId: common.String(string(gatewayId)),
	}

	_, err = client.DeleteGateway(ctx, req)
	return err
}

// getGatewayRetryPolicy returns a retry policy that waits while the gateway is CREATING.
func (c *GatewayServiceManager) getGatewayRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(apigateway.GetGatewayResponse); ok {
			return resp.LifecycleState == apigateway.GatewayLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
