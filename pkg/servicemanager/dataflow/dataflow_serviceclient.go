/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dataflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocidataflow "github.com/oracle/oci-go-sdk/v65/dataflow"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
)

// DataFlowClientInterface defines the OCI operations used by DataFlowApplicationServiceManager.
type DataFlowClientInterface interface {
	CreateApplication(ctx context.Context, request ocidataflow.CreateApplicationRequest) (ocidataflow.CreateApplicationResponse, error)
	GetApplication(ctx context.Context, request ocidataflow.GetApplicationRequest) (ocidataflow.GetApplicationResponse, error)
	ListApplications(ctx context.Context, request ocidataflow.ListApplicationsRequest) (ocidataflow.ListApplicationsResponse, error)
	UpdateApplication(ctx context.Context, request ocidataflow.UpdateApplicationRequest) (ocidataflow.UpdateApplicationResponse, error)
	DeleteApplication(ctx context.Context, request ocidataflow.DeleteApplicationRequest) (ocidataflow.DeleteApplicationResponse, error)
}

func getDataFlowClient(provider common.ConfigurationProvider) (ocidataflow.DataFlowClient, error) {
	return ocidataflow.NewDataFlowClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *DataFlowApplicationServiceManager) getOCIClient() (DataFlowClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getDataFlowClient(c.Provider)
}

// CreateDataFlowApplication calls the OCI API to create a new Data Flow Application.
func (c *DataFlowApplicationServiceManager) CreateDataFlowApplication(ctx context.Context, app ociv1beta1.DataFlowApplication) (*ocidataflow.Application, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating DataFlowApplication", "name", app.Spec.DisplayName)

	lang, ok := ocidataflow.GetMappingApplicationLanguageEnum(app.Spec.Language)
	if !ok {
		return nil, fmt.Errorf("invalid language %q: must be PYTHON, SCALA, JAVA, or SQL", app.Spec.Language)
	}

	details := ocidataflow.CreateApplicationDetails{
		CompartmentId: common.String(string(app.Spec.CompartmentId)),
		DisplayName:   common.String(app.Spec.DisplayName),
		Language:      lang,
		DriverShape:   common.String(app.Spec.DriverShape),
		ExecutorShape: common.String(app.Spec.ExecutorShape),
		NumExecutors:  common.Int(app.Spec.NumExecutors),
		SparkVersion:  common.String(app.Spec.SparkVersion),
	}

	if app.Spec.FileUri != "" {
		details.FileUri = common.String(app.Spec.FileUri)
	}
	if app.Spec.ClassName != "" {
		details.ClassName = common.String(app.Spec.ClassName)
	}
	if len(app.Spec.Arguments) > 0 {
		details.Arguments = app.Spec.Arguments
	}
	if app.Spec.Configuration != nil {
		details.Configuration = app.Spec.Configuration
	}
	if app.Spec.Description != "" {
		details.Description = common.String(app.Spec.Description)
	}
	if app.Spec.LogsBucketUri != "" {
		details.LogsBucketUri = common.String(app.Spec.LogsBucketUri)
	}
	if app.Spec.WarehouseBucketUri != "" {
		details.WarehouseBucketUri = common.String(app.Spec.WarehouseBucketUri)
	}
	if app.Spec.ArchiveUri != "" {
		details.ArchiveUri = common.String(app.Spec.ArchiveUri)
	}
	if app.Spec.FreeFormTags != nil {
		details.FreeformTags = app.Spec.FreeFormTags
	}

	req := ocidataflow.CreateApplicationRequest{
		CreateApplicationDetails: details,
	}

	resp, err := client.CreateApplication(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Application, nil
}

// GetDataFlowApplication retrieves a Data Flow Application by OCID.
func (c *DataFlowApplicationServiceManager) GetDataFlowApplication(ctx context.Context, applicationId ociv1beta1.OCID) (*ocidataflow.Application, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := ocidataflow.GetApplicationRequest{
		ApplicationId: common.String(string(applicationId)),
	}

	resp, err := client.GetApplication(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Application, nil
}

// GetDataFlowApplicationByName looks up an existing application by display name.
// Returns nil if no matching ACTIVE application is found.
func (c *DataFlowApplicationServiceManager) GetDataFlowApplicationByName(ctx context.Context, app ociv1beta1.DataFlowApplication) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := ocidataflow.ListApplicationsRequest{
		CompartmentId: common.String(string(app.Spec.CompartmentId)),
		DisplayName:   common.String(app.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListApplications(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing DataFlowApplications")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "INACTIVE" {
			c.Log.DebugLog(fmt.Sprintf("DataFlowApplication %s exists with OCID %s", app.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("DataFlowApplication %s does not exist", app.Spec.DisplayName))
	return nil, nil
}

// UpdateDataFlowApplication updates an existing Data Flow Application.
func (c *DataFlowApplicationServiceManager) UpdateDataFlowApplication(ctx context.Context, app *ociv1beta1.DataFlowApplication) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	updateDetails := ocidataflow.UpdateApplicationDetails{}
	updateNeeded := false

	if app.Spec.DisplayName != "" {
		updateDetails.DisplayName = common.String(app.Spec.DisplayName)
		updateNeeded = true
	}
	if app.Spec.Description != "" {
		updateDetails.Description = common.String(app.Spec.Description)
		updateNeeded = true
	}
	if app.Spec.NumExecutors > 0 {
		updateDetails.NumExecutors = common.Int(app.Spec.NumExecutors)
		updateNeeded = true
	}
	if app.Spec.Configuration != nil {
		updateDetails.Configuration = app.Spec.Configuration
		updateNeeded = true
	}
	if len(app.Spec.Arguments) > 0 {
		updateDetails.Arguments = app.Spec.Arguments
		updateNeeded = true
	}
	if app.Spec.SparkVersion != "" {
		updateDetails.SparkVersion = common.String(app.Spec.SparkVersion)
		updateNeeded = true
	}
	if app.Spec.DriverShape != "" {
		updateDetails.DriverShape = common.String(app.Spec.DriverShape)
		updateNeeded = true
	}
	if app.Spec.ExecutorShape != "" {
		updateDetails.ExecutorShape = common.String(app.Spec.ExecutorShape)
		updateNeeded = true
	}
	if app.Spec.FileUri != "" {
		updateDetails.FileUri = common.String(app.Spec.FileUri)
		updateNeeded = true
	}
	if app.Spec.ClassName != "" {
		updateDetails.ClassName = common.String(app.Spec.ClassName)
		updateNeeded = true
	}
	if app.Spec.ArchiveUri != "" {
		updateDetails.ArchiveUri = common.String(app.Spec.ArchiveUri)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := ocidataflow.UpdateApplicationRequest{
		ApplicationId:            common.String(string(app.Status.OsokStatus.Ocid)),
		UpdateApplicationDetails: updateDetails,
	}

	_, err = client.UpdateApplication(ctx, req)
	return err
}

// DeleteDataFlowApplication deletes the Data Flow Application for the given OCID.
func (c *DataFlowApplicationServiceManager) DeleteDataFlowApplication(ctx context.Context, applicationId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := ocidataflow.DeleteApplicationRequest{
		ApplicationId: common.String(string(applicationId)),
	}

	_, err = client.DeleteApplication(ctx, req)
	if err != nil {
		// Treat 404 as already deleted
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "NotFound") {
			c.Log.InfoLog(fmt.Sprintf("DataFlowApplication %s already deleted", applicationId))
			return nil
		}
		return err
	}
	return nil
}
