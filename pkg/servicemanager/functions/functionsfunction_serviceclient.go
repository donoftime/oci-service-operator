/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import (
	"context"
	"fmt"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (m *FunctionsFunctionServiceManager) getOCIClient() (FunctionsManagementClientInterface, error) {
	if m.ociClient != nil {
		return m.ociClient, nil
	}
	return getFunctionsManagementClient(m.Provider)
}

// CreateFunction calls the OCI API to create a new Functions function.
func (m *FunctionsFunctionServiceManager) CreateFunction(ctx context.Context, fn ociv1beta1.FunctionsFunction) (ocifunctions.CreateFunctionResponse, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return ocifunctions.CreateFunctionResponse{}, err
	}

	m.Log.DebugLog("Creating FunctionsFunction", "name", fn.Spec.DisplayName)

	details := ocifunctions.CreateFunctionDetails{
		ApplicationId: common.String(string(fn.Spec.ApplicationId)),
		DisplayName:   common.String(fn.Spec.DisplayName),
		Image:         common.String(fn.Spec.Image),
		MemoryInMBs:   common.Int64(fn.Spec.MemoryInMBs),
	}

	if fn.Spec.TimeoutInSeconds > 0 {
		details.TimeoutInSeconds = common.Int(fn.Spec.TimeoutInSeconds)
	}
	if len(fn.Spec.Config) > 0 {
		details.Config = fn.Spec.Config
	}
	if fn.Spec.FreeFormTags != nil {
		details.FreeformTags = fn.Spec.FreeFormTags
	}
	if fn.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&fn.Spec.DefinedTags)
	}

	req := ocifunctions.CreateFunctionRequest{
		CreateFunctionDetails: details,
	}

	return client.CreateFunction(ctx, req)
}

