/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/util"
)

// Compile-time check that NoSQLDatabaseServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &NoSQLDatabaseServiceManager{}

// NoSQLDatabaseServiceManager implements OSOKServiceManager for OCI NoSQL Database.
type NoSQLDatabaseServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
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

	var tableInstance *nosql.Table

	if strings.TrimSpace(string(db.Spec.TableId)) == "" {
		// No ID provided — check by name or create
		tableOcid, err := c.GetTableOcid(ctx, *db)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if tableOcid == nil {
			// Create a new NoSQL table (async operation — returns work request, not OCID directly)
			_, err := c.CreateTable(ctx, *db)
			if err != nil {
				db.Status.OsokStatus = util.UpdateOSOKStatusCondition(db.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Create NoSQL table failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				db.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
				c.Log.ErrorLog(err, "Create NoSQL table bad request")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			c.Log.InfoLog(fmt.Sprintf("NoSQL table %s is Provisioning", db.Spec.Name))
			db.Status.OsokStatus = util.UpdateOSOKStatusCondition(db.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "NoSQL table Provisioning", c.Log)

			// Poll by name to get the OCID once the table is created
			tableOcid, err = c.GetTableOcid(ctx, *db)
			if err != nil {
				c.Log.ErrorLog(err, "Error while looking up NoSQL table after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			if tableOcid == nil {
				// Table not yet visible — requeue
				return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true, RequeueDuration: 30 * time.Second}, nil
			}

			tableInstance, err = c.GetTable(ctx, *tableOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting NoSQL table after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting existing NoSQL table %s", *tableOcid))
			tableInstance, err = c.GetTable(ctx, *tableOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting NoSQL table by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}

		db.Status.OsokStatus.Ocid = ociv1beta1.OCID(*tableInstance.Id)
		db.Status.OsokStatus = util.UpdateOSOKStatusCondition(db.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("NoSQL table %s is %s", *tableInstance.Name, tableInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("NoSQL table %s is %s", *tableInstance.Name, tableInstance.LifecycleState))

	} else {
		// Bind to an existing table by ID
		tableInstance, err = c.GetTable(ctx, db.Spec.TableId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing NoSQL table")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateTable(ctx, db); err != nil {
			c.Log.ErrorLog(err, "Error while updating NoSQL table")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		db.Status.OsokStatus = util.UpdateOSOKStatusCondition(db.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "NoSQL table Bound/Updated", c.Log)
		c.Log.InfoLog(fmt.Sprintf("NoSQL table %s is bound/updated", *tableInstance.Name))
	}

	db.Status.OsokStatus.Ocid = ociv1beta1.OCID(*tableInstance.Id)
	if db.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		db.Status.OsokStatus.CreatedAt = &now
	}

	if tableInstance.LifecycleState == nosql.TableLifecycleStateFailed {
		db.Status.OsokStatus = util.UpdateOSOKStatusCondition(db.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("NoSQL table %s creation Failed", *tableInstance.Name), c.Log)
		c.Log.InfoLog(fmt.Sprintf("NoSQL table %s creation Failed", *tableInstance.Name))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the NoSQL table (called by the finalizer).
func (c *NoSQLDatabaseServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	db, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if db.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("NoSQL table has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting NoSQL table %s", db.Status.OsokStatus.Ocid))
	if err := c.DeleteTable(ctx, db.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting NoSQL table")
		return false, err
	}

	return true, nil
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
