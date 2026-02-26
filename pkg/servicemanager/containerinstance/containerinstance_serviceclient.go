/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// ContainerInstanceClientInterface defines the OCI operations used by ContainerInstanceServiceManager.
type ContainerInstanceClientInterface interface {
	CreateContainerInstance(ctx context.Context, request containerinstances.CreateContainerInstanceRequest) (containerinstances.CreateContainerInstanceResponse, error)
	GetContainerInstance(ctx context.Context, request containerinstances.GetContainerInstanceRequest) (containerinstances.GetContainerInstanceResponse, error)
	ListContainerInstances(ctx context.Context, request containerinstances.ListContainerInstancesRequest) (containerinstances.ListContainerInstancesResponse, error)
	UpdateContainerInstance(ctx context.Context, request containerinstances.UpdateContainerInstanceRequest) (containerinstances.UpdateContainerInstanceResponse, error)
	DeleteContainerInstance(ctx context.Context, request containerinstances.DeleteContainerInstanceRequest) (containerinstances.DeleteContainerInstanceResponse, error)
}

func getContainerInstanceClient(provider common.ConfigurationProvider) (containerinstances.ContainerInstanceClient, error) {
	return containerinstances.NewContainerInstanceClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *ContainerInstanceServiceManager) getOCIClient() (ContainerInstanceClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getContainerInstanceClient(c.Provider)
}

// CreateContainerInstance calls the OCI API to create a new container instance.
func (c *ContainerInstanceServiceManager) CreateContainerInstance(ctx context.Context, ci ociv1beta1.ContainerInstance) (containerinstances.CreateContainerInstanceResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return containerinstances.CreateContainerInstanceResponse{}, err
	}

	c.Log.DebugLog("Creating ContainerInstance", "name", ci.Spec.DisplayName)

	containers := make([]containerinstances.CreateContainerDetails, 0, len(ci.Spec.Containers))
	for _, ctr := range ci.Spec.Containers {
		cd := containerinstances.CreateContainerDetails{
			ImageUrl: common.String(ctr.ImageUrl),
		}
		if ctr.DisplayName != nil {
			cd.DisplayName = ctr.DisplayName
		}
		if len(ctr.Command) > 0 {
			cd.Command = ctr.Command
		}
		if len(ctr.Arguments) > 0 {
			cd.Arguments = ctr.Arguments
		}
		if ctr.WorkingDirectory != nil {
			cd.WorkingDirectory = ctr.WorkingDirectory
		}
		if len(ctr.EnvironmentVariables) > 0 {
			cd.EnvironmentVariables = ctr.EnvironmentVariables
		}
		if ctr.ResourceConfig != nil {
			rc := &containerinstances.CreateContainerResourceConfigDetails{}
			if ctr.ResourceConfig.VcpusLimit != nil {
				rc.VcpusLimit = ctr.ResourceConfig.VcpusLimit
			}
			if ctr.ResourceConfig.MemoryLimitInGBs != nil {
				rc.MemoryLimitInGBs = ctr.ResourceConfig.MemoryLimitInGBs
			}
			cd.ResourceConfig = rc
		}
		containers = append(containers, cd)
	}

	vnics := make([]containerinstances.CreateContainerVnicDetails, 0, len(ci.Spec.Vnics))
	for _, vnic := range ci.Spec.Vnics {
		vd := containerinstances.CreateContainerVnicDetails{
			SubnetId: common.String(string(vnic.SubnetId)),
		}
		if vnic.DisplayName != nil {
			vd.DisplayName = vnic.DisplayName
		}
		if len(vnic.NsgIds) > 0 {
			nsgIds := make([]string, len(vnic.NsgIds))
			for i, id := range vnic.NsgIds {
				nsgIds[i] = string(id)
			}
			vd.NsgIds = nsgIds
		}
		vnics = append(vnics, vd)
	}

	shapeCfg := &containerinstances.CreateContainerInstanceShapeConfigDetails{
		Ocpus:       common.Float32(ci.Spec.ShapeConfig.Ocpus),
		MemoryInGBs: common.Float32(ci.Spec.ShapeConfig.MemoryInGBs),
	}

	details := containerinstances.CreateContainerInstanceDetails{
		CompartmentId:      common.String(string(ci.Spec.CompartmentId)),
		AvailabilityDomain: common.String(ci.Spec.AvailabilityDomain),
		Shape:              common.String(ci.Spec.Shape),
		ShapeConfig:        shapeCfg,
		Containers:         containers,
		Vnics:              vnics,
	}

	if ci.Spec.DisplayName != nil {
		details.DisplayName = ci.Spec.DisplayName
	}
	if ci.Spec.FaultDomain != nil {
		details.FaultDomain = ci.Spec.FaultDomain
	}
	if ci.Spec.GracefulShutdownTimeoutInSeconds != nil {
		details.GracefulShutdownTimeoutInSeconds = ci.Spec.GracefulShutdownTimeoutInSeconds
	}
	if ci.Spec.ContainerRestartPolicy != nil {
		details.ContainerRestartPolicy = containerinstances.ContainerInstanceContainerRestartPolicyEnum(*ci.Spec.ContainerRestartPolicy)
	}
	if ci.Spec.FreeFormTags != nil {
		details.FreeformTags = ci.Spec.FreeFormTags
	}
	if ci.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&ci.Spec.DefinedTags)
	}

	req := containerinstances.CreateContainerInstanceRequest{
		CreateContainerInstanceDetails: details,
	}

	return client.CreateContainerInstance(ctx, req)
}

