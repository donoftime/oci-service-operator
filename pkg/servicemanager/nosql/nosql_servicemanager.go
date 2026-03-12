/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
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

	if db.Status.OsokStatus.Ocid == "" {
		if db.Spec.TableId == "" {
			c.Log.InfoLog("NoSQL table has no OCID, nothing to delete")
			return true, nil
		}
		db.Status.OsokStatus.Ocid = db.Spec.TableId
	}

	if _, err := c.GetTable(ctx, db.Status.OsokStatus.Ocid, nil); err != nil {
		if isNotFoundServiceError(err) {
			return true, nil
		}
		c.Log.ErrorLog(err, "Error while getting NoSQL table during delete")
		return false, err
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting NoSQL table %s", db.Status.OsokStatus.Ocid))
	if err := c.DeleteTable(ctx, db.Status.OsokStatus.Ocid); err != nil && !isNotFoundServiceError(err) {
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
