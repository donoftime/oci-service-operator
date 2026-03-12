/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql

import (
	"context"
	"fmt"
	"reflect"

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
	GetWorkRequest(ctx context.Context, request nosql.GetWorkRequestRequest) (nosql.GetWorkRequestResponse, error)
	ListWorkRequests(ctx context.Context, request nosql.ListWorkRequestsRequest) (nosql.ListWorkRequestsResponse, error)
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

	tableID, err := resolveTableID(db.Status.OsokStatus.Ocid, db.Spec.TableId)
	if err != nil {
		return err
	}

	existingTable, err := c.GetTable(ctx, tableID, nil)
	if err != nil {
		return err
	}

	updateDetails, updateNeeded := buildUpdateTableDetails(db, existingTable)
	if !updateNeeded {
		return nil
	}

	req := nosql.UpdateTableRequest{
		TableNameOrId:      common.String(string(tableID)),
		UpdateTableDetails: updateDetails,
	}

	_, err = client.UpdateTable(ctx, req)
	return err
}

// DeleteTable deletes the NoSQL table for the given OCID.
func (c *NoSQLDatabaseServiceManager) DeleteTable(ctx context.Context, tableId ociv1beta1.OCID) error {
	_, err := c.submitDeleteTable(ctx, tableId)
	return err
}

func (c *NoSQLDatabaseServiceManager) submitDeleteTable(ctx context.Context, tableID ociv1beta1.OCID) (*string, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := nosql.DeleteTableRequest{
		TableNameOrId: common.String(string(tableID)),
		IsIfExists:    common.Bool(true),
	}

	resp, err := client.DeleteTable(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.OpcWorkRequestId, nil
}

func (c *NoSQLDatabaseServiceManager) findDeleteTableWorkRequestID(ctx context.Context, compartmentID, tableID ociv1beta1.OCID) (*string, error) {
	if !canFindDeleteTableWorkRequest(compartmentID, tableID) {
		return nil, nil
	}

	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := newDeleteTableWorkRequestListRequest(compartmentID)

	for {
		workRequestID, nextPage, err := c.findDeleteTableWorkRequestPage(ctx, client, req, tableID)
		if err != nil {
			return nil, err
		}
		if workRequestID != nil {
			return workRequestID, nil
		}
		if nextPage == nil || *nextPage == "" {
			return nil, nil
		}
		req.Page = nextPage
	}
}

func canFindDeleteTableWorkRequest(compartmentID, tableID ociv1beta1.OCID) bool {
	return compartmentID != "" && tableID != ""
}

func newDeleteTableWorkRequestListRequest(compartmentID ociv1beta1.OCID) nosql.ListWorkRequestsRequest {
	return nosql.ListWorkRequestsRequest{
		CompartmentId: common.String(string(compartmentID)),
		Limit:         common.Int(100),
	}
}

func (c *NoSQLDatabaseServiceManager) findDeleteTableWorkRequestPage(
	ctx context.Context,
	client NosqlClientInterface,
	req nosql.ListWorkRequestsRequest,
	tableID ociv1beta1.OCID,
) (*string, *string, error) {
	resp, err := client.ListWorkRequests(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	workRequestID := matchDeleteTableWorkRequest(resp.Items, tableID)
	if workRequestID != nil {
		return workRequestID, nil, nil
	}

	return nil, resp.OpcNextPage, nil
}

func matchDeleteTableWorkRequest(items []nosql.WorkRequestSummary, tableID ociv1beta1.OCID) *string {
	for _, item := range items {
		if workRequestID := matchDeleteTableWorkRequestSummary(item, tableID); workRequestID != nil {
			return workRequestID
		}
	}

	return nil
}

func matchDeleteTableWorkRequestSummary(item nosql.WorkRequestSummary, tableID ociv1beta1.OCID) *string {
	if !isDeleteTableWorkRequestSummary(item) {
		return nil
	}
	if !noSQLWorkRequestTargetsTable(item.Resources, tableID) {
		return nil
	}

	return item.Id
}

func isDeleteTableWorkRequestSummary(item nosql.WorkRequestSummary) bool {
	return item.OperationType == nosql.WorkRequestSummaryOperationTypeDeleteTable && item.Id != nil
}

func noSQLWorkRequestTargetsTable(resources []nosql.WorkRequestResource, tableID ociv1beta1.OCID) bool {
	for _, resource := range resources {
		if resource.Identifier != nil && *resource.Identifier == string(tableID) {
			return true
		}
	}
	return false
}

func (c *NoSQLDatabaseServiceManager) getTableWorkRequest(ctx context.Context, workRequestID string) (*nosql.WorkRequest, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := nosql.GetWorkRequestRequest{
		WorkRequestId: common.String(workRequestID),
	}

	resp, err := client.GetWorkRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.WorkRequest, nil
}

func buildUpdateTableDetails(db *ociv1beta1.NoSQLDatabase, existingTable *nosql.Table) (nosql.UpdateTableDetails, bool) {
	updateDetails := nosql.UpdateTableDetails{}
	updateNeeded := false

	if ddlStatementChanged(db.Spec.DdlStatement, existingTable.DdlStatement) {
		updateDetails.DdlStatement = common.String(db.Spec.DdlStatement)
		updateNeeded = true
	}

	if tableLimitsChanged(db.Spec.TableLimits, existingTable.TableLimits) {
		updateDetails.TableLimits = &nosql.TableLimits{
			MaxReadUnits:    common.Int(db.Spec.TableLimits.MaxReadUnits),
			MaxWriteUnits:   common.Int(db.Spec.TableLimits.MaxWriteUnits),
			MaxStorageInGBs: common.Int(db.Spec.TableLimits.MaxStorageInGBs),
		}
		updateNeeded = true
	}

	if freeformTagsChanged(db.Spec.FreeFormTags, existingTable.FreeformTags) {
		updateDetails.FreeformTags = db.Spec.FreeFormTags
		updateNeeded = true
	}

	if definedTagsChanged(db.Spec.DefinedTags, existingTable.DefinedTags) {
		updateDetails.DefinedTags = *util.ConvertToOciDefinedTags(&db.Spec.DefinedTags)
		updateNeeded = true
	}

	return updateDetails, updateNeeded
}

func ddlStatementChanged(desired string, existing *string) bool {
	return desired != "" && desired != safeString(existing)
}

func tableLimitsChanged(desired *ociv1beta1.NoSQLDatabaseTableLimits, existing *nosql.TableLimits) bool {
	if desired == nil {
		return false
	}
	if existing == nil {
		return true
	}

	return desired.MaxReadUnits != safeInt(existing.MaxReadUnits) ||
		desired.MaxWriteUnits != safeInt(existing.MaxWriteUnits) ||
		desired.MaxStorageInGBs != safeInt(existing.MaxStorageInGBs)
}

func freeformTagsChanged(desired map[string]string, existing map[string]string) bool {
	if desired == nil {
		return false
	}

	return !reflect.DeepEqual(existing, desired)
}

func definedTagsChanged(desired map[string]ociv1beta1.MapValue, existing map[string]map[string]interface{}) bool {
	if desired == nil {
		return false
	}

	return !reflect.DeepEqual(existing, *util.ConvertToOciDefinedTags(&desired))
}

func safeInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