// GetContainerInstance retrieves a container instance by OCID.
func (c *ContainerInstanceServiceManager) GetContainerInstance(ctx context.Context, ciId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*containerinstances.ContainerInstance, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := containerinstances.GetContainerInstanceRequest{
		ContainerInstanceId: common.String(string(ciId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetContainerInstance(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.ContainerInstance, nil
}

// GetContainerInstanceOcid looks up an existing container instance by display name.
func (c *ContainerInstanceServiceManager) GetContainerInstanceOcid(ctx context.Context, ci ociv1beta1.ContainerInstance) (*ociv1beta1.OCID, error) {
	if ci.Spec.DisplayName == nil {
		return nil, nil
	}

	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := containerinstances.ListContainerInstancesRequest{
		CompartmentId:      common.String(string(ci.Spec.CompartmentId)),
		DisplayName:        ci.Spec.DisplayName,
		AvailabilityDomain: common.String(ci.Spec.AvailabilityDomain),
		Limit:              common.Int(1),
	}

	resp, err := client.ListContainerInstances(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing container instances")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" || state == "INACTIVE" {
			c.Log.DebugLog(fmt.Sprintf("ContainerInstance %s exists with OCID %s", *ci.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("ContainerInstance %s does not exist", *ci.Spec.DisplayName))
	return nil, nil
}

// UpdateContainerInstance updates an existing container instance's display name.
func (c *ContainerInstanceServiceManager) UpdateContainerInstance(ctx context.Context, ci *ociv1beta1.ContainerInstance) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetContainerInstance(ctx, ci.Status.OsokStatus.Ocid, nil)
	if err != nil {
		return err
	}

	updateDetails := containerinstances.UpdateContainerInstanceDetails{}
	updateNeeded := false

	if ci.Spec.DisplayName != nil && (existing.DisplayName == nil || *existing.DisplayName != *ci.Spec.DisplayName) {
		updateDetails.DisplayName = ci.Spec.DisplayName
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := containerinstances.UpdateContainerInstanceRequest{
		ContainerInstanceId:             common.String(string(ci.Status.OsokStatus.Ocid)),
		UpdateContainerInstanceDetails:  updateDetails,
	}

	_, err = client.UpdateContainerInstance(ctx, req)
	return err
}

// DeleteContainerInstance deletes the container instance for the given OCID.
func (c *ContainerInstanceServiceManager) DeleteContainerInstance(ctx context.Context, ciId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := containerinstances.DeleteContainerInstanceRequest{
		ContainerInstanceId: common.String(string(ciId)),
	}

	_, err = client.DeleteContainerInstance(ctx, req)
	return err
}

// getRetryPolicy returns a retry policy that waits while a container instance is in CREATING state.
func (c *ContainerInstanceServiceManager) getRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(containerinstances.GetContainerInstanceResponse); ok {
			return resp.LifecycleState == containerinstances.ContainerInstanceLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
