/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"context"
	"fmt"
	"reflect"
	"sort"
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
	ChangeContainerInstanceCompartment(ctx context.Context, request containerinstances.ChangeContainerInstanceCompartmentRequest) (containerinstances.ChangeContainerInstanceCompartmentResponse, error)
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

	return client.CreateContainerInstance(ctx, buildCreateContainerInstanceRequest(ci))
}

func buildCreateContainerInstanceRequest(ci ociv1beta1.ContainerInstance) containerinstances.CreateContainerInstanceRequest {
	return containerinstances.CreateContainerInstanceRequest{
		CreateContainerInstanceDetails: buildCreateContainerInstanceDetails(ci),
	}
}

func buildCreateContainerInstanceDetails(ci ociv1beta1.ContainerInstance) containerinstances.CreateContainerInstanceDetails {
	details := containerinstances.CreateContainerInstanceDetails{
		CompartmentId:      common.String(string(ci.Spec.CompartmentId)),
		AvailabilityDomain: common.String(ci.Spec.AvailabilityDomain),
		Shape:              common.String(ci.Spec.Shape),
		ShapeConfig:        buildShapeConfig(ci.Spec.ShapeConfig),
		Containers:         buildContainers(ci.Spec.Containers),
		Vnics:              buildContainerVnics(ci.Spec.Vnics),
	}

	applyOptionalCreateDetails(&details, ci)
	return details
}

func buildShapeConfig(shapeConfig ociv1beta1.ContainerInstanceShapeConfig) *containerinstances.CreateContainerInstanceShapeConfigDetails {
	return &containerinstances.CreateContainerInstanceShapeConfigDetails{
		Ocpus:       common.Float32(shapeConfig.Ocpus),
		MemoryInGBs: common.Float32(shapeConfig.MemoryInGBs),
	}
}

func buildContainers(containers []ociv1beta1.ContainerDetails) []containerinstances.CreateContainerDetails {
	result := make([]containerinstances.CreateContainerDetails, 0, len(containers))
	for _, ctr := range containers {
		result = append(result, buildContainer(ctr))
	}
	return result
}

func buildContainer(ctr ociv1beta1.ContainerDetails) containerinstances.CreateContainerDetails {
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
		cd.ResourceConfig = buildContainerResourceConfig(ctr.ResourceConfig)
	}
	if len(ctr.VolumeMounts) > 0 {
		cd.VolumeMounts = buildVolumeMounts(ctr.VolumeMounts)
	}

	return cd
}

func buildContainerResourceConfig(resourceConfig *ociv1beta1.ContainerResourceConfig) *containerinstances.CreateContainerResourceConfigDetails {
	rc := &containerinstances.CreateContainerResourceConfigDetails{}
	if resourceConfig.VcpusLimit != nil {
		rc.VcpusLimit = resourceConfig.VcpusLimit
	}
	if resourceConfig.MemoryLimitInGBs != nil {
		rc.MemoryLimitInGBs = resourceConfig.MemoryLimitInGBs
	}
	return rc
}

func buildVolumeMounts(volumeMounts []ociv1beta1.ContainerVolumeMount) []containerinstances.CreateVolumeMountDetails {
	result := make([]containerinstances.CreateVolumeMountDetails, 0, len(volumeMounts))
	for _, vm := range volumeMounts {
		result = append(result, buildVolumeMount(vm))
	}
	return result
}

func buildVolumeMount(volumeMount ociv1beta1.ContainerVolumeMount) containerinstances.CreateVolumeMountDetails {
	vmd := containerinstances.CreateVolumeMountDetails{
		MountPath:  common.String(volumeMount.MountPath),
		VolumeName: common.String(volumeMount.VolumeName),
	}
	if volumeMount.SubPath != nil {
		vmd.SubPath = volumeMount.SubPath
	}
	if volumeMount.IsReadOnly != nil {
		vmd.IsReadOnly = volumeMount.IsReadOnly
	}
	return vmd
}

func buildContainerVnics(vnics []ociv1beta1.ContainerVnicDetails) []containerinstances.CreateContainerVnicDetails {
	result := make([]containerinstances.CreateContainerVnicDetails, 0, len(vnics))
	for _, vnic := range vnics {
		result = append(result, buildContainerVnic(vnic))
	}
	return result
}

func buildContainerVnic(vnic ociv1beta1.ContainerVnicDetails) containerinstances.CreateContainerVnicDetails {
	vd := containerinstances.CreateContainerVnicDetails{
		SubnetId: common.String(string(vnic.SubnetId)),
	}
	if vnic.DisplayName != nil {
		vd.DisplayName = vnic.DisplayName
	}
	if len(vnic.NsgIds) > 0 {
		vd.NsgIds = convertOCIDsToStrings(vnic.NsgIds)
	}
	return vd
}

