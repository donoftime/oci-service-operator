/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"context"
	"errors"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type AdbServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        DatabaseClientInterface
}

func NewAdbServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *AdbServiceManager {
	return &AdbServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

func (c *AdbServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	autonomousDatabases, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	adbInstance, response, done, err := c.resolveAdbInstance(ctx, autonomousDatabases, req)
	if err != nil || done {
		return response, err
	}

	lifecycleResponse := reconcileLifecycleStatus(&autonomousDatabases.Status.OsokStatus, adbInstance, c.Log)
	if !lifecycleResponse.IsSuccessful {
		return lifecycleResponse, nil
	}

	if autonomousDatabases.Spec.Wallet.WalletPassword.Secret.SecretName != "" {
		c.Log.InfoLog(fmt.Sprintf("Wallet Password Secret Name provided for %s Autonomous Database", autonomousDatabases.Spec.DisplayName))
		response, err := c.GenerateWallet(ctx, *adbInstance.Id, *adbInstance.DisplayName, autonomousDatabases.Spec.Wallet.WalletPassword.Secret.SecretName,
			autonomousDatabases.Namespace, autonomousDatabases.Spec.Wallet.WalletName, autonomousDatabases.Name)
		return servicemanager.OSOKResponse{IsSuccessful: response}, err
	} else {
		c.Log.InfoLog(fmt.Sprintf("Wallet Password Secret Name is empty. Not creating wallet for %s Autonomous Database",
			autonomousDatabases.Spec.DisplayName))
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func isValidUpdate(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return hasAdbFieldUpdates(autonomousDatabases, adbInstance) ||
		hasAdbOptionalBoolUpdates(autonomousDatabases, adbInstance) ||
		hasAdbTagUpdates(autonomousDatabases, adbInstance)
}

func hasAdbFieldUpdates(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return adbDisplayNameUpdated(autonomousDatabases, adbInstance) ||
		adbDbNameUpdated(autonomousDatabases, adbInstance) ||
		adbCpuCoreCountUpdated(autonomousDatabases, adbInstance) ||
		adbStorageUpdated(autonomousDatabases, adbInstance) ||
		adbDbWorkloadUpdated(autonomousDatabases, adbInstance) ||
		adbDbVersionUpdated(autonomousDatabases, adbInstance) ||
		adbLicenseModelUpdated(autonomousDatabases, adbInstance)
}

func hasAdbOptionalBoolUpdates(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return shouldUpdateOptionalBool(autonomousDatabases.Spec.HasExplicitIsAutoScalingEnabled(), autonomousDatabases.Spec.IsAutoScalingEnabled, adbInstance.IsAutoScalingEnabled) ||
		shouldUpdateOptionalBool(autonomousDatabases.Spec.HasExplicitIsFreeTier(), autonomousDatabases.Spec.IsFreeTier, adbInstance.IsFreeTier)
}

func hasAdbTagUpdates(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	if autonomousDatabases.Spec.FreeFormTags != nil && !reflect.DeepEqual(autonomousDatabases.Spec.FreeFormTags, adbInstance.FreeformTags) {
		return true
	}

	if autonomousDatabases.Spec.DefinedTags == nil {
		return false
	}

	defTag := *util.ConvertToOciDefinedTags(&autonomousDatabases.Spec.DefinedTags)
	return !reflect.DeepEqual(adbInstance.DefinedTags, defTag)
}

func adbDisplayNameUpdated(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return autonomousDatabases.Spec.DisplayName != "" && autonomousDatabases.Spec.DisplayName != *adbInstance.DisplayName
}

func adbDbNameUpdated(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return autonomousDatabases.Spec.DbName != "" && autonomousDatabases.Spec.DbName != *adbInstance.DbName
}

func adbCpuCoreCountUpdated(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return autonomousDatabases.Spec.CpuCoreCount != 0 && autonomousDatabases.Spec.CpuCoreCount != *adbInstance.CpuCoreCount
}

func adbStorageUpdated(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return autonomousDatabases.Spec.DataStorageSizeInTBs != 0 &&
		autonomousDatabases.Spec.DataStorageSizeInTBs != *adbInstance.DataStorageSizeInTBs
}

func adbDbWorkloadUpdated(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return autonomousDatabases.Spec.DbWorkload != "" && autonomousDatabases.Spec.DbWorkload != string(adbInstance.DbWorkload)
}

func adbDbVersionUpdated(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return autonomousDatabases.Spec.DbVersion != "" && autonomousDatabases.Spec.DbVersion != *adbInstance.DbVersion
}

func adbLicenseModelUpdated(autonomousDatabases ociv1beta1.AutonomousDatabases, adbInstance database.AutonomousDatabase) bool {
	return autonomousDatabases.Spec.LicenseModel != "" && autonomousDatabases.Spec.LicenseModel != string(adbInstance.LicenseModel)
}

func (c *AdbServiceManager) resolveAdbInstance(ctx context.Context, autonomousDatabases *ociv1beta1.AutonomousDatabases,
	req ctrl.Request) (*database.AutonomousDatabase, servicemanager.OSOKResponse, bool, error) {
	if strings.TrimSpace(string(autonomousDatabases.Spec.AdbId)) == "" {
		c.Log.DebugLog("AutonomousDatabase Id is empty. Check if adb is already existing.")
		return c.resolveManagedAdb(ctx, autonomousDatabases, req)
	}

	return c.resolveBoundAdb(ctx, autonomousDatabases)
}

func (c *AdbServiceManager) resolveManagedAdb(ctx context.Context, autonomousDatabases *ociv1beta1.AutonomousDatabases,
	req ctrl.Request) (*database.AutonomousDatabase, servicemanager.OSOKResponse, bool, error) {
	adbOcid, err := c.GetAdbOcid(ctx, *autonomousDatabases)
	if err != nil {
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if adbOcid == nil {
		return c.createManagedAdb(ctx, autonomousDatabases, req)
	}

	c.Log.InfoLog(fmt.Sprintf("Getting Autonomous Database %s", *adbOcid))
	adbInstance, err := c.GetAdb(ctx, *adbOcid, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting Autonomous database")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return adbInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *AdbServiceManager) resolveBoundAdb(ctx context.Context, autonomousDatabases *ociv1beta1.AutonomousDatabases) (*database.AutonomousDatabase, servicemanager.OSOKResponse, bool, error) {
	adbInstance, err := c.GetAdb(ctx, autonomousDatabases.Spec.AdbId, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting Autonomous database")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	if isValidUpdate(*autonomousDatabases, *adbInstance) {
		if err = c.UpdateAdb(ctx, autonomousDatabases); err != nil {
			c.Log.ErrorLog(err, "Error while updating Autonomous database")
			return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
		}
		c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is updated successfully", *adbInstance.DisplayName))
	} else {
		c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is bounded successfully", *adbInstance.DisplayName))
	}

	return adbInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *AdbServiceManager) createManagedAdb(ctx context.Context, autonomousDatabases *ociv1beta1.AutonomousDatabases,
	req ctrl.Request) (*database.AutonomousDatabase, servicemanager.OSOKResponse, bool, error) {
	pwd, err := c.getAdminPassword(ctx, autonomousDatabases, req.Namespace)
	if err != nil {
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	resp, err := c.CreateAdb(ctx, *autonomousDatabases, pwd)
	if err != nil {
		return c.handleCreateAdbError(autonomousDatabases, err)
	}

	c.markAdbProvisioning(autonomousDatabases, *resp.Id)

	retryPolicy := c.getAdbRetryPolicy(9)
	adbInstance, err := c.GetAdb(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting Autonomous database")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return adbInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *AdbServiceManager) getAdminPassword(ctx context.Context, autonomousDatabases *ociv1beta1.AutonomousDatabases,
	namespace string) (string, error) {
	c.Log.DebugLog("Getting Admin password from Secret")
	pwdMap, err := c.CredentialClient.GetSecret(ctx, autonomousDatabases.Spec.AdminPassword.Secret.SecretName, namespace)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting the admin password secret")
		return "", err
	}

	pwd, ok := pwdMap["password"]
	if !ok {
		err = errors.New("password key in admin password secret is not found")
		c.Log.ErrorLog(err, "password key in admin password secret is not found")
		return "", err
	}

	return string(pwd), nil
}

func (c *AdbServiceManager) handleCreateAdbError(autonomousDatabases *ociv1beta1.AutonomousDatabases,
	err error) (*database.AutonomousDatabase, servicemanager.OSOKResponse, bool, error) {
	autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
		ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
	if serviceErr, ok := err.(common.ServiceError); ok && serviceErr.GetHTTPStatusCode() == 400 &&
		serviceErr.GetCode() == "InvalidParameter" {
		autonomousDatabases.Status.OsokStatus.Message = serviceErr.GetCode()
		c.Log.ErrorLog(err, "Create AutonomousDatabase failed")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, nil
	}

	return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
}

func (c *AdbServiceManager) markAdbProvisioning(autonomousDatabases *ociv1beta1.AutonomousDatabases, adbID string) {
	c.Log.InfoLog(fmt.Sprintf("AutonomousDatabase %s is Provisioning", autonomousDatabases.Spec.DisplayName))
	autonomousDatabases.Status.OsokStatus = util.UpdateOSOKStatusCondition(autonomousDatabases.Status.OsokStatus,
		ociv1beta1.Provisioning, v1.ConditionTrue, "", "AutonomousDatabase Provisioning", c.Log)
	autonomousDatabases.Status.OsokStatus.Ocid = ociv1beta1.OCID(adbID)
}

func (c *AdbServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	autonomousDatabases, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	adbID := resolveDeleteAdbID(autonomousDatabases)
	if adbID == "" {
		return true, nil
	}

	if _, err := c.GetAdb(ctx, adbID, nil); err != nil {
		if isNotFoundServiceError(err) {
			return c.finalizeDeleteWalletSecret(ctx, autonomousDatabases)
		}
		return false, err
	}

	workRequestID, err := c.submitDeleteAdb(ctx, adbID)
	if err != nil && !isNotFoundServiceError(err) {
		return false, err
	}
	if workRequestID != nil && *workRequestID != "" {
		c.Log.InfoLog(fmt.Sprintf("Submitted Autonomous Database delete work request %s for %s", *workRequestID, adbID))
	}
	return false, nil
}

func resolveDeleteAdbID(autonomousDatabases *ociv1beta1.AutonomousDatabases) ociv1beta1.OCID {
	if autonomousDatabases.Status.OsokStatus.Ocid != "" {
		return autonomousDatabases.Status.OsokStatus.Ocid
	}

	return autonomousDatabases.Spec.AdbId
}

func (c *AdbServiceManager) finalizeDeleteWalletSecret(ctx context.Context, autonomousDatabases *ociv1beta1.AutonomousDatabases) (bool, error) {
	walletName := walletSecretName(autonomousDatabases)
	if walletName == "" {
		return true, nil
	}

	c.logPreservedLegacyWalletSecret(ctx, autonomousDatabases, walletName)
	if _, secretErr := servicemanager.DeleteOwnedSecretIfPresent(ctx, c.CredentialClient, walletName, autonomousDatabases.Namespace, autonomousDatabaseKindName, autonomousDatabases.Name); secretErr != nil {
		c.Log.ErrorLog(secretErr, "Error while deleting Autonomous Database wallet secret")
	}

	return true, nil
}

func (c *AdbServiceManager) logPreservedLegacyWalletSecret(ctx context.Context, autonomousDatabases *ociv1beta1.AutonomousDatabases, walletName string) {
	existingSecret, err := c.CredentialClient.GetSecret(ctx, walletName, autonomousDatabases.Namespace)
	if err != nil {
		if !servicemanager.IsSecretNotFoundError(err) {
			c.Log.ErrorLog(err, "Error while inspecting Autonomous Database wallet secret ownership")
		}
		return
	}

	if servicemanager.SecretOwnedBy(existingSecret, autonomousDatabaseKindName, autonomousDatabases.Name) {
		return
	}

	c.Log.InfoLog(fmt.Sprintf(
		"Preserving legacy Autonomous Database wallet secret %s/%s because it is not owned by %s %s",
		autonomousDatabases.Namespace,
		walletName,
		autonomousDatabaseKindName,
		autonomousDatabases.Name,
	))
}

func (c *AdbServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {

	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *AdbServiceManager) convert(obj runtime.Object) (*ociv1beta1.AutonomousDatabases, error) {
	copy, err := obj.(*ociv1beta1.AutonomousDatabases)
	if !err {
		return nil, fmt.Errorf("failed to convert the type assertion for Autonomous Databases")
	}
	return copy, nil
}

func (c *AdbServiceManager) getAdbRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(database.GetAutonomousDatabaseResponse); ok {
			return resp.LifecycleState == "PROVISIONING"
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
