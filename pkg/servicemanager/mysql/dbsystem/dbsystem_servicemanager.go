/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"errors"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"time"
)

type DbSystemServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        MySQLDbSystemClientInterface
}

func NewDbSystemServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *DbSystemServiceManager {
	return &DbSystemServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

func (c *DbSystemServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	mysqlDbSystem, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	mySqlDbSystemInstance, response, done, err := c.resolveDbSystemForReconcile(ctx, mysqlDbSystem, req)
	if err != nil || done {
		return response, err
	}

	lifecycleResponse := reconcileLifecycleStatus(&mysqlDbSystem.Status.OsokStatus, mySqlDbSystemInstance, c.Log)
	if !lifecycleResponse.IsSuccessful {
		return lifecycleResponse, nil
	}

	if mySqlDbSystemInstance.LifecycleState == mysql.DbSystemLifecycleStateActive {
		_, err := c.addToSecret(ctx, mysqlDbSystem.Namespace, mysqlDbSystem.Name, *mySqlDbSystemInstance)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				return servicemanager.OSOKResponse{IsSuccessful: true}, nil
			}
			c.Log.InfoLog("Secret creation failed")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func isValidUpdate(dbSystem ociv1beta1.MySqlDbSystem, mySqlDbInstance mysql.DbSystem) bool {
	return mySQLFieldUpdates(dbSystem, mySqlDbInstance) || mySQLTagUpdates(dbSystem, mySqlDbInstance)
}

func (c *DbSystemServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	mysqlDbSystem, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	dbSystemID := resolveDeleteMySQLDbSystemID(mysqlDbSystem)
	if dbSystemID == "" {
		return true, nil
	}

	currentDbSystem, done, handled, err := c.getMySQLDbSystemForDelete(ctx, mysqlDbSystem, dbSystemID)
	if handled {
		return done, err
	}

	done, handled, err = c.handleExistingDeleteMySQLWorkRequest(ctx, mysqlDbSystem, dbSystemID, currentDbSystem)
	if handled {
		return done, err
	}

	if _, err := c.submitDeleteMySqlDbSystem(ctx, dbSystemID); err != nil && !isNotFoundServiceError(err) {
		return false, err
	}
	return false, nil
}

func (c *DbSystemServiceManager) finalizeDeleteSecret(ctx context.Context, mysqlDbSystem *ociv1beta1.MySqlDbSystem) (bool, error) {
	if _, secretErr := c.deleteFromSecret(ctx, mysqlDbSystem.Namespace, mysqlDbSystem.Name); secretErr != nil {
		c.Log.ErrorLog(secretErr, "Error while deleting MySqlDbSystem secret")
	}
	return true, nil
}

func resolveDeleteMySQLDbSystemID(mysqlDbSystem *ociv1beta1.MySqlDbSystem) ociv1beta1.OCID {
	if mysqlDbSystem.Status.OsokStatus.Ocid != "" {
		return mysqlDbSystem.Status.OsokStatus.Ocid
	}

	return mysqlDbSystem.Spec.MySqlDbSystemId
}

func (c *DbSystemServiceManager) getMySQLDbSystemForDelete(
	ctx context.Context,
	mysqlDbSystem *ociv1beta1.MySqlDbSystem,
	dbSystemID ociv1beta1.OCID,
) (*mysql.DbSystem, bool, bool, error) {
	currentDbSystem, err := c.GetMySqlDbSystem(ctx, dbSystemID, nil)
	if err == nil {
		return currentDbSystem, false, false, nil
	}
	if isNotFoundServiceError(err) {
		done, secretErr := c.finalizeDeleteSecret(ctx, mysqlDbSystem)
		return nil, done, true, secretErr
	}
	if isRetryableReadServiceError(err) {
		c.Log.ErrorLog(err, "Transient MySqlDbSystem read failure during delete; requeueing")
		return nil, false, true, nil
	}

	return nil, false, true, err
}

func (c *DbSystemServiceManager) handleExistingDeleteMySQLWorkRequest(
	ctx context.Context,
	mysqlDbSystem *ociv1beta1.MySqlDbSystem,
	dbSystemID ociv1beta1.OCID,
	currentDbSystem *mysql.DbSystem,
) (bool, bool, error) {
	workRequestID, err := c.findDeleteMySQLWorkRequestID(ctx, resolveDeleteMySQLCompartmentID(mysqlDbSystem, currentDbSystem), dbSystemID)
	if err != nil {
		if isRetryableReadServiceError(err) {
			c.Log.ErrorLog(err, "Transient MySqlDbSystem work request lookup failure during delete; requeueing")
			return false, true, nil
		}
		return false, true, err
	}
	if workRequestID == nil {
		return false, false, nil
	}

	completed, inProgress, err := c.handleDeleteMySQLWorkRequest(ctx, *workRequestID)
	if err != nil {
		if isRetryableReadServiceError(err) {
			c.Log.ErrorLog(err, "Transient MySqlDbSystem work request read failure during delete; requeueing")
			return false, true, nil
		}
		return false, true, err
	}
	if inProgress {
		return false, true, nil
	}
	if !completed {
		return false, false, nil
	}

	done, err := c.finalizeDeleteSecret(ctx, mysqlDbSystem)
	return done, true, err
}

func resolveDeleteMySQLCompartmentID(mysqlDbSystem *ociv1beta1.MySqlDbSystem, currentDbSystem *mysql.DbSystem) ociv1beta1.OCID {
	if mysqlDbSystem.Spec.CompartmentId != "" {
		return mysqlDbSystem.Spec.CompartmentId
	}
	if currentDbSystem != nil && currentDbSystem.CompartmentId != nil {
		return ociv1beta1.OCID(*currentDbSystem.CompartmentId)
	}

	return ""
}

func (c *DbSystemServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *DbSystemServiceManager) convert(obj runtime.Object) (*ociv1beta1.MySqlDbSystem, error) {
	copy, err := obj.(*ociv1beta1.MySqlDbSystem)
	if !err {
		return nil, fmt.Errorf("failed to convert the type assertion for MySqlDbSystem")
	}
	return copy, nil
}

func (c *DbSystemServiceManager) getDbSystemRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(mysql.GetDbSystemResponse); ok {
			return resp.LifecycleState == "CREATING"
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

func (c *DbSystemServiceManager) handleRetryableReadError(mysqlDbSystem *ociv1beta1.MySqlDbSystem, resourceID ociv1beta1.OCID,
	operation string, err error) (servicemanager.OSOKResponse, bool, error) {
	if !isRetryableReadServiceError(err) {
		return servicemanager.OSOKResponse{}, false, err
	}

	c.Log.ErrorLog(err, fmt.Sprintf("Transient MySqlDbSystem read failure while %s; requeueing", operation))
	response := requeueForTransientReadFailure(&mysqlDbSystem.Status.OsokStatus, resourceID, mysqlDbSystem.Spec.DisplayName, operation, err, c.Log)
	return response, true, nil
}

func (c *DbSystemServiceManager) resolveDbSystemForReconcile(ctx context.Context, mysqlDbSystem *ociv1beta1.MySqlDbSystem,
	req ctrl.Request) (*mysql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	if strings.TrimSpace(string(mysqlDbSystem.Spec.MySqlDbSystemId)) == "" {
		return c.resolveManagedDbSystem(ctx, mysqlDbSystem, req)
	}

	return c.resolveBoundDbSystem(ctx, mysqlDbSystem)
}

func (c *DbSystemServiceManager) resolveManagedDbSystem(ctx context.Context, mysqlDbSystem *ociv1beta1.MySqlDbSystem,
	req ctrl.Request) (*mysql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	c.Log.DebugLog("MySqlDbSystem Id is empty. Check if mysql DB exists.")

	mySqlDbSystemOcid, err := c.GetMySqlDbSystemOcid(ctx, *mysqlDbSystem)
	if err != nil {
		if response, handled, handleErr := c.handleRetryableReadError(mysqlDbSystem, mysqlDbSystem.Status.OsokStatus.Ocid, "listing existing MySqlDbSystems", err); handled {
			return nil, response, true, handleErr
		}
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if mySqlDbSystemOcid == nil {
		return c.createManagedDbSystem(ctx, mysqlDbSystem, req)
	}

	c.Log.InfoLog(fmt.Sprintf("Getting MySqlDbSystem %s", *mySqlDbSystemOcid))
	mySqlDbSystemInstance, err := c.GetMySqlDbSystem(ctx, *mySqlDbSystemOcid, nil)
	if err != nil {
		if response, handled, handleErr := c.handleRetryableReadError(mysqlDbSystem, *mySqlDbSystemOcid, "getting existing MySqlDbSystem", err); handled {
			return nil, response, true, handleErr
		}
		c.Log.ErrorLog(err, "Error while getting MySqlDbSystem database")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return mySqlDbSystemInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *DbSystemServiceManager) createManagedDbSystem(ctx context.Context, mysqlDbSystem *ociv1beta1.MySqlDbSystem,
	req ctrl.Request) (*mysql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	username, password, err := c.getAdminCredentials(ctx, mysqlDbSystem, req.Namespace)
	if err != nil {
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	resp, err := c.CreateDbSystem(ctx, *mysqlDbSystem, username, password)
	if err != nil {
		return c.handleCreateDbSystemError(mysqlDbSystem, err)
	}

	c.markDbSystemProvisioning(mysqlDbSystem, *resp.Id)
	retryPolicy := c.getDbSystemRetryPolicy(30)
	mySqlDbSystemInstance, err := c.GetMySqlDbSystem(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
	if err != nil {
		if response, handled, handleErr := c.handleRetryableReadError(mysqlDbSystem, ociv1beta1.OCID(*resp.Id), "observing newly created MySqlDbSystem", err); handled {
			return nil, response, true, handleErr
		}
		c.Log.ErrorLog(err, "Error while getting MySqlDbSystem")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return mySqlDbSystemInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *DbSystemServiceManager) resolveBoundDbSystem(ctx context.Context,
	mysqlDbSystem *ociv1beta1.MySqlDbSystem) (*mysql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	mySqlDbSystemInstance, err := c.GetMySqlDbSystem(ctx, mysqlDbSystem.Spec.MySqlDbSystemId, nil)
	if err != nil {
		if response, handled, handleErr := c.handleRetryableReadError(mysqlDbSystem, mysqlDbSystem.Spec.MySqlDbSystemId, "getting bound MySqlDbSystem", err); handled {
			return nil, response, true, handleErr
		}
		c.Log.ErrorLog(err, "Error while getting the MySqlDbSystem")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	if isValidUpdate(*mysqlDbSystem, *mySqlDbSystemInstance) {
		if err = c.UpdateMySqlDbSystem(ctx, mysqlDbSystem, mySqlDbSystemInstance); err != nil {
			c.Log.ErrorLog(err, "Error while updating MysqlDbSystem")
			return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
		}
		c.Log.InfoLog(fmt.Sprintf("MySqlDbSystem %s is updated successfully", *mySqlDbSystemInstance.DisplayName))
	} else {
		c.Log.InfoLog(fmt.Sprintf("MysqlDbSystem %s is bound successfully", *mySqlDbSystemInstance.DisplayName))
	}

	return mySqlDbSystemInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *DbSystemServiceManager) getAdminCredentials(ctx context.Context, mysqlDbSystem *ociv1beta1.MySqlDbSystem,
	namespace string) (string, string, error) {
	c.Log.DebugLog("Getting Admin Username from Secret")
	unameMap, err := c.CredentialClient.GetSecret(ctx, mysqlDbSystem.Spec.AdminUsername.Secret.SecretName, namespace)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting the admin secret")
		return "", "", err
	}

	uname, ok := unameMap["username"]
	if !ok {
		err = errors.New("username key in admin secret is not found")
		c.Log.ErrorLog(err, "username key in admin secret is not found")
		return "", "", err
	}

	c.Log.DebugLog("Getting Admin password from Secret")
	pwdMap, err := c.CredentialClient.GetSecret(ctx, mysqlDbSystem.Spec.AdminPassword.Secret.SecretName, namespace)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting the admin secret")
		return "", "", err
	}

	pwd, ok := pwdMap["password"]
	if !ok {
		err = errors.New("password key in admin secret is not found")
		c.Log.ErrorLog(err, "password key in admin secret is not found")
		return "", "", err
	}

	return string(uname), string(pwd), nil
}

func (c *DbSystemServiceManager) handleCreateDbSystemError(mysqlDbSystem *ociv1beta1.MySqlDbSystem,
	err error) (*mysql.DbSystem, servicemanager.OSOKResponse, bool, error) {
	mysqlDbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(mysqlDbSystem.Status.OsokStatus,
		ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
	var badRequestErr errorutil.BadRequestOciError
	if !errors.As(err, &badRequestErr) {
		c.Log.ErrorLog(err, "Assertion Error for BadRequestOciError")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if serviceErr, ok := common.IsServiceError(err); ok {
		mysqlDbSystem.Status.OsokStatus.Message = serviceErr.GetCode()
	}
	c.Log.ErrorLog(err, "Create MySqlDbSystem failed")
	return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
}

func (c *DbSystemServiceManager) markDbSystemProvisioning(mysqlDbSystem *ociv1beta1.MySqlDbSystem, dbSystemID string) {
	c.Log.InfoLog(fmt.Sprintf("MySqlDbSystem %s is Provisioning", mysqlDbSystem.Spec.DisplayName))
	mysqlDbSystem.Status.OsokStatus = util.UpdateOSOKStatusCondition(mysqlDbSystem.Status.OsokStatus,
		ociv1beta1.Provisioning, v1.ConditionTrue, "", "MySqlDbSystem Provisioning", c.Log)
	mysqlDbSystem.Status.OsokStatus.Ocid = ociv1beta1.OCID(dbSystemID)
}

func mySQLFieldUpdates(dbSystem ociv1beta1.MySqlDbSystem, mySqlDbInstance mysql.DbSystem) bool {
	return mySQLDisplayNameUpdated(dbSystem, mySqlDbInstance) ||
		mySQLDescriptionUpdated(dbSystem, mySqlDbInstance) ||
		mySQLConfigurationUpdated(dbSystem, mySqlDbInstance)
}

func mySQLTagUpdates(dbSystem ociv1beta1.MySqlDbSystem, mySqlDbInstance mysql.DbSystem) bool {
	if dbSystem.Spec.FreeFormTags != nil && !reflect.DeepEqual(dbSystem.Spec.FreeFormTags, mySqlDbInstance.FreeformTags) {
		return true
	}
	if dbSystem.Spec.DefinedTags == nil {
		return false
	}

	defTag := *util.ConvertToOciDefinedTags(&dbSystem.Spec.DefinedTags)
	return !reflect.DeepEqual(mySqlDbInstance.DefinedTags, defTag)
}

func mySQLDisplayNameUpdated(dbSystem ociv1beta1.MySqlDbSystem, mySqlDbInstance mysql.DbSystem) bool {
	return dbSystem.Spec.DisplayName != "" && dbSystem.Spec.DisplayName != *mySqlDbInstance.DisplayName
}

func mySQLDescriptionUpdated(dbSystem ociv1beta1.MySqlDbSystem, mySqlDbInstance mysql.DbSystem) bool {
	return dbSystem.Spec.Description != "" && dbSystem.Spec.Description != *mySqlDbInstance.Description
}

func mySQLConfigurationUpdated(dbSystem ociv1beta1.MySqlDbSystem, mySqlDbInstance mysql.DbSystem) bool {
	return dbSystem.Spec.ConfigurationId.Id != "" && string(dbSystem.Spec.ConfigurationId.Id) != *mySqlDbInstance.ConfigurationId
}

func (c *DbSystemServiceManager) handleDeleteMySQLWorkRequest(ctx context.Context, workRequestID string) (bool, bool, error) {
	workRequest, err := c.getMySQLWorkRequest(ctx, workRequestID)
	if err != nil {
		return false, false, err
	}

	switch workRequest.Status {
	case mysql.WorkRequestOperationStatusAccepted,
		mysql.WorkRequestOperationStatusInProgress,
		mysql.WorkRequestOperationStatusCanceling:
		return false, true, nil
	case mysql.WorkRequestOperationStatusSucceeded:
		return true, false, nil
	case mysql.WorkRequestOperationStatusFailed,
		mysql.WorkRequestOperationStatusCanceled:
		return false, false, fmt.Errorf("MySqlDbSystem delete work request %s ended with status %s", workRequestID, workRequest.Status)
	default:
		return false, false, nil
	}
}