func convertOCIDsToStrings(ids []ociv1beta1.OCID) []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = string(id)
	}
	return result
}

func applyOptionalCreateDetails(details *containerinstances.CreateContainerInstanceDetails, ci ociv1beta1.ContainerInstance) {
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
	if len(ci.Spec.ImagePullSecrets) > 0 {
		details.ImagePullSecrets = buildImagePullSecrets(ci.Spec.ImagePullSecrets)
	}
}

func buildImagePullSecrets(secrets []ociv1beta1.ContainerImagePullSecret) []containerinstances.CreateImagePullSecretDetails {
	result := make([]containerinstances.CreateImagePullSecretDetails, 0, len(secrets))
	for _, secret := range secrets {
		result = append(result, containerinstances.CreateBasicImagePullSecretDetails{
			RegistryEndpoint: common.String(secret.RegistryEndpoint),
			Username:         common.String(secret.Username),
			Password:         common.String(secret.Password),
		})
	}
	return result
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

	targetID, err := resolveContainerInstanceID(ci.Status.OsokStatus.Ocid, ci.Spec.ContainerInstanceId)
	if err != nil {
		return err
	}

	existing, err := c.GetContainerInstance(ctx, targetID, nil)
	if err != nil {
		return err
	}

	if err := validateContainerInstanceUnsupportedChanges(ci, existing); err != nil {
		return err
	}

	if err := moveContainerInstanceCompartmentIfNeeded(ctx, client, ci, existing, targetID); err != nil {
		return err
	}

	updateDetails, updateNeeded := buildContainerInstanceUpdateDetails(ci, existing)
	if !updateNeeded {
		return nil
	}

	req := containerinstances.UpdateContainerInstanceRequest{
		ContainerInstanceId:            common.String(string(targetID)),
		UpdateContainerInstanceDetails: updateDetails,
	}

	_, err = client.UpdateContainerInstance(ctx, req)
	return err
}

func moveContainerInstanceCompartmentIfNeeded(ctx context.Context, client ContainerInstanceClientInterface,
	ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance, targetID ociv1beta1.OCID) error {
	if ci.Spec.CompartmentId == "" || (existing.CompartmentId != nil && *existing.CompartmentId == string(ci.Spec.CompartmentId)) {
		return nil
	}

	_, err := client.ChangeContainerInstanceCompartment(ctx, containerinstances.ChangeContainerInstanceCompartmentRequest{
		ContainerInstanceId: common.String(string(targetID)),
		ChangeContainerInstanceCompartmentDetails: containerinstances.ChangeContainerInstanceCompartmentDetails{
			CompartmentId: common.String(string(ci.Spec.CompartmentId)),
		},
	})
	return err
}

func buildContainerInstanceUpdateDetails(ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) (containerinstances.UpdateContainerInstanceDetails, bool) {
	updateDetails := containerinstances.UpdateContainerInstanceDetails{}
	updateNeeded := applyContainerInstanceDisplayNameUpdate(&updateDetails, ci, existing)
	if applyContainerInstanceFreeformTagUpdate(&updateDetails, ci, existing) {
		updateNeeded = true
	}
	if applyContainerInstanceDefinedTagUpdate(&updateDetails, ci, existing) {
		updateNeeded = true
	}

	return updateDetails, updateNeeded
}

func applyContainerInstanceDisplayNameUpdate(updateDetails *containerinstances.UpdateContainerInstanceDetails,
	ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) bool {
	if ci.Spec.DisplayName == nil || (existing.DisplayName != nil && *existing.DisplayName == *ci.Spec.DisplayName) {
		return false
	}
	updateDetails.DisplayName = ci.Spec.DisplayName
	return true
}

func applyContainerInstanceFreeformTagUpdate(updateDetails *containerinstances.UpdateContainerInstanceDetails,
	ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) bool {
	if ci.Spec.FreeFormTags == nil || reflect.DeepEqual(existing.FreeformTags, ci.Spec.FreeFormTags) {
		return false
	}
	updateDetails.FreeformTags = ci.Spec.FreeFormTags
	return true
}

func applyContainerInstanceDefinedTagUpdate(updateDetails *containerinstances.UpdateContainerInstanceDetails,
	ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) bool {
	if ci.Spec.DefinedTags == nil {
		return false
	}
	desiredDefinedTags := *util.ConvertToOciDefinedTags(&ci.Spec.DefinedTags)
	if reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
		return false
	}
	updateDetails.DefinedTags = desiredDefinedTags
	return true
}

func validateContainerInstanceUnsupportedChanges(ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) error {
	if err := validateContainerAvailabilityDomain(ci, existing); err != nil {
		return err
	}
	if err := validateContainerShape(ci, existing); err != nil {
		return err
	}
	if err := validateContainerFaultDomain(ci, existing); err != nil {
		return err
	}
	if err := validateContainerShutdownTimeout(ci, existing); err != nil {
		return err
	}
	return validateContainerRestartPolicy(ci, existing)
}

