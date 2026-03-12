/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// GatewayClientInterface is the subset of apigateway.GatewayClient methods used by
// GatewayServiceManager. It allows injection of a mock in tests.
type GatewayClientInterface interface {
	CreateGateway(ctx context.Context, request apigateway.CreateGatewayRequest) (apigateway.CreateGatewayResponse, error)
	GetGateway(ctx context.Context, request apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error)
	ListGateways(ctx context.Context, request apigateway.ListGatewaysRequest) (apigateway.ListGatewaysResponse, error)
	ChangeGatewayCompartment(ctx context.Context, request apigateway.ChangeGatewayCompartmentRequest) (apigateway.ChangeGatewayCompartmentResponse, error)
	UpdateGateway(ctx context.Context, request apigateway.UpdateGatewayRequest) (apigateway.UpdateGatewayResponse, error)
	DeleteGateway(ctx context.Context, request apigateway.DeleteGatewayRequest) (apigateway.DeleteGatewayResponse, error)
}

// getGatewayClientOrCreate returns the injected client when set; otherwise creates one from the provider.
func (c *GatewayServiceManager) getGatewayClientOrCreate() (GatewayClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return apigateway.NewGatewayClientWithConfigurationProvider(c.Provider)
}

// CreateGateway calls the OCI API to create a new API Gateway.
func (c *GatewayServiceManager) CreateGateway(ctx context.Context, gw ociv1beta1.ApiGateway) (apigateway.CreateGatewayResponse, error) {
	client, err := c.getGatewayClientOrCreate()
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
	client, err := c.getGatewayClientOrCreate()
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
	client, err := c.getGatewayClientOrCreate()
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
	client, err := c.getGatewayClientOrCreate()
	if err != nil {
		return err
	}

	targetID, err := servicemanager.ResolveResourceID(gw.Status.OsokStatus.Ocid, gw.Spec.ApiGatewayId)
	if err != nil {
		return err
	}

	existing, err := c.GetGateway(ctx, targetID, nil)
	if err != nil {
		return err
	}

	if err := validateGatewayUnsupportedChanges(gw, existing); err != nil {
		return err
	}

	if gw.Spec.CompartmentId != "" &&
		(existing.CompartmentId == nil || *existing.CompartmentId != string(gw.Spec.CompartmentId)) {
		if _, err = client.ChangeGatewayCompartment(ctx, apigateway.ChangeGatewayCompartmentRequest{
			GatewayId: common.String(string(targetID)),
			ChangeGatewayCompartmentDetails: apigateway.ChangeGatewayCompartmentDetails{
				CompartmentId: common.String(string(gw.Spec.CompartmentId)),
			},
		}); err != nil {
			return err
		}
	}

	updateDetails, updateNeeded := buildGatewayUpdateDetails(gw, existing)

	if !updateNeeded {
		return nil
	}

	req := apigateway.UpdateGatewayRequest{
		GatewayId:            common.String(string(targetID)),
		UpdateGatewayDetails: updateDetails,
	}

	_, err = client.UpdateGateway(ctx, req)
	return err
}

func buildGatewayUpdateDetails(gw *ociv1beta1.ApiGateway, existing *apigateway.Gateway) (apigateway.UpdateGatewayDetails, bool) {
	updateDetails := apigateway.UpdateGatewayDetails{}
	updateNeeded := applyGatewayDisplayNameUpdate(&updateDetails, gw, existing)
	if applyGatewayNetworkSecurityGroupUpdate(&updateDetails, gw, existing) {
		updateNeeded = true
	}
	if applyGatewayCertificateUpdate(&updateDetails, gw, existing) {
		updateNeeded = true
	}
	if applyGatewayFreeformTagUpdate(&updateDetails, gw, existing) {
		updateNeeded = true
	}
	if applyGatewayDefinedTagUpdate(&updateDetails, gw, existing) {
		updateNeeded = true
	}

	return updateDetails, updateNeeded
}

func applyGatewayDisplayNameUpdate(updateDetails *apigateway.UpdateGatewayDetails, gw *ociv1beta1.ApiGateway, existing *apigateway.Gateway) bool {
	if gw.Spec.DisplayName == "" || safeGatewayString(existing.DisplayName) == gw.Spec.DisplayName {
		return false
	}
	updateDetails.DisplayName = common.String(gw.Spec.DisplayName)
	return true
}

func applyGatewayNetworkSecurityGroupUpdate(updateDetails *apigateway.UpdateGatewayDetails, gw *ociv1beta1.ApiGateway, existing *apigateway.Gateway) bool {
	if len(gw.Spec.NetworkSecurityGroupIds) == 0 || reflect.DeepEqual(existing.NetworkSecurityGroupIds, gw.Spec.NetworkSecurityGroupIds) {
		return false
	}
	updateDetails.NetworkSecurityGroupIds = gw.Spec.NetworkSecurityGroupIds
	return true
}

func applyGatewayCertificateUpdate(updateDetails *apigateway.UpdateGatewayDetails, gw *ociv1beta1.ApiGateway, existing *apigateway.Gateway) bool {
	if gw.Spec.CertificateId == "" || safeGatewayString(existing.CertificateId) == string(gw.Spec.CertificateId) {
		return false
	}
	updateDetails.CertificateId = common.String(string(gw.Spec.CertificateId))
	return true
}

func applyGatewayFreeformTagUpdate(updateDetails *apigateway.UpdateGatewayDetails, gw *ociv1beta1.ApiGateway, existing *apigateway.Gateway) bool {
	if gw.Spec.FreeFormTags == nil || reflect.DeepEqual(existing.FreeformTags, gw.Spec.FreeFormTags) {
		return false
	}
	updateDetails.FreeformTags = gw.Spec.FreeFormTags
	return true
}

func applyGatewayDefinedTagUpdate(updateDetails *apigateway.UpdateGatewayDetails, gw *ociv1beta1.ApiGateway, existing *apigateway.Gateway) bool {
	if gw.Spec.DefinedTags == nil {
		return false
	}
	desiredDefinedTags := *util.ConvertToOciDefinedTags(&gw.Spec.DefinedTags)
	if reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
		return false
	}
	updateDetails.DefinedTags = desiredDefinedTags
	return true
}

func validateGatewayUnsupportedChanges(gw *ociv1beta1.ApiGateway, existing *apigateway.Gateway) error {
	if gw.Spec.EndpointType != "" && existing.EndpointType != "" && string(existing.EndpointType) != gw.Spec.EndpointType {
		return fmt.Errorf("endpointType cannot be updated in place")
	}
	if gw.Spec.SubnetId != "" && safeGatewayString(existing.SubnetId) != "" && safeGatewayString(existing.SubnetId) != string(gw.Spec.SubnetId) {
		return fmt.Errorf("subnetId cannot be updated in place")
	}
	return nil
}

// DeleteGateway deletes the API Gateway for the given OCID.
func (c *GatewayServiceManager) DeleteGateway(ctx context.Context, gatewayId ociv1beta1.OCID) error {
	client, err := c.getGatewayClientOrCreate()
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