// GetFunction retrieves a Functions function by OCID.
func (m *FunctionsFunctionServiceManager) GetFunction(ctx context.Context, fnId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*ocifunctions.Function, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := ocifunctions.GetFunctionRequest{
		FunctionId: common.String(string(fnId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetFunction(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Function, nil
}

// GetFunctionOcid looks up an existing function by display name in the given application and returns its OCID if found.
func (m *FunctionsFunctionServiceManager) GetFunctionOcid(ctx context.Context, fn ociv1beta1.FunctionsFunction) (*ociv1beta1.OCID, error) {
	client, err := m.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := ocifunctions.ListFunctionsRequest{
		ApplicationId: common.String(string(fn.Spec.ApplicationId)),
		DisplayName:   common.String(fn.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListFunctions(ctx, req)
	if err != nil {
		m.Log.ErrorLog(err, "Error listing FunctionsFunctions")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			m.Log.DebugLog(fmt.Sprintf("FunctionsFunction %s exists with OCID %s", fn.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	m.Log.DebugLog(fmt.Sprintf("FunctionsFunction %s does not exist", fn.Spec.DisplayName))
	return nil, nil
}

// UpdateFunction updates an existing Functions function.
func (m *FunctionsFunctionServiceManager) UpdateFunction(ctx context.Context, fn *ociv1beta1.FunctionsFunction) error {
	client, err := m.getOCIClient()
	if err != nil {
		return err
	}

	targetID, err := servicemanager.ResolveResourceID(fn.Status.OsokStatus.Ocid, fn.Spec.FunctionsFunctionId)
	if err != nil {
		return err
	}

	existing, err := m.GetFunction(ctx, targetID, nil)
	if err != nil {
		return err
	}

	if err := validateFunctionUnsupportedChanges(fn, existing); err != nil {
		return err
	}

	updateDetails, updateNeeded := buildFunctionUpdateDetails(fn, existing)
	if !updateNeeded {
		return nil
	}

	req := ocifunctions.UpdateFunctionRequest{
		FunctionId:            common.String(string(targetID)),
		UpdateFunctionDetails: updateDetails,
	}

	_, err = client.UpdateFunction(ctx, req)
	return err
}

func buildFunctionUpdateDetails(fn *ociv1beta1.FunctionsFunction, existing *ocifunctions.Function) (ocifunctions.UpdateFunctionDetails, bool) {
	updateDetails := ocifunctions.UpdateFunctionDetails{}
	updateNeeded := applyFunctionImageUpdate(&updateDetails, fn, existing)
	if applyFunctionMemoryUpdate(&updateDetails, fn, existing) {
		updateNeeded = true
	}
	if applyFunctionTimeoutUpdate(&updateDetails, fn, existing) {
		updateNeeded = true
	}
	if applyFunctionConfigUpdate(&updateDetails, fn, existing) {
		updateNeeded = true
	}
	if applyFunctionFreeformTagUpdate(&updateDetails, fn, existing) {
		updateNeeded = true
	}
	if applyFunctionDefinedTagUpdate(&updateDetails, fn, existing) {
		updateNeeded = true
	}

	return updateDetails, updateNeeded
}

func applyFunctionImageUpdate(updateDetails *ocifunctions.UpdateFunctionDetails, fn *ociv1beta1.FunctionsFunction, existing *ocifunctions.Function) bool {
	if fn.Spec.Image == "" || safeFunctionsString(existing.Image) == fn.Spec.Image {
		return false
	}
	updateDetails.Image = common.String(fn.Spec.Image)
	return true
}

func applyFunctionMemoryUpdate(updateDetails *ocifunctions.UpdateFunctionDetails, fn *ociv1beta1.FunctionsFunction, existing *ocifunctions.Function) bool {
	if fn.Spec.MemoryInMBs <= 0 || (existing.MemoryInMBs != nil && *existing.MemoryInMBs == fn.Spec.MemoryInMBs) {
		return false
	}
	updateDetails.MemoryInMBs = common.Int64(fn.Spec.MemoryInMBs)
	return true
}

func applyFunctionTimeoutUpdate(updateDetails *ocifunctions.UpdateFunctionDetails, fn *ociv1beta1.FunctionsFunction, existing *ocifunctions.Function) bool {
	if fn.Spec.TimeoutInSeconds <= 0 || (existing.TimeoutInSeconds != nil && *existing.TimeoutInSeconds == fn.Spec.TimeoutInSeconds) {
		return false
	}
	updateDetails.TimeoutInSeconds = common.Int(fn.Spec.TimeoutInSeconds)
	return true
}

func applyFunctionConfigUpdate(updateDetails *ocifunctions.UpdateFunctionDetails, fn *ociv1beta1.FunctionsFunction, existing *ocifunctions.Function) bool {
	if len(fn.Spec.Config) == 0 || reflect.DeepEqual(existing.Config, fn.Spec.Config) {
		return false
	}
	updateDetails.Config = fn.Spec.Config
	return true
}

func applyFunctionFreeformTagUpdate(updateDetails *ocifunctions.UpdateFunctionDetails, fn *ociv1beta1.FunctionsFunction, existing *ocifunctions.Function) bool {
	if fn.Spec.FreeFormTags == nil || reflect.DeepEqual(existing.FreeformTags, fn.Spec.FreeFormTags) {
		return false
	}
	updateDetails.FreeformTags = fn.Spec.FreeFormTags
	return true
}

func applyFunctionDefinedTagUpdate(updateDetails *ocifunctions.UpdateFunctionDetails, fn *ociv1beta1.FunctionsFunction, existing *ocifunctions.Function) bool {
	if fn.Spec.DefinedTags == nil {
		return false
	}
	desiredDefinedTags := *util.ConvertToOciDefinedTags(&fn.Spec.DefinedTags)
	if reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
		return false
	}
	updateDetails.DefinedTags = desiredDefinedTags
	return true
}

func validateFunctionUnsupportedChanges(fn *ociv1beta1.FunctionsFunction, existing *ocifunctions.Function) error {
	if err := rejectFunctionsImmutableOCIDChange("applicationId", fn.Spec.ApplicationId, existing.ApplicationId); err != nil {
		return err
	}
	if fn.Spec.DisplayName != "" && safeFunctionsString(existing.DisplayName) != fn.Spec.DisplayName {
		return fmt.Errorf("displayName cannot be updated in place")
	}
	return nil
}

// DeleteFunction deletes the Functions function for the given OCID.
func (m *FunctionsFunctionServiceManager) DeleteFunction(ctx context.Context, fnId ociv1beta1.OCID) error {
	client, err := m.getOCIClient()
	if err != nil {
		return err
	}

	req := ocifunctions.DeleteFunctionRequest{
		FunctionId: common.String(string(fnId)),
	}

	_, err = client.DeleteFunction(ctx, req)
	return err
}
