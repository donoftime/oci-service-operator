/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

func getContainerInstanceClient(provider common.ConfigurationProvider) (containerinstances.ContainerInstanceClient, error) {
	return containerinstances.NewContainerInstanceClientWithConfigurationProvider(provider)
}

// CreateContainerInstance calls the OCI API to create a new container instance.
func (c *ContainerInstanceServiceManager) CreateContainerInstance(ctx context.Context, ci ociv1beta1.ContainerInstance) (containerinstances.CreateContainerInstanceResponse, error) {
	client, err := getContainerInstanceClient(c.Provider)
	if err != nil {
		return containerinstances.CreateContainerInstanceResponse{}, err
	}

	c.Log.DebugLog("Creating ContainerInstance", "name", ci.Spec.DisplayName)

	containers := make([]containerinstances.CreateContainerDetails, 0, len(ci.Spec.Containers))
	for _, cspec := range ci.Spec.Containers {
		container := containerinstances.CreateContainerDetails{
			ImageUrl: common.String(cspec.ImageUrl),
		}
		if cspec.DisplayName != "" {
			container.DisplayName = common.String(cspec.DisplayName)
		}
		if len(cspec.Command) > 0 {
			container.Command = cspec.Command
		}
		if len(cspec.Arguments) > 0 {
			container.Arguments = cspec.Arguments
		}
		if cspec.WorkingDirectory != "" {
			container.WorkingDirectory = common.String(cspec.WorkingDirectory)
		}
		if len(cspec.EnvironmentVariables) > 0 {
			container.EnvironmentVariables = cspec.EnvironmentVariables
		}
		containers = append(containers, container)
	}

	vnics := make([]containerinstances.CreateContainerVnicDetails, 0, len(ci.Spec.Vnics))
	for _, vspec := range ci.Spec.Vnics {
		vnic := containerinstances.CreateContainerVnicDetails{
			SubnetId: common.String(string(vspec.SubnetId)),
		}
		if vspec.DisplayName != "" {
			vnic.DisplayName = common.String(vspec.DisplayName)
		}
		if vspec.IsPublicIpAssigned != nil {
			vnic.IsPublicIpAssigned = vspec.IsPublicIpAssigned
		}
		if len(vspec.NsgIds) > 0 {
			vnic.NsgIds = vspec.NsgIds
		}
		vnics = append(vnics, vnic)
	}

	shapeConfig := containerinstances.CreateContainerInstanceShapeConfigDetails{
		Ocpus: common.Float32(ci.Spec.Ocpus),
	}
	if ci.Spec.MemoryInGBs != nil {
		shapeConfig.MemoryInGBs = ci.Spec.MemoryInGBs
	}

	details := containerinstances.CreateContainerInstanceDetails{
		CompartmentId:      common.String(string(ci.Spec.CompartmentId)),
		AvailabilityDomain: common.String(ci.Spec.AvailabilityDomain),
		Shape:              common.String(ci.Spec.Shape),
		ShapeConfig:        &shapeConfig,
		Containers:         containers,
		Vnics:              vnics,
	}

	if ci.Spec.DisplayName != "" {
		details.DisplayName = common.String(ci.Spec.DisplayName)
	}
	if ci.Spec.FaultDomain != "" {
		details.FaultDomain = common.String(ci.Spec.FaultDomain)
	}
	if ci.Spec.GracefulShutdownTimeoutInSeconds != nil {
		details.GracefulShutdownTimeoutInSeconds = ci.Spec.GracefulShutdownTimeoutInSeconds
	}
	if ci.Spec.ContainerRestartPolicy != "" {
		details.ContainerRestartPolicy = containerinstances.ContainerInstanceContainerRestartPolicyEnum(ci.Spec.ContainerRestartPolicy)
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
	client, err := getContainerInstanceClient(c.Provider)
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

// GetContainerInstanceOcid looks up an existing container instance by display name and returns its OCID if found.
func (c *ContainerInstanceServiceManager) GetContainerInstanceOcid(ctx context.Context, ci ociv1beta1.ContainerInstance) (*ociv1beta1.OCID, error) {
	if ci.Spec.DisplayName == "" {
		return nil, nil
	}

	client, err := getContainerInstanceClient(c.Provider)
	if err != nil {
		return nil, err
	}

	req := containerinstances.ListContainerInstancesRequest{
		CompartmentId: common.String(string(ci.Spec.CompartmentId)),
		DisplayName:   common.String(ci.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListContainerInstances(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing ContainerInstances")
		return nil, err
	}

	for _, item := range resp.Items {
		state := item.LifecycleState
		if state == containerinstances.ContainerInstanceLifecycleStateActive ||
			state == containerinstances.ContainerInstanceLifecycleStateCreating ||
			state == containerinstances.ContainerInstanceLifecycleStateInactive {
			c.Log.DebugLog(fmt.Sprintf("ContainerInstance %s exists with OCID %s", ci.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("ContainerInstance %s does not exist", ci.Spec.DisplayName))
	return nil, nil
}

// UpdateContainerInstance updates mutable fields (displayName, tags) of an existing container instance.
func (c *ContainerInstanceServiceManager) UpdateContainerInstance(ctx context.Context, ci *ociv1beta1.ContainerInstance) error {
	client, err := getContainerInstanceClient(c.Provider)
	if err != nil {
		return err
	}

	updateDetails := containerinstances.UpdateContainerInstanceDetails{}
	updateNeeded := false

	if ci.Spec.DisplayName != "" {
		updateDetails.DisplayName = common.String(ci.Spec.DisplayName)
		updateNeeded = true
	}
	if ci.Spec.FreeFormTags != nil {
		updateDetails.FreeformTags = ci.Spec.FreeFormTags
		updateNeeded = true
	}
	if ci.Spec.DefinedTags != nil {
		updateDetails.DefinedTags = *util.ConvertToOciDefinedTags(&ci.Spec.DefinedTags)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := containerinstances.UpdateContainerInstanceRequest{
		ContainerInstanceId:                common.String(string(ci.Status.OsokStatus.Ocid)),
		UpdateContainerInstanceDetails:     updateDetails,
	}

	_, err = client.UpdateContainerInstance(ctx, req)
	return err
}

// DeleteContainerInstance deletes the container instance for the given OCID.
func (c *ContainerInstanceServiceManager) DeleteContainerInstance(ctx context.Context, ciId ociv1beta1.OCID) error {
	client, err := getContainerInstanceClient(c.Provider)
	if err != nil {
		return err
	}

	req := containerinstances.DeleteContainerInstanceRequest{
		ContainerInstanceId: common.String(string(ciId)),
	}

	_, err = client.DeleteContainerInstance(ctx, req)
	return err
}
