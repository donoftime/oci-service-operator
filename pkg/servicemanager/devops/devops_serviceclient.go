/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package devops

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocidevops "github.com/oracle/oci-go-sdk/v65/devops"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// DevopsClientInterface defines the OCI operations used by DevopsProjectServiceManager.
type DevopsClientInterface interface {
	CreateProject(ctx context.Context, request ocidevops.CreateProjectRequest) (ocidevops.CreateProjectResponse, error)
	GetProject(ctx context.Context, request ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error)
	ListProjects(ctx context.Context, request ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error)
	UpdateProject(ctx context.Context, request ocidevops.UpdateProjectRequest) (ocidevops.UpdateProjectResponse, error)
	DeleteProject(ctx context.Context, request ocidevops.DeleteProjectRequest) (ocidevops.DeleteProjectResponse, error)
}

func getDevopsClient(provider common.ConfigurationProvider) (ocidevops.DevopsClient, error) {
	return ocidevops.NewDevopsClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *DevopsProjectServiceManager) getOCIClient() (DevopsClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getDevopsClient(c.Provider)
}

// CreateDevopsProject calls the OCI API to create a new DevOps project.
func (c *DevopsProjectServiceManager) CreateDevopsProject(ctx context.Context, project ociv1beta1.DevopsProject) (ocidevops.CreateProjectResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return ocidevops.CreateProjectResponse{}, err
	}

	c.Log.DebugLog("Creating DevopsProject", "name", project.Spec.Name)

	details := ocidevops.CreateProjectDetails{
		Name:          common.String(project.Spec.Name),
		CompartmentId: common.String(string(project.Spec.CompartmentId)),
		NotificationConfig: &ocidevops.NotificationConfig{
			TopicId: common.String(string(project.Spec.NotificationTopicId)),
		},
		FreeformTags: project.Spec.FreeFormTags,
	}

	if project.Spec.Description != "" {
		details.Description = common.String(project.Spec.Description)
	}

	if project.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&project.Spec.DefinedTags)
	}

	req := ocidevops.CreateProjectRequest{
		CreateProjectDetails: details,
	}

	return client.CreateProject(ctx, req)
}

// GetDevopsProject retrieves a DevOps project by OCID.
func (c *DevopsProjectServiceManager) GetDevopsProject(ctx context.Context, projectId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*ocidevops.Project, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := ocidevops.GetProjectRequest{
		ProjectId: common.String(string(projectId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetProject(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Project, nil
}

// GetDevopsProjectOcid looks up an existing DevOps project by name and returns its OCID if found.
func (c *DevopsProjectServiceManager) GetDevopsProjectOcid(ctx context.Context, project ociv1beta1.DevopsProject) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := ocidevops.ListProjectsRequest{
		CompartmentId: common.String(string(project.Spec.CompartmentId)),
		Name:          common.String(project.Spec.Name),
		Limit:         common.Int(1),
	}

	resp, err := client.ListProjects(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing DevOps projects")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("DevopsProject %s exists with OCID %s", project.Spec.Name, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("DevopsProject %s does not exist", project.Spec.Name))
	return nil, nil
}

// UpdateDevopsProject updates an existing DevOps project.
func (c *DevopsProjectServiceManager) UpdateDevopsProject(ctx context.Context, project *ociv1beta1.DevopsProject) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existing, err := c.GetDevopsProject(ctx, project.Status.OsokStatus.Ocid, nil)
	if err != nil {
		return err
	}

	updateDetails := ocidevops.UpdateProjectDetails{}
	updateNeeded := false

	if project.Spec.Description != "" && (existing.Description == nil || *existing.Description != project.Spec.Description) {
		updateDetails.Description = common.String(project.Spec.Description)
		updateNeeded = true
	}

	if project.Spec.NotificationTopicId != "" {
		newTopicId := string(project.Spec.NotificationTopicId)
		if existing.NotificationConfig == nil || *existing.NotificationConfig.TopicId != newTopicId {
			updateDetails.NotificationConfig = &ocidevops.NotificationConfig{
				TopicId: common.String(newTopicId),
			}
			updateNeeded = true
		}
	}

	if !updateNeeded {
		return nil
	}

	req := ocidevops.UpdateProjectRequest{
		ProjectId:            common.String(string(project.Status.OsokStatus.Ocid)),
		UpdateProjectDetails: updateDetails,
	}

	_, err = client.UpdateProject(ctx, req)
	return err
}

// DeleteDevopsProject deletes the DevOps project for the given OCID.
func (c *DevopsProjectServiceManager) DeleteDevopsProject(ctx context.Context, projectId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := ocidevops.DeleteProjectRequest{
		ProjectId: common.String(string(projectId)),
	}

	_, err = client.DeleteProject(ctx, req)
	return err
}

// getRetryPolicy returns a retry policy that waits while a project is in CREATING state.
func (c *DevopsProjectServiceManager) getRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(ocidevops.GetProjectResponse); ok {
			return resp.LifecycleState == ocidevops.ProjectLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
