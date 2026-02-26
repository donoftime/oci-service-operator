/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package postgresql

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/psql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// PostgresClientInterface defines the OCI operations used by PostgresDbSystemServiceManager.
type PostgresClientInterface interface {
	CreateDbSystem(ctx context.Context, request psql.CreateDbSystemRequest) (psql.CreateDbSystemResponse, error)
	GetDbSystem(ctx context.Context, request psql.GetDbSystemRequest) (psql.GetDbSystemResponse, error)
	ListDbSystems(ctx context.Context, request psql.ListDbSystemsRequest) (psql.ListDbSystemsResponse, error)
	UpdateDbSystem(ctx context.Context, request psql.UpdateDbSystemRequest) (psql.UpdateDbSystemResponse, error)
	DeleteDbSystem(ctx context.Context, request psql.DeleteDbSystemRequest) (psql.DeleteDbSystemResponse, error)
}

func getPostgresClient(provider common.ConfigurationProvider) (psql.PostgresqlClient, error) {
	return psql.NewPostgresqlClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *PostgresDbSystemServiceManager) getOCIClient() (PostgresClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getPostgresClient(c.Provider)
}

// CreatePostgresDbSystem calls the OCI API to create a new PostgreSQL DB system.
func (c *PostgresDbSystemServiceManager) CreatePostgresDbSystem(ctx context.Context, dbSystem ociv1beta1.PostgresDbSystem) (psql.CreateDbSystemResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return psql.CreateDbSystemResponse{}, err
	}

	c.Log.DebugLog("Creating PostgresDbSystem", "name", dbSystem.Spec.DisplayName)

	storageDetails := buildStorageDetails(dbSystem.Spec.StorageType)

	details := psql.CreateDbSystemDetails{
		DisplayName:   common.String(dbSystem.Spec.DisplayName),
		CompartmentId: common.String(string(dbSystem.Spec.CompartmentId)),
		DbVersion:     common.String(dbSystem.Spec.DbVersion),
		Shape:         common.String(dbSystem.Spec.Shape),
		NetworkDetails: &psql.NetworkDetails{
			SubnetId: common.String(string(dbSystem.Spec.SubnetId)),
		},
		StorageDetails: storageDetails,
	}

	if dbSystem.Spec.Description != "" {
		details.Description = common.String(dbSystem.Spec.Description)
	}
	if dbSystem.Spec.InstanceCount > 0 {
		details.InstanceCount = common.Int(dbSystem.Spec.InstanceCount)
	}
	if dbSystem.Spec.InstanceOcpuCount > 0 {
		details.InstanceOcpuCount = common.Int(dbSystem.Spec.InstanceOcpuCount)
	}
	if dbSystem.Spec.InstanceMemoryInGBs > 0 {
		details.InstanceMemorySizeInGBs = common.Int(dbSystem.Spec.InstanceMemoryInGBs)
	}
	if dbSystem.Spec.FreeFormTags != nil {
		details.FreeformTags = dbSystem.Spec.FreeFormTags
	}
	if dbSystem.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&dbSystem.Spec.DefinedTags)
	}

	req := psql.CreateDbSystemRequest{
		CreateDbSystemDetails: details,
	}

	return client.CreateDbSystem(ctx, req)
}

// GetPostgresDbSystem retrieves a PostgreSQL DB system by OCID.
func (c *PostgresDbSystemServiceManager) GetPostgresDbSystem(ctx context.Context, dbSystemId ociv1beta1.OCID) (*psql.DbSystem, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := psql.GetDbSystemRequest{
		DbSystemId: common.String(string(dbSystemId)),
	}

	resp, err := client.GetDbSystem(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.DbSystem, nil
}

// GetPostgresDbSystemByName looks up an existing DB system by display name and returns its OCID if found.
func (c *PostgresDbSystemServiceManager) GetPostgresDbSystemByName(ctx context.Context, dbSystem ociv1beta1.PostgresDbSystem) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := psql.ListDbSystemsRequest{
		CompartmentId: common.String(string(dbSystem.Spec.CompartmentId)),
		DisplayName:   common.String(dbSystem.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListDbSystems(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing PostgreSQL DB systems")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("PostgresDbSystem %s exists with OCID %s", dbSystem.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("PostgresDbSystem %s does not exist", dbSystem.Spec.DisplayName))
	return nil, nil
}

// UpdatePostgresDbSystem updates an existing PostgreSQL DB system.
func (c *PostgresDbSystemServiceManager) UpdatePostgresDbSystem(ctx context.Context, dbSystem *ociv1beta1.PostgresDbSystem) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	updateDetails := psql.UpdateDbSystemDetails{}
	updateNeeded := false

	existing, err := c.GetPostgresDbSystem(ctx, dbSystem.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	if dbSystem.Spec.DisplayName != "" && *existing.DisplayName != dbSystem.Spec.DisplayName {
		updateDetails.DisplayName = common.String(dbSystem.Spec.DisplayName)
		updateNeeded = true
	}

	if dbSystem.Spec.Description != "" && (existing.Description == nil || *existing.Description != dbSystem.Spec.Description) {
		updateDetails.Description = common.String(dbSystem.Spec.Description)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := psql.UpdateDbSystemRequest{
		DbSystemId:            common.String(string(dbSystem.Status.OsokStatus.Ocid)),
		UpdateDbSystemDetails: updateDetails,
	}

	_, err = client.UpdateDbSystem(ctx, req)
	return err
}

// DeletePostgresDbSystem deletes the PostgreSQL DB system for the given OCID.
func (c *PostgresDbSystemServiceManager) DeletePostgresDbSystem(ctx context.Context, dbSystemId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := psql.DeleteDbSystemRequest{
		DbSystemId: common.String(string(dbSystemId)),
	}

	_, err = client.DeleteDbSystem(ctx, req)
	return err
}

// buildStorageDetails constructs the appropriate StorageDetails based on the storage type.
func buildStorageDetails(storageType string) psql.StorageDetails {
	isRegionallyDurable := true
	return psql.OciOptimizedStorageDetails{
		IsRegionallyDurable: common.Bool(isRegionallyDurable),
	}
}
