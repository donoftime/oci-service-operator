/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package postgresql

import (
	"context"
	goerrors "errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/psql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/util"
)

// Compile-time check that PostgresDbSystemServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &PostgresDbSystemServiceManager{}

// PostgresDbSystemServiceManager implements OSOKServiceManager for OCI Database with PostgreSQL.
type PostgresDbSystemServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        PostgresClientInterface
}

// NewPostgresDbSystemServiceManager creates a new PostgresDbSystemServiceManager.
func NewPostgresDbSystemServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *PostgresDbSystemServiceManager {
	return &PostgresDbSystemServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the PostgresDbSystem resource against OCI.
func (c *PostgresDbSystemServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	dbSystem, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	dbSystemInstance, response, done, err := c.resolveDbSystemForReconcile(ctx, dbSystem)
	if err != nil || done {
		return response, err
	}

	reconcileResponse := reconcileLifecycleStatus(&dbSystem.Status.OsokStatus, dbSystemInstance, c.Log)
	if !reconcileResponse.IsSuccessful {
		return reconcileResponse, nil
	}

	_, err = c.addToSecret(ctx, dbSystem.Namespace, dbSystem.Name, *dbSystemInstance)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		}
		c.Log.InfoLog("Secret creation failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return reconcileResponse, nil
}

func (c *PostgresDbSystemServiceManager) resolveDbSystemForReconcile(ctx context.Context,
	dbSystem *ociv1beta1.PostgresDbSystem) (*psql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	if strings.TrimSpace(string(dbSystem.Spec.PostgresDbSystemId)) == "" {
		return c.resolveManagedDbSystem(ctx, dbSystem)
	}

	return c.resolveBoundDbSystem(ctx, dbSystem)
}

func (c *PostgresDbSystemServiceManager) resolveManagedDbSystem(ctx context.Context,
	dbSystem *ociv1beta1.PostgresDbSystem) (*psql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	dbSystemOcid, err := c.GetPostgresDbSystemByName(ctx, *dbSystem)
	if err != nil {
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if dbSystemOcid == nil {
		return c.createManagedDbSystem(ctx, dbSystem)
	}

	c.Log.InfoLog(fmt.Sprintf("Getting existing PostgresDbSystem %s", *dbSystemOcid))
	dbSystemInstance, err := c.GetPostgresDbSystem(ctx, *dbSystemOcid)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting PostgresDbSystem by OCID")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return dbSystemInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *PostgresDbSystemServiceManager) createManagedDbSystem(ctx context.Context,
	dbSystem *ociv1beta1.PostgresDbSystem) (*psql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	resp, err := c.CreatePostgresDbSystem(ctx, *dbSystem)
	if err != nil {
		return c.handleCreateDbSystemError(dbSystem, err)
	}

	c.markPostgresDbSystemProvisioning(dbSystem, *resp.Id)
	dbSystemInstance, err := c.GetPostgresDbSystem(ctx, ociv1beta1.OCID(*resp.Id))
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting PostgresDbSystem after create")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return dbSystemInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *PostgresDbSystemServiceManager) resolveBoundDbSystem(ctx context.Context,
	dbSystem *ociv1beta1.PostgresDbSystem) (*psql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	dbSystemInstance, err := c.GetPostgresDbSystem(ctx, dbSystem.Spec.PostgresDbSystemId)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting existing PostgresDbSystem")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	if err = c.UpdatePostgresDbSystem(ctx, dbSystem); err != nil {
		c.Log.ErrorLog(err, "Error while updating PostgresDbSystem")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return dbSystemInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *PostgresDbSystemServiceManager) handleCreateDbSystemError(dbSystem *ociv1beta1.PostgresDbSystem,
	err error) (*psql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	dbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(dbSystem.Status.OsokStatus,
		ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
	var badRequestErr errorutil.BadRequestOciError
	if !goerrors.As(err, &badRequestErr) {
		c.Log.ErrorLog(err, "Create PostgresDbSystem failed")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if serviceErr, ok := common.IsServiceError(err); ok {
		dbSystem.Status.OsokStatus.Message = serviceErr.GetCode()
	}
	c.Log.ErrorLog(err, "Create PostgresDbSystem bad request")
	return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
}

func (c *PostgresDbSystemServiceManager) markPostgresDbSystemProvisioning(dbSystem *ociv1beta1.PostgresDbSystem, dbSystemID string) {
	c.Log.InfoLog(fmt.Sprintf("PostgresDbSystem %s is Provisioning", dbSystem.Spec.DisplayName))
	dbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(dbSystem.Status.OsokStatus,
		ociv1beta1.Provisioning, v1.ConditionTrue, "", "PostgresDbSystem Provisioning", c.Log)
	dbSystem.Status.OsokStatus.Ocid = ociv1beta1.OCID(dbSystemID)
}

// Delete handles deletion of the PostgreSQL DB system (called by the finalizer).
func (c *PostgresDbSystemServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	dbSystem, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := resolveDbSystemID(dbSystem.Status.OsokStatus.Ocid, dbSystem.Spec.PostgresDbSystemId)
	if err != nil {
		c.Log.InfoLog("PostgresDbSystem has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting PostgresDbSystem %s", targetID))
	if err := c.DeletePostgresDbSystem(ctx, targetID); err != nil {
		// Treat 404 as already deleted
		if isNotFoundServiceError(err) {
			c.Log.InfoLog("PostgresDbSystem not found, treating as already deleted")
		} else {
			c.Log.ErrorLog(err, "Error while deleting PostgresDbSystem")
			return false, err
		}
	}

	if _, err := c.GetPostgresDbSystem(ctx, targetID); err != nil {
		if !isNotFoundServiceError(err) {
			return false, err
		}
		if _, err := servicemanager.DeleteOwnedSecretIfPresent(ctx, c.CredentialClient, dbSystem.Name, dbSystem.Namespace, "PostgresDbSystem", dbSystem.Name); err != nil {
			c.Log.ErrorLog(err, "Error while deleting PostgresDbSystem secret")
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *PostgresDbSystemServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *PostgresDbSystemServiceManager) convert(obj runtime.Object) (*ociv1beta1.PostgresDbSystem, error) {
	dbSystem, ok := obj.(*ociv1beta1.PostgresDbSystem)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for PostgresDbSystem")
	}
	return dbSystem, nil
}
