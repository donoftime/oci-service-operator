/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	"reflect"
)

type AdbServiceClient interface {
	CreateAdb(ctx context.Context, adb ociv1beta1.AutonomousDatabases) (database.AutonomousDatabase, error)

	UpdateAdb(ctx context.Context, request database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error)

	GetAdb(ctx context.Context, request database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error)

	DeleteAdb(ctx context.Context, adbId ociv1beta1.OCID) error

	servicemanager.OSOKServiceManager
}

// DatabaseClientInterface defines the OCI operations used by AdbServiceManager.
type DatabaseClientInterface interface {
	CreateAutonomousDatabase(ctx context.Context, request database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error)
	ListAutonomousDatabases(ctx context.Context, request database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error)
	GetAutonomousDatabase(ctx context.Context, request database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error)
	ChangeAutonomousDatabaseCompartment(ctx context.Context, request database.ChangeAutonomousDatabaseCompartmentRequest) (database.ChangeAutonomousDatabaseCompartmentResponse, error)
	UpdateAutonomousDatabase(ctx context.Context, request database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error)
	DeleteAutonomousDatabase(ctx context.Context, request database.DeleteAutonomousDatabaseRequest) (database.DeleteAutonomousDatabaseResponse, error)
}

func getDbClient(provider common.ConfigurationProvider) (database.DatabaseClient, error) {
	return database.NewDatabaseClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *AdbServiceManager) getOCIClient() (DatabaseClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getDbClient(c.Provider)
}

func (c *AdbServiceManager) CreateAdb(ctx context.Context, adb ociv1beta1.AutonomousDatabases, adminPwd string) (database.CreateAutonomousDatabaseResponse, error) {
	dbClient, err := c.getOCIClient()
	if err != nil {
		return database.CreateAutonomousDatabaseResponse{}, err
	}

	c.Log.DebugLog("Creating Autonomous Database ", "name", adb.Spec.DisplayName)

	createAutonomousDatabaseDetails := database.CreateAutonomousDatabaseDetails{
		CompartmentId:        common.String(string(adb.Spec.CompartmentId)),
		DisplayName:          common.String(adb.Spec.DisplayName),
		DbName:               common.String(adb.Spec.DbName),
		DataStorageSizeInTBs: common.Int(adb.Spec.DataStorageSizeInTBs),
		AdminPassword:        common.String(adminPwd),
		IsDedicated:          common.Bool(adb.Spec.IsDedicated),
		DbWorkload:           database.CreateAutonomousDatabaseBaseDbWorkloadEnum(adb.Spec.DbWorkload),
		FreeformTags:         adb.Spec.FreeFormTags,
		DefinedTags:          *util.ConvertToOciDefinedTags(&adb.Spec.DefinedTags),
	}

	if adb.Spec.HasExplicitIsAutoScalingEnabled() {
		createAutonomousDatabaseDetails.IsAutoScalingEnabled = common.Bool(adb.Spec.IsAutoScalingEnabled)
	}
	if adb.Spec.HasExplicitIsFreeTier() {
		createAutonomousDatabaseDetails.IsFreeTier = common.Bool(adb.Spec.IsFreeTier)
	}

	if adb.Spec.ComputeModel != "" {
		createAutonomousDatabaseDetails.ComputeModel = database.CreateAutonomousDatabaseBaseComputeModelEnum(adb.Spec.ComputeModel)
		createAutonomousDatabaseDetails.ComputeCount = common.Float32(adb.Spec.ComputeCount)
	} else {
		createAutonomousDatabaseDetails.CpuCoreCount = common.Int(adb.Spec.CpuCoreCount)
	}

	if adb.Spec.DbVersion != "" {
		createAutonomousDatabaseDetails.DbVersion = common.String(adb.Spec.DbVersion)
	}

	if adb.Spec.LicenseModel != "" {
		createAutonomousDatabaseDetails.LicenseModel = database.CreateAutonomousDatabaseBaseLicenseModelEnum(adb.Spec.LicenseModel)
	}

	createAutonomousDatabaseRequest := database.CreateAutonomousDatabaseRequest{
		CreateAutonomousDatabaseDetails: createAutonomousDatabaseDetails,
	}

	return dbClient.CreateAutonomousDatabase(ctx, createAutonomousDatabaseRequest)
}

