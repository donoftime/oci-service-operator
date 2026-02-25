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

func getFunctionsManagementClient(provider common.ConfigurationProvider) (ocifunctions.FunctionsManagementClient, error) {
	return ocifunctions.NewFunctionsManagementClientWithConfigurationProvider(provider)
}

// CreateApplication calls the OCI API to create a new Functions application.
func (m *FunctionsApplicationServiceManager) CreateApplication(ctx context.Context, app ociv1beta1.FunctionsApplication) (ocifunctions.CreateApplicationResponse, error) {
	client, err := getFunctionsManagementClient(m.Provider)
	if err != nil {
		return ocifunctions.CreateApplicationResponse{}, err
	}

	m.Log.DebugLog("Creating FunctionsApplication", "name", app.Spec.DisplayName)

	details := ocifunctions.CreateApplicationDetails{
		CompartmentId: common.String(string(app.Spec.CompartmentId)),
		DisplayName:   common.String(app.Spec.DisplayName),
		SubnetIds:     app.Spec.SubnetIds,
	}

	if len(app.Spec.Config) > 0 {
		details.Config = app.Spec.Config
	}
	if len(app.Spec.NetworkSecurityGroupIds) > 0 {
		details.NetworkSecurityGroupIds = app.Spec.NetworkSecurityGroupIds
	}
	if app.Spec.SyslogUrl != "" {
		details.SyslogUrl = common.String(app.Spec.SyslogUrl)
	}
	if app.Spec.Shape != "" {
		details.Shape = ocifunctions.CreateApplicationDetailsShapeEnum(app.Spec.Shape)
	}
	if app.Spec.FreeFormTags != nil {
		details.FreeformTags = app.Spec.FreeFormTags
	}
	if app.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&app.Spec.DefinedTags)
	}

	req := ocifunctions.CreateApplicationRequest{
		CreateApplicationDetails: details,
	}

	return client.CreateApplication(ctx, req)
}

// GetApplication retrieves a Functions application by OCID.
func (m *FunctionsApplicationServiceManager) GetApplication(ctx context.Context, appId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*ocifunctions.Application, error) {
	client, err := getFunctionsManagementClient(m.Provider)
	if err != nil {
		return nil, err
	}

	req := ocifunctions.GetApplicationRequest{
		ApplicationId: common.String(string(appId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetApplication(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Application, nil
}

// GetApplicationOcid looks up an existing application by display name and returns its OCID if found.
func (m *FunctionsApplicationServiceManager) GetApplicationOcid(ctx context.Context, app ociv1beta1.FunctionsApplication) (*ociv1beta1.OCID, error) {
	client, err := getFunctionsManagementClient(m.Provider)
	if err != nil {
		return nil, err
	}

	req := ocifunctions.ListApplicationsRequest{
		CompartmentId: common.String(string(app.Spec.CompartmentId)),
		DisplayName:   common.String(app.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListApplications(ctx, req)
	if err != nil {
		m.Log.ErrorLog(err, "Error listing FunctionsApplications")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			m.Log.DebugLog(fmt.Sprintf("FunctionsApplication %s exists with OCID %s", app.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	m.Log.DebugLog(fmt.Sprintf("FunctionsApplication %s does not exist", app.Spec.DisplayName))
	return nil, nil
}

// UpdateApplication updates an existing Functions application.
func (m *FunctionsApplicationServiceManager) UpdateApplication(ctx context.Context, app *ociv1beta1.FunctionsApplication) error {
	client, err := getFunctionsManagementClient(m.Provider)
	if err != nil {
		return err
	}

	updateDetails := ocifunctions.UpdateApplicationDetails{}
	updateNeeded := false

	if len(app.Spec.Config) > 0 {
		updateDetails.Config = app.Spec.Config
		updateNeeded = true
	}
	if len(app.Spec.NetworkSecurityGroupIds) > 0 {
		updateDetails.NetworkSecurityGroupIds = app.Spec.NetworkSecurityGroupIds
		updateNeeded = true
	}
	if app.Spec.SyslogUrl != "" {
		updateDetails.SyslogUrl = common.String(app.Spec.SyslogUrl)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := ocifunctions.UpdateApplicationRequest{
		ApplicationId:            common.String(string(app.Status.OsokStatus.Ocid)),
		UpdateApplicationDetails: updateDetails,
	}

	_, err = client.UpdateApplication(ctx, req)
	return err
}

// DeleteApplication deletes the Functions application for the given OCID.
func (m *FunctionsApplicationServiceManager) DeleteApplication(ctx context.Context, appId ociv1beta1.OCID) error {
	client, err := getFunctionsManagementClient(m.Provider)
	if err != nil {
		return err
	}

	req := ocifunctions.DeleteApplicationRequest{
		ApplicationId: common.String(string(appId)),
	}

	_, err = client.DeleteApplication(ctx, req)
	return err
}
