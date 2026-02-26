/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compute

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// ComputeInstanceClientInterface defines the OCI operations used by ComputeInstanceServiceManager.
type ComputeInstanceClientInterface interface {
	LaunchInstance(ctx context.Context, request core.LaunchInstanceRequest) (core.LaunchInstanceResponse, error)
	GetInstance(ctx context.Context, request core.GetInstanceRequest) (core.GetInstanceResponse, error)
	ListInstances(ctx context.Context, request core.ListInstancesRequest) (core.ListInstancesResponse, error)
	UpdateInstance(ctx context.Context, request core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error)
	TerminateInstance(ctx context.Context, request core.TerminateInstanceRequest) (core.TerminateInstanceResponse, error)
}

func getComputeClient(provider common.ConfigurationProvider) (core.ComputeClient, error) {
	return core.NewComputeClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *ComputeInstanceServiceManager) getOCIClient() (ComputeInstanceClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getComputeClient(c.Provider)
}

// LaunchInstance calls the OCI API to launch a new compute instance.
func (c *ComputeInstanceServiceManager) LaunchInstance(ctx context.Context, ci ociv1beta1.ComputeInstance) (core.LaunchInstanceResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return core.LaunchInstanceResponse{}, err
	}

	c.Log.DebugLog("Launching ComputeInstance", "name", ci.Spec.DisplayName)

	details := core.LaunchInstanceDetails{
		CompartmentId:      common.String(string(ci.Spec.CompartmentId)),
		AvailabilityDomain: common.String(ci.Spec.AvailabilityDomain),
		Shape:              common.String(ci.Spec.Shape),
		ImageId:            common.String(string(ci.Spec.ImageId)),
		SubnetId:           common.String(string(ci.Spec.SubnetId)),
	}

	if ci.Spec.DisplayName != nil {
		details.DisplayName = ci.Spec.DisplayName
	}
	if ci.Spec.ShapeConfig != nil {
		details.ShapeConfig = &core.LaunchInstanceShapeConfigDetails{
			Ocpus:       common.Float32(ci.Spec.ShapeConfig.Ocpus),
			MemoryInGBs: common.Float32(ci.Spec.ShapeConfig.MemoryInGBs),
		}
	}
	if ci.Spec.FreeFormTags != nil {
		details.FreeformTags = ci.Spec.FreeFormTags
	}
	if ci.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&ci.Spec.DefinedTags)
	}

	// Disable legacy IMDS v1 endpoints — required for tenancies that enforce IMDS v2.
	details.InstanceOptions = &core.InstanceOptions{
		AreLegacyImdsEndpointsDisabled: common.Bool(true),
	}

	req := core.LaunchInstanceRequest{
		LaunchInstanceDetails: details,
	}

	return client.LaunchInstance(ctx, req)
}

// GetInstance retrieves a compute instance by OCID.
func (c *ComputeInstanceServiceManager) GetInstance(ctx context.Context, instanceId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*core.Instance, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := core.GetInstanceRequest{
		InstanceId: common.String(string(instanceId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetInstance(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Instance, nil
}

// GetInstanceOcid looks up an existing compute instance by display name.
func (c *ComputeInstanceServiceManager) GetInstanceOcid(ctx context.Context, ci ociv1beta1.ComputeInstance) (*ociv1beta1.OCID, error) {
	if ci.Spec.DisplayName == nil {
		return nil, nil
	}

	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := core.ListInstancesRequest{
		CompartmentId: common.String(string(ci.Spec.CompartmentId)),
		DisplayName:   ci.Spec.DisplayName,
		Limit:         common.Int(1),
	}

	resp, err := client.ListInstances(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing compute instances")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "RUNNING" || state == "PROVISIONING" || state == "STARTING" || state == "STOPPING" || state == "STOPPED" {
			c.Log.DebugLog(fmt.Sprintf("ComputeInstance %s exists with OCID %s", *ci.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("ComputeInstance %s does not exist", *ci.Spec.DisplayName))
	return nil, nil
}

// UpdateInstance updates an existing compute instance's display name.
func (c *ComputeInstanceServiceManager) UpdateInstance(ctx context.Context, ci *ociv1beta1.ComputeInstance) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetInstance(ctx, ci.Status.OsokStatus.Ocid, nil)
	if err != nil {
		return err
	}

	updateDetails := core.UpdateInstanceDetails{}
	updateNeeded := false

	if ci.Spec.DisplayName != nil && (existing.DisplayName == nil || *existing.DisplayName != *ci.Spec.DisplayName) {
		updateDetails.DisplayName = ci.Spec.DisplayName
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := core.UpdateInstanceRequest{
		InstanceId:            common.String(string(ci.Status.OsokStatus.Ocid)),
		UpdateInstanceDetails: updateDetails,
	}

	_, err = client.UpdateInstance(ctx, req)
	return err
}

// TerminateInstance terminates the compute instance for the given OCID.
func (c *ComputeInstanceServiceManager) TerminateInstance(ctx context.Context, instanceId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := core.TerminateInstanceRequest{
		InstanceId: common.String(string(instanceId)),
	}

	_, err = client.TerminateInstance(ctx, req)
	return err
}

// getRetryPolicy returns a retry policy that waits while a compute instance is in PROVISIONING state.
func (c *ComputeInstanceServiceManager) getRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(core.GetInstanceResponse); ok {
			return resp.LifecycleState == core.InstanceLifecycleStateProvisioning
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