func (c *AdbServiceManager) GetAdbOcid(ctx context.Context, adb ociv1beta1.AutonomousDatabases) (*ociv1beta1.OCID, error) {
	dbClient, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	// List ADBs based on compartmentId and displayName and lifecycle-state as Active
	listAdbRequest := database.ListAutonomousDatabasesRequest{
		CompartmentId: common.String(string(adb.Spec.CompartmentId)),
		DisplayName:   common.String(adb.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	listAdbResponse, err := dbClient.ListAutonomousDatabases(ctx, listAdbRequest)
	if err != nil {
		c.Log.ErrorLog(err, "Error while listing Autonomous Database")
		return nil, err
	}

	if len(listAdbResponse.Items) > 0 {
		status := listAdbResponse.Items[0].LifecycleState
		if status == database.AutonomousDatabaseSummaryLifecycleStateAvailable ||
			status == database.AutonomousDatabaseSummaryLifecycleStateAvailableNeedsAttention ||
			status == database.AutonomousDatabaseSummaryLifecycleStateProvisioning ||
			status == database.AutonomousDatabaseSummaryLifecycleStateUpdating {

			c.Log.DebugLog(fmt.Sprintf("Autonomous Database %s exists.", adb.Spec.DisplayName))

			return (*ociv1beta1.OCID)(listAdbResponse.Items[0].Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("Autonomous Database %s does not exist.", adb.Spec.DisplayName))
	return nil, nil
}

func (c *AdbServiceManager) DeleteAdb(ctx context.Context, adbId ociv1beta1.OCID) error {
	_, err := c.submitDeleteAdb(ctx, adbId)
	return err
}

func (c *AdbServiceManager) submitDeleteAdb(ctx context.Context, adbId ociv1beta1.OCID) (*string, error) {
	dbClient, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := database.DeleteAutonomousDatabaseRequest{
		AutonomousDatabaseId: common.String(string(adbId)),
	}

	resp, err := dbClient.DeleteAutonomousDatabase(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.OpcWorkRequestId, nil
}

// Sync the Autonomous Database details
func (c *AdbServiceManager) GetAdb(ctx context.Context, adbId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*database.AutonomousDatabase, error) {
	dbClient, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	getAutonomousDatabaseRequest := database.GetAutonomousDatabaseRequest{
		AutonomousDatabaseId: common.String(string(adbId)),
	}

	if retryPolicy != nil {
		getAutonomousDatabaseRequest.RequestMetadata.RetryPolicy = retryPolicy
	}

	response, err := dbClient.GetAutonomousDatabase(ctx, getAutonomousDatabaseRequest)
	if err != nil {
		return nil, err
	}

	return &response.AutonomousDatabase, nil
}

func (c *AdbServiceManager) UpdateAdb(ctx context.Context, adb *ociv1beta1.AutonomousDatabases) error {
	dbClient, err := c.getOCIClient()
	if err != nil {
		return err
	}

	targetID, err := servicemanager.ResolveResourceID(adb.Status.OsokStatus.Ocid, adb.Spec.AdbId)
	if err != nil {
		return err
	}

	existingAdb, err := c.GetAdb(ctx, targetID, nil)
	if err != nil {
		return err
	}

	if adb.Spec.DbName != "" && adb.Spec.DbName != *existingAdb.DbName {
		return fmt.Errorf("dbName cannot be updated in place")
	}

	if err = c.moveAdbCompartmentIfNeeded(ctx, dbClient, adb, existingAdb, targetID); err != nil {
		return err
	}

	updateAutonomousDatabaseDetails, updateNeeded := buildUpdateAutonomousDatabaseDetails(adb, existingAdb)
	if updateNeeded, err = c.applyAdbPasswordUpdate(ctx, adb, &updateAutonomousDatabaseDetails, updateNeeded); err != nil {
		return err
	}
	if updateNeeded {
		updateAutonomousDatabaseRequest := database.UpdateAutonomousDatabaseRequest{
			AutonomousDatabaseId:            common.String(string(targetID)),
			UpdateAutonomousDatabaseDetails: updateAutonomousDatabaseDetails,
		}

		if _, err := dbClient.UpdateAutonomousDatabase(ctx, updateAutonomousDatabaseRequest); err != nil {
			return err
		}
	}

	return nil
}

func (c *AdbServiceManager) moveAdbCompartmentIfNeeded(ctx context.Context, dbClient DatabaseClientInterface,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase, targetID ociv1beta1.OCID) error {
	if adb.Spec.CompartmentId == "" || (existingAdb.CompartmentId != nil && *existingAdb.CompartmentId == string(adb.Spec.CompartmentId)) {
		return nil
	}

	_, err := dbClient.ChangeAutonomousDatabaseCompartment(ctx, database.ChangeAutonomousDatabaseCompartmentRequest{
		AutonomousDatabaseId: common.String(string(targetID)),
		ChangeCompartmentDetails: database.ChangeCompartmentDetails{
			CompartmentId: common.String(string(adb.Spec.CompartmentId)),
		},
	})
	return err
}

func (c *AdbServiceManager) applyAdbPasswordUpdate(ctx context.Context, adb *ociv1beta1.AutonomousDatabases,
	updateDetails *database.UpdateAutonomousDatabaseDetails, updateNeeded bool) (bool, error) {
	if adb.Spec.AdminPassword.Secret.SecretName == "" {
		return updateNeeded, nil
	}

	password, err := c.getAdminPassword(ctx, adb, adb.Namespace)
	if err != nil {
		return false, err
	}
	updateDetails.AdminPassword = common.String(password)
	return true, nil
}

func buildUpdateAutonomousDatabaseDetails(adb *ociv1beta1.AutonomousDatabases,
	existingAdb *database.AutonomousDatabase) (database.UpdateAutonomousDatabaseDetails, bool) {
	updateAutonomousDatabaseDetails := database.UpdateAutonomousDatabaseDetails{}

	updateNeeded := applyAdbIdentityUpdates(&updateAutonomousDatabaseDetails, adb, existingAdb)
	updateNeeded = applyAdbCapacityUpdates(&updateAutonomousDatabaseDetails, adb, existingAdb) || updateNeeded
	updateNeeded = applyAdbOptionalBoolUpdates(&updateAutonomousDatabaseDetails, adb, existingAdb) || updateNeeded
	updateNeeded = applyAdbTagUpdates(&updateAutonomousDatabaseDetails, adb, existingAdb) || updateNeeded

	return updateAutonomousDatabaseDetails, updateNeeded
}

func applyAdbIdentityUpdates(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	updateNeeded := applyAdbDisplayNameUpdate(updateDetails, adb, existingAdb)
	updateNeeded = applyAdbDbWorkloadUpdate(updateDetails, adb, existingAdb) || updateNeeded
	updateNeeded = applyAdbDbVersionUpdate(updateDetails, adb, existingAdb) || updateNeeded
	updateNeeded = applyAdbLicenseModelUpdate(updateDetails, adb, existingAdb) || updateNeeded
	updateNeeded = applyAdbComputeModelAndCountUpdate(updateDetails, adb, existingAdb) || updateNeeded
	return updateNeeded
}

func applyAdbCapacityUpdates(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	updateNeeded := false

	if adb.Spec.DataStorageSizeInTBs != 0 && adb.Spec.DataStorageSizeInTBs != *existingAdb.DataStorageSizeInTBs {
		updateDetails.DataStorageSizeInTBs = common.Int(adb.Spec.DataStorageSizeInTBs)
		updateNeeded = true
	}
	if adb.Spec.CpuCoreCount != 0 && adb.Spec.CpuCoreCount != *existingAdb.CpuCoreCount {
		updateDetails.CpuCoreCount = common.Int(adb.Spec.CpuCoreCount)
		updateNeeded = true
	}

	return updateNeeded
}

func applyAdbComputeModelAndCountUpdate(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	updateNeeded := false

	if adb.Spec.ComputeModel != "" && string(existingAdb.ComputeModel) != adb.Spec.ComputeModel {
		updateDetails.ComputeModel = database.UpdateAutonomousDatabaseDetailsComputeModelEnum(adb.Spec.ComputeModel)
		updateNeeded = true
	}
	if adb.Spec.ComputeModel != "" && existingAdb.ComputeCount != nil && adb.Spec.ComputeCount != *existingAdb.ComputeCount {
		updateDetails.ComputeCount = common.Float32(adb.Spec.ComputeCount)
		updateNeeded = true
	}

	return updateNeeded
}

func applyAdbOptionalBoolUpdates(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	updateNeeded := false

	if shouldUpdateOptionalBool(adb.Spec.HasExplicitIsAutoScalingEnabled(), adb.Spec.IsAutoScalingEnabled, existingAdb.IsAutoScalingEnabled) {
		updateDetails.IsAutoScalingEnabled = common.Bool(adb.Spec.IsAutoScalingEnabled)
		updateNeeded = true
	}
	if shouldUpdateOptionalBool(adb.Spec.HasExplicitIsFreeTier(), adb.Spec.IsFreeTier, existingAdb.IsFreeTier) {
		updateDetails.IsFreeTier = common.Bool(adb.Spec.IsFreeTier)
		updateNeeded = true
	}

	return updateNeeded
}

func applyAdbTagUpdates(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	updateNeeded := false

	if adb.Spec.FreeFormTags != nil && !reflect.DeepEqual(existingAdb.FreeformTags, adb.Spec.FreeFormTags) {
		updateDetails.FreeformTags = adb.Spec.FreeFormTags
		updateNeeded = true
	}
	if adb.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&adb.Spec.DefinedTags); !reflect.DeepEqual(existingAdb.DefinedTags, defTag) {
			updateDetails.DefinedTags = defTag
			updateNeeded = true
		}
	}

	return updateNeeded
}

func applyAdbDisplayNameUpdate(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	if adb.Spec.DisplayName == "" || *existingAdb.DisplayName == adb.Spec.DisplayName {
		return false
	}

	updateDetails.DisplayName = common.String(adb.Spec.DisplayName)
	return true
}

func applyAdbDbWorkloadUpdate(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	if adb.Spec.DbWorkload == "" || string(existingAdb.DbWorkload) == adb.Spec.DbWorkload {
		return false
	}

	updateDetails.DbWorkload = database.UpdateAutonomousDatabaseDetailsDbWorkloadEnum(adb.Spec.DbWorkload)
	return true
}

func applyAdbDbVersionUpdate(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	if adb.Spec.DbVersion == "" || adb.Spec.DbVersion == *existingAdb.DbVersion {
		return false
	}

	updateDetails.DbVersion = common.String(adb.Spec.DbVersion)
	return true
}

func applyAdbLicenseModelUpdate(updateDetails *database.UpdateAutonomousDatabaseDetails,
	adb *ociv1beta1.AutonomousDatabases, existingAdb *database.AutonomousDatabase) bool {
	if adb.Spec.LicenseModel == "" || string(existingAdb.LicenseModel) == adb.Spec.LicenseModel {
		return false
	}

	updateDetails.LicenseModel = database.UpdateAutonomousDatabaseDetailsLicenseModelEnum(adb.Spec.LicenseModel)
	return true
}
