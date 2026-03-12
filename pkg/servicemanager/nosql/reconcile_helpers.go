/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql

import (
	"context"
	goerrors "errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"

	"github.com/oracle/oci-service-operator/pkg/util"
)

func (c *NoSQLDatabaseServiceManager) resolveTableForReconcile(ctx context.Context, db *ociv1beta1.NoSQLDatabase) (*nosql.Table, *servicemanager.OSOKResponse, error) {
	if strings.TrimSpace(string(db.Spec.TableId)) != "" {
		return c.bindTableByID(ctx, db)
	}

	if strings.TrimSpace(string(db.Status.OsokStatus.Ocid)) != "" {
		tableInstance, err := c.GetTable(ctx, db.Status.OsokStatus.Ocid, nil)
		if err != nil {
			if !isNotFoundServiceError(err) {
				return nil, nil, err
			}
			db.Status.OsokStatus.Ocid = ""
		} else {
			if err := c.UpdateTable(ctx, db); err != nil {
				c.Log.ErrorLog(err, "Error while updating NoSQL table")
				return nil, nil, err
			}
			return tableInstance, nil, nil
		}
	}

	return c.createOrLookupTable(ctx, db)
}

func (c *NoSQLDatabaseServiceManager) createOrLookupTable(ctx context.Context, db *ociv1beta1.NoSQLDatabase) (*nosql.Table, *servicemanager.OSOKResponse, error) {
	tableOcid, err := c.GetTableOcid(ctx, *db)
	if err != nil {
		return nil, nil, err
	}
	if tableOcid == nil {
		return c.createTableAndResolve(ctx, db)
	}

	tableInstance, err := c.GetTable(ctx, *tableOcid, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting NoSQL table by OCID")
		return nil, nil, err
	}

	db.Status.OsokStatus.Ocid = ociv1beta1.OCID(safeString(tableInstance.Id))
	if err := c.UpdateTable(ctx, db); err != nil {
		c.Log.ErrorLog(err, "Error while updating NoSQL table")
		return nil, nil, err
	}
	return tableInstance, nil, nil
}

func (c *NoSQLDatabaseServiceManager) createTableAndResolve(ctx context.Context, db *ociv1beta1.NoSQLDatabase) (*nosql.Table, *servicemanager.OSOKResponse, error) {
	if _, err := c.CreateTable(ctx, *db); err != nil {
		db.Status.OsokStatus = util.UpdateOSOKStatusCondition(db.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
		var badRequestErr errorutil.BadRequestOciError
		if !goerrors.As(err, &badRequestErr) {
			c.Log.ErrorLog(err, "Create NoSQL table failed")
			return nil, nil, err
		}
		if serviceErr, ok := common.IsServiceError(err); ok {
			db.Status.OsokStatus.Message = serviceErr.GetCode()
		}
		c.Log.ErrorLog(err, "Create NoSQL table bad request")
		response := servicemanager.OSOKResponse{IsSuccessful: false}
		return nil, &response, err
	}

	c.Log.InfoLog(fmt.Sprintf("NoSQL table %s is Provisioning", db.Spec.Name))
	db.Status.OsokStatus = util.UpdateOSOKStatusCondition(db.Status.OsokStatus,
		ociv1beta1.Provisioning, v1.ConditionTrue, "", "NoSQL table Provisioning", c.Log)

	tableOcid, err := c.GetTableOcid(ctx, *db)
	if err != nil {
		c.Log.ErrorLog(err, "Error while looking up NoSQL table after create")
		return nil, nil, err
	}
	if tableOcid == nil {
		response := servicemanager.OSOKResponse{
			IsSuccessful:    false,
			ShouldRequeue:   true,
			RequeueDuration: tableRequeueDuration,
		}
		return nil, &response, nil
	}

	tableInstance, err := c.GetTable(ctx, *tableOcid, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting NoSQL table after create")
		return nil, nil, err
	}

	db.Status.OsokStatus.Ocid = ociv1beta1.OCID(safeString(tableInstance.Id))
	return tableInstance, nil, nil
}

func (c *NoSQLDatabaseServiceManager) bindTableByID(ctx context.Context, db *ociv1beta1.NoSQLDatabase) (*nosql.Table, *servicemanager.OSOKResponse, error) {
	tableInstance, err := c.GetTable(ctx, db.Spec.TableId, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting existing NoSQL table")
		return nil, nil, err
	}

	db.Status.OsokStatus.Ocid = db.Spec.TableId
	if err := c.UpdateTable(ctx, db); err != nil {
		c.Log.ErrorLog(err, "Error while updating NoSQL table")
		return nil, nil, err
	}

	return tableInstance, nil, nil
}
