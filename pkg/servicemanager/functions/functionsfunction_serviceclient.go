/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// CreateFunction calls the OCI API to create a new Functions function.
func (m *FunctionsFunctionServiceManager) CreateFunction(ctx context.Context, fn ociv1beta1.FunctionsFunction) (ocifunctions.CreateFunctionResponse, error) {
	client, err := getFunctionsManagementClient(m.Provider)
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
	client, err := getFunctionsManagementClient(m.Provider)
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
	client, err := getFunctionsManagementClient(m.Provider)
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
	client, err := getFunctionsManagementClient(m.Provider)
	if err != nil {
		return err
	}

	updateDetails := ocifunctions.UpdateFunctionDetails{}
	updateNeeded := false

	if fn.Spec.Image != "" {
		updateDetails.Image = common.String(fn.Spec.Image)
		updateNeeded = true
	}
	if fn.Spec.MemoryInMBs > 0 {
		updateDetails.MemoryInMBs = common.Int64(fn.Spec.MemoryInMBs)
		updateNeeded = true
	}
	if fn.Spec.TimeoutInSeconds > 0 {
		updateDetails.TimeoutInSeconds = common.Int(fn.Spec.TimeoutInSeconds)
		updateNeeded = true
	}
	if len(fn.Spec.Config) > 0 {
		updateDetails.Config = fn.Spec.Config
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := ocifunctions.UpdateFunctionRequest{
		FunctionId:            common.String(string(fn.Status.OsokStatus.Ocid)),
		UpdateFunctionDetails: updateDetails,
	}

	_, err = client.UpdateFunction(ctx, req)
	return err
}

// DeleteFunction deletes the Functions function for the given OCID.
func (m *FunctionsFunctionServiceManager) DeleteFunction(ctx context.Context, fnId ociv1beta1.OCID) error {
	client, err := getFunctionsManagementClient(m.Provider)
	if err != nil {
		return err
	}

	req := ocifunctions.DeleteFunctionRequest{
		FunctionId: common.String(string(fnId)),
	}

	_, err = client.DeleteFunction(ctx, req)
	return err
}