func validateContainerAvailabilityDomain(ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) error {
	if ci.Spec.AvailabilityDomain != "" && existing.AvailabilityDomain != nil && *existing.AvailabilityDomain != ci.Spec.AvailabilityDomain {
		return fmt.Errorf("availabilityDomain cannot be updated in place")
	}
	return nil
}

func validateContainerShape(ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) error {
	if ci.Spec.Shape != "" && existing.Shape != nil && *existing.Shape != ci.Spec.Shape {
		return fmt.Errorf("shape cannot be updated in place")
	}
	return nil
}

func validateContainerFaultDomain(ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) error {
	if ci.Spec.FaultDomain != nil && existing.FaultDomain != nil && *existing.FaultDomain != *ci.Spec.FaultDomain {
		return fmt.Errorf("faultDomain cannot be updated in place")
	}
	return nil
}

func validateContainerShutdownTimeout(ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) error {
	if ci.Spec.GracefulShutdownTimeoutInSeconds != nil &&
		existing.GracefulShutdownTimeoutInSeconds != nil &&
		*existing.GracefulShutdownTimeoutInSeconds != *ci.Spec.GracefulShutdownTimeoutInSeconds {
		return fmt.Errorf("gracefulShutdownTimeoutInSeconds cannot be updated in place")
	}
	return nil
}

func validateContainerRestartPolicy(ci *ociv1beta1.ContainerInstance, existing *containerinstances.ContainerInstance) error {
	if ci.Spec.ContainerRestartPolicy != nil &&
		existing.ContainerRestartPolicy != "" &&
		string(existing.ContainerRestartPolicy) != *ci.Spec.ContainerRestartPolicy {
		return fmt.Errorf("containerRestartPolicy cannot be updated in place")
	}
	return nil
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

// ListAllContainerInstances returns all non-DELETED container instances matching
// the CR's DisplayName, CompartmentId, and AvailabilityDomain, sorted by
// TimeCreated ascending (oldest first). Returns an empty slice if DisplayName is nil.
func (c *ContainerInstanceServiceManager) ListAllContainerInstances(
	ctx context.Context,
	ci ociv1beta1.ContainerInstance,
) ([]containerinstances.ContainerInstanceSummary, error) {
	if ci.Spec.DisplayName == nil {
		return []containerinstances.ContainerInstanceSummary{}, nil
	}

	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := containerinstances.ListContainerInstancesRequest{
		CompartmentId:      common.String(string(ci.Spec.CompartmentId)),
		DisplayName:        ci.Spec.DisplayName,
		AvailabilityDomain: common.String(ci.Spec.AvailabilityDomain),
	}

	resp, err := client.ListContainerInstances(ctx, req)
	if err != nil {
		return nil, err
	}

	var result []containerinstances.ContainerInstanceSummary
	for _, item := range resp.Items {
		if item.LifecycleState != containerinstances.ContainerInstanceLifecycleStateDeleted {
			result = append(result, item)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].TimeCreated == nil {
			return true
		}
		if result[j].TimeCreated == nil {
			return false
		}
		return result[i].TimeCreated.Before(result[j].TimeCreated.Time)
	})

	return result, nil
}

// GarbageCollect deletes old container instances beyond the configured MaxInstances limit.
// The oldest instances (by TimeCreated) are deleted first. GC failures are logged but
// do not prevent further deletions. Returns the first error encountered, if any.
func (c *ContainerInstanceServiceManager) GarbageCollect(
	ctx context.Context,
	ci ociv1beta1.ContainerInstance,
) error {
	maxInstances := int32(3)
	if ci.Spec.GCPolicy != nil && ci.Spec.GCPolicy.MaxInstances > 0 {
		maxInstances = ci.Spec.GCPolicy.MaxInstances
	}

	instances, err := c.ListAllContainerInstances(ctx, ci)
	if err != nil {
		return err
	}

	if int32(len(instances)) <= maxInstances {
		return nil
	}

	toDelete := instances[:len(instances)-int(maxInstances)]
	var firstErr error
	for _, inst := range toDelete {
		created := ""
		if inst.TimeCreated != nil {
			created = inst.TimeCreated.String()
		}
		c.Log.InfoLog(fmt.Sprintf("GC: deleting old ContainerInstance %s (created %s)", *inst.Id, created))
		if delErr := c.DeleteContainerInstance(ctx, ociv1beta1.OCID(*inst.Id)); delErr != nil {
			c.Log.ErrorLog(delErr, fmt.Sprintf("GC: failed to delete ContainerInstance %s", *inst.Id))
			if firstErr == nil {
				firstErr = delErr
			}
		}
	}
	return firstErr
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
