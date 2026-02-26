/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// NosqlClientInterface defines the OCI operations used by NoSQLDatabaseServiceManager.
type NosqlClientInterface interface {
	CreateTable(ctx context.Context, request nosql.CreateTableRequest) (nosql.CreateTableResponse, error)
	GetTable(ctx context.Context, request nosql.GetTableRequest) (nosql.GetTableResponse, error)
	ListTables(ctx context.Context, request nosql.ListTablesRequest) (nosql.ListTablesResponse, error)
	UpdateTable(ctx context.Context, request nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error)
	DeleteTable(ctx context.Context, request nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error)
}

func getNosqlClient(provider common.ConfigurationProvider) (nosql.NosqlClient, error) {
	return nosql.NewNosqlClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *NoSQLDatabaseServiceManager) getOCIClient() (NosqlClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getNosqlClient(c.Provider)
}

// CreateTable calls the OCI API to create a new NoSQL table.
func (c *NoSQLDatabaseServiceManager) CreateTable(ctx context.Context, db ociv1beta1.NoSQLDatabase) (nosql.CreateTableResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nosql.CreateTableResponse{}, err
	}

	c.Log.DebugLog("Creating NoSQL table", "name", db.Spec.Name)

	details := nosql.CreateTableDetails{
		Name:          common.String(db.Spec.Name),
		CompartmentId: common.String(string(db.Spec.CompartmentId)),
		DdlStatement:  common.String(db.Spec.DdlStatement),
		FreeformTags:  db.Spec.FreeFormTags,
	}

	if db.Spec.TableLimits != nil {
		details.TableLimits = &nosql.TableLimits{
			MaxReadUnits:    common.Int(db.Spec.TableLimits.MaxReadUnits),
			MaxWriteUnits:   common.Int(db.Spec.TableLimits.MaxWriteUnits),
			MaxStorageInGBs: common.Int(db.Spec.TableLimits.MaxStorageInGBs),
		}
	}

	if db.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&db.Spec.DefinedTags)
	}

	req := nosql.CreateTableRequest{
		CreateTableDetails: details,
	}

	return client.CreateTable(ctx, req)
}

// GetTable retrieves a NoSQL table by OCID.
func (c *NoSQLDatabaseServiceManager) GetTable(ctx context.Context, tableId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*nosql.Table, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := nosql.GetTableRequest{
		TableNameOrId: common.String(string(tableId)),
		CompartmentId: nil,
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetTable(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Table, nil
}

// GetTableOcid looks up an existing NoSQL table by name and returns its OCID if found.
func (c *NoSQLDatabaseServiceManager) GetTableOcid(ctx context.Context, db ociv1beta1.NoSQLDatabase) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := nosql.ListTablesRequest{
		CompartmentId: common.String(string(db.Spec.CompartmentId)),
		Name:          common.String(db.Spec.Name),
		Limit:         common.Int(1),
	}

	resp, err := client.ListTables(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing NoSQL tables")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("NoSQL table %s exists with OCID %s", db.Spec.Name, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("NoSQL table %s does not exist", db.Spec.Name))
	return nil, nil
}

// UpdateTable updates the DDL statement and limits for an existing NoSQL table.
func (c *NoSQLDatabaseServiceManager) UpdateTable(ctx context.Context, db *ociv1beta1.NoSQLDatabase) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	updateDetails := nosql.UpdateTableDetails{}
	updateNeeded := false

	if db.Spec.DdlStatement != "" {
		updateDetails.DdlStatement = common.String(db.Spec.DdlStatement)
		updateNeeded = true
	}

	if db.Spec.TableLimits != nil {
		updateDetails.TableLimits = &nosql.TableLimits{
			MaxReadUnits:    common.Int(db.Spec.TableLimits.MaxReadUnits),
			MaxWriteUnits:   common.Int(db.Spec.TableLimits.MaxWriteUnits),
			MaxStorageInGBs: common.Int(db.Spec.TableLimits.MaxStorageInGBs),
		}
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := nosql.UpdateTableRequest{
		TableNameOrId:     common.String(string(db.Status.OsokStatus.Ocid)),
		UpdateTableDetails: updateDetails,
	}

	_, err = client.UpdateTable(ctx, req)
	return err
}

// DeleteTable deletes the NoSQL table for the given OCID.
func (c *NoSQLDatabaseServiceManager) DeleteTable(ctx context.Context, tableId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := nosql.DeleteTableRequest{
		TableNameOrId: common.String(string(tableId)),
		IsIfExists:    common.Bool(true),
	}

	_, err = client.DeleteTable(ctx, req)
	return err
}

// getRetryPolicy returns a retry policy that waits while a table is in CREATING state.
func (c *NoSQLDatabaseServiceManager) getRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(nosql.GetTableResponse); ok {
			return resp.LifecycleState == nosql.TableLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
