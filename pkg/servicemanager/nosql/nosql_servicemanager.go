/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that NoSQLDatabaseServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &NoSQLDatabaseServiceManager{}

// NoSQLDatabaseServiceManager implements OSOKServiceManager for OCI NoSQL Database.
type NoSQLDatabaseServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        NosqlClientInterface
}

// NewNoSQLDatabaseServiceManager creates a new NoSQLDatabaseServiceManager.
func NewNoSQLDatabaseServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *NoSQLDatabaseServiceManager {
	return &NoSQLDatabaseServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the NoSQLDatabase resource against OCI.
func (c *NoSQLDatabaseServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	db, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	tableInstance, response, err := c.resolveTableForReconcile(ctx, db)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if response != nil {
		return *response, nil
	}

	return reconcileLifecycleStatus(&db.Status.OsokStatus, tableInstance, c.Log), nil
}

// Delete handles deletion of the NoSQL table (called by the finalizer).
func (c *NoSQLDatabaseServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	db, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	tableID, done := c.resolveDeleteTableID(db)
	if done {
		return true, nil
	}

	currentTable, done, err := c.getTableForDelete(ctx, tableID)
	if done || err != nil {
		return done, err
	}

	done, handled, err := c.handleExistingDeleteTableWorkRequest(ctx, db, tableID, currentTable)
	if handled || err != nil {
		return done, err
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting NoSQL table %s", tableID))
	if _, err := c.submitDeleteTable(ctx, tableID); err != nil && !isNotFoundServiceError(err) {
		c.Log.ErrorLog(err, "Error while deleting NoSQL table")
		return false, err
	}

	return false, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *NoSQLDatabaseServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *NoSQLDatabaseServiceManager) convert(obj runtime.Object) (*ociv1beta1.NoSQLDatabase, error) {
	db, ok := obj.(*ociv1beta1.NoSQLDatabase)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for NoSQLDatabase")
	}
	return db, nil
}

func (c *NoSQLDatabaseServiceManager) resolveDeleteTableID(db *ociv1beta1.NoSQLDatabase) (ociv1beta1.OCID, bool) {
	if db.Status.OsokStatus.Ocid != "" {
		return db.Status.OsokStatus.Ocid, false
	}
	if db.Spec.TableId == "" {
		c.Log.InfoLog("NoSQL table has no OCID, nothing to delete")
		return "", true
	}

	db.Status.OsokStatus.Ocid = db.Spec.TableId
	return db.Status.OsokStatus.Ocid, false
}

func (c *NoSQLDatabaseServiceManager) getTableForDelete(
	ctx context.Context,
	tableID ociv1beta1.OCID,
) (*nosql.Table, bool, error) {
	currentTable, err := c.GetTable(ctx, tableID, nil)
	if err == nil {
		return currentTable, false, nil
	}
	if isNotFoundServiceError(err) {
		return nil, true, nil
	}

	c.Log.ErrorLog(err, "Error while getting NoSQL table during delete")
	return nil, false, err
}

func (c *NoSQLDatabaseServiceManager) handleExistingDeleteTableWorkRequest(
	ctx context.Context,
	db *ociv1beta1.NoSQLDatabase,
	tableID ociv1beta1.OCID,
	currentTable *nosql.Table,
) (bool, bool, error) {
	workRequestID, err := c.findDeleteTableWorkRequestID(ctx, resolveDeleteTableCompartmentID(db, currentTable), tableID)
	if err != nil {
		return false, true, err
	}
	if workRequestID == nil {
		return false, false, nil
	}

	completed, inProgress, err := c.handleDeleteTableWorkRequest(ctx, *workRequestID)
	if err != nil {
		return false, true, err
	}
	if inProgress {
		return false, true, nil
	}

	return completed, completed, nil
}

func resolveDeleteTableCompartmentID(db *ociv1beta1.NoSQLDatabase, currentTable *nosql.Table) ociv1beta1.OCID {
	if db.Spec.CompartmentId != "" {
		return db.Spec.CompartmentId
	}
	if currentTable != nil && currentTable.CompartmentId != nil {
		return ociv1beta1.OCID(*currentTable.CompartmentId)
	}

	return ""
}

func (c *NoSQLDatabaseServiceManager) handleDeleteTableWorkRequest(ctx context.Context, workRequestID string) (bool, bool, error) {
	workRequest, err := c.getTableWorkRequest(ctx, workRequestID)
	if err != nil {
		return false, false, err
	}

	switch workRequest.Status {
	case nosql.WorkRequestStatusAccepted,
		nosql.WorkRequestStatusInProgress,
		nosql.WorkRequestStatusCanceling:
		return false, true, nil
	case nosql.WorkRequestStatusSucceeded:
		return true, false, nil
	case nosql.WorkRequestStatusFailed,
		nosql.WorkRequestStatusCanceled:
		return false, false, fmt.Errorf("NoSQL delete work request %s ended with status %s", workRequestID, workRequest.Status)
	default:
		return false, false, nil
	}
}
