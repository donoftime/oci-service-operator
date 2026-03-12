/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package postgresql

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/psql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// PostgresClientInterface defines the OCI operations used by PostgresDbSystemServiceManager.
type PostgresClientInterface interface {
	CreateDbSystem(ctx context.Context, request psql.CreateDbSystemRequest) (psql.CreateDbSystemResponse, error)
	GetDbSystem(ctx context.Context, request psql.GetDbSystemRequest) (psql.GetDbSystemResponse, error)
	ListDbSystems(ctx context.Context, request psql.ListDbSystemsRequest) (psql.ListDbSystemsResponse, error)
	ChangeDbSystemCompartment(ctx context.Context, request psql.ChangeDbSystemCompartmentRequest) (psql.ChangeDbSystemCompartmentResponse, error)
	UpdateDbSystem(ctx context.Context, request psql.UpdateDbSystemRequest) (psql.UpdateDbSystemResponse, error)
	DeleteDbSystem(ctx context.Context, request psql.DeleteDbSystemRequest) (psql.DeleteDbSystemResponse, error)
}

func getPostgresClient(provider common.ConfigurationProvider) (psql.PostgresqlClient, error) {
	return psql.NewPostgresqlClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *PostgresDbSystemServiceManager) getOCIClient() (PostgresClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getPostgresClient(c.Provider)
}

// CreatePostgresDbSystem calls the OCI API to create a new PostgreSQL DB system.
func (c *PostgresDbSystemServiceManager) CreatePostgresDbSystem(ctx context.Context, dbSystem ociv1beta1.PostgresDbSystem) (psql.CreateDbSystemResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return psql.CreateDbSystemResponse{}, err
	}

	c.Log.DebugLog("Creating PostgresDbSystem", "name", dbSystem.Spec.DisplayName)

	storageDetails := buildStorageDetails()

	details := psql.CreateDbSystemDetails{
		DisplayName:   common.String(dbSystem.Spec.DisplayName),
		CompartmentId: common.String(string(dbSystem.Spec.CompartmentId)),
		DbVersion:     common.String(dbSystem.Spec.DbVersion),
		Shape:         common.String(dbSystem.Spec.Shape),
		NetworkDetails: &psql.NetworkDetails{
			SubnetId: common.String(string(dbSystem.Spec.SubnetId)),
		},
		StorageDetails: storageDetails,
	}

	applyPostgresTextFields(&details, dbSystem)
	applyPostgresCapacityFields(&details, dbSystem)
	applyPostgresTagFields(&details, dbSystem)

	if dbSystem.Spec.AdminUsername.Secret.SecretName != "" && dbSystem.Spec.AdminPassword.Secret.SecretName != "" {
		credentials, err := c.loadDbSystemCredentials(ctx, dbSystem)
		if err != nil {
			return psql.CreateDbSystemResponse{}, err
		}
		details.Credentials = credentials
	}

	req := psql.CreateDbSystemRequest{
		CreateDbSystemDetails: details,
	}

	return client.CreateDbSystem(ctx, req)
}

func applyPostgresTextFields(details *psql.CreateDbSystemDetails, dbSystem ociv1beta1.PostgresDbSystem) {
	if dbSystem.Spec.Description != "" {
		details.Description = common.String(dbSystem.Spec.Description)
	}
}

func applyPostgresCapacityFields(details *psql.CreateDbSystemDetails, dbSystem ociv1beta1.PostgresDbSystem) {
	if dbSystem.Spec.InstanceCount > 0 {
		details.InstanceCount = common.Int(dbSystem.Spec.InstanceCount)
	}
	if dbSystem.Spec.InstanceOcpuCount > 0 {
		details.InstanceOcpuCount = common.Int(dbSystem.Spec.InstanceOcpuCount)
	}
	if dbSystem.Spec.InstanceMemoryInGBs > 0 {
		details.InstanceMemorySizeInGBs = common.Int(dbSystem.Spec.InstanceMemoryInGBs)
	}
}

func applyPostgresTagFields(details *psql.CreateDbSystemDetails, dbSystem ociv1beta1.PostgresDbSystem) {
	if dbSystem.Spec.FreeFormTags != nil {
		details.FreeformTags = dbSystem.Spec.FreeFormTags
	}
	if dbSystem.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&dbSystem.Spec.DefinedTags)
	}
}

func (c *PostgresDbSystemServiceManager) loadDbSystemCredentials(ctx context.Context,
	dbSystem ociv1beta1.PostgresDbSystem) (*psql.Credentials, error) {
	c.Log.DebugLog("Getting Admin Username from Secret")
	unameMap, err := c.CredentialClient.GetSecret(ctx, dbSystem.Spec.AdminUsername.Secret.SecretName, dbSystem.Namespace)
	if err != nil {
		return nil, err
	}
	uname, ok := unameMap["username"]
	if !ok {
		return nil, errors.New("username key in admin secret is not found")
	}

	c.Log.DebugLog("Getting Admin Password from Secret")
	pwdMap, err := c.CredentialClient.GetSecret(ctx, dbSystem.Spec.AdminPassword.Secret.SecretName, dbSystem.Namespace)
	if err != nil {
		return nil, err
	}
	pwd, ok := pwdMap["password"]
	if !ok {
		return nil, errors.New("password key in admin secret is not found")
	}

	credentials := psql.Credentials{
		Username: common.String(string(uname)),
		PasswordDetails: psql.PlainTextPasswordDetails{
			Password: common.String(string(pwd)),
		},
	}
	return &credentials, nil
}

// GetPostgresDbSystem retrieves a PostgreSQL DB system by OCID.
func (c *PostgresDbSystemServiceManager) GetPostgresDbSystem(ctx context.Context, dbSystemId ociv1beta1.OCID) (*psql.DbSystem, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := psql.GetDbSystemRequest{
		DbSystemId: common.String(string(dbSystemId)),
	}

	resp, err := client.GetDbSystem(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.DbSystem, nil
}

// GetPostgresDbSystemByName looks up an existing DB system by display name and returns its OCID if found.
func (c *PostgresDbSystemServiceManager) GetPostgresDbSystemByName(ctx context.Context, dbSystem ociv1beta1.PostgresDbSystem) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := psql.ListDbSystemsRequest{
		CompartmentId: common.String(string(dbSystem.Spec.CompartmentId)),
		DisplayName:   common.String(dbSystem.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListDbSystems(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing PostgreSQL DB systems")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("PostgresDbSystem %s exists with OCID %s", dbSystem.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("PostgresDbSystem %s does not exist", dbSystem.Spec.DisplayName))
	return nil, nil
}

// UpdatePostgresDbSystem updates an existing PostgreSQL DB system.
func (c *PostgresDbSystemServiceManager) UpdatePostgresDbSystem(ctx context.Context, dbSystem *ociv1beta1.PostgresDbSystem) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	targetID, err := resolveDbSystemID(dbSystem.Status.OsokStatus.Ocid, dbSystem.Spec.PostgresDbSystemId)
	if err != nil {
		return err
	}

	existing, err := c.GetPostgresDbSystem(ctx, targetID)
	if err != nil {
		return err
	}

	if err := validatePostgresUnsupportedChanges(dbSystem, existing); err != nil {
		return err
	}

	if err := movePostgresDbSystemCompartmentIfNeeded(ctx, client, dbSystem, existing, targetID); err != nil {
		return err
	}

	updateDetails, updateNeeded := buildPostgresDbSystemUpdateDetails(dbSystem, existing)
	if !updateNeeded {
		return nil
	}

	req := psql.UpdateDbSystemRequest{
		DbSystemId:            common.String(string(targetID)),
		UpdateDbSystemDetails: updateDetails,
	}

	_, err = client.UpdateDbSystem(ctx, req)
	return err
}

func movePostgresDbSystemCompartmentIfNeeded(ctx context.Context, client PostgresClientInterface,
	dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem, targetID ociv1beta1.OCID) error {
	if dbSystem.Spec.CompartmentId == "" || (existing.CompartmentId != nil && *existing.CompartmentId == string(dbSystem.Spec.CompartmentId)) {
		return nil
	}

	_, err := client.ChangeDbSystemCompartment(ctx, psql.ChangeDbSystemCompartmentRequest{
		DbSystemId: common.String(string(targetID)),
		ChangeDbSystemCompartmentDetails: psql.ChangeDbSystemCompartmentDetails{
			CompartmentId: common.String(string(dbSystem.Spec.CompartmentId)),
		},
	})
	return err
}

func buildPostgresDbSystemUpdateDetails(dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) (psql.UpdateDbSystemDetails, bool) {
	updateDetails := psql.UpdateDbSystemDetails{}
	updateNeeded := applyPostgresDisplayNameUpdate(&updateDetails, dbSystem, existing)
	if applyPostgresDescriptionUpdate(&updateDetails, dbSystem, existing) {
		updateNeeded = true
	}
	if applyPostgresFreeformTagUpdate(&updateDetails, dbSystem, existing) {
		updateNeeded = true
	}
	if applyPostgresDefinedTagUpdate(&updateDetails, dbSystem, existing) {
		updateNeeded = true
	}

	return updateDetails, updateNeeded
}

func applyPostgresDisplayNameUpdate(updateDetails *psql.UpdateDbSystemDetails, dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) bool {
	if dbSystem.Spec.DisplayName == "" || *existing.DisplayName == dbSystem.Spec.DisplayName {
		return false
	}
	updateDetails.DisplayName = common.String(dbSystem.Spec.DisplayName)
	return true
}

func applyPostgresDescriptionUpdate(updateDetails *psql.UpdateDbSystemDetails, dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) bool {
	if dbSystem.Spec.Description == "" || (existing.Description != nil && *existing.Description == dbSystem.Spec.Description) {
		return false
	}
	updateDetails.Description = common.String(dbSystem.Spec.Description)
	return true
}

func applyPostgresFreeformTagUpdate(updateDetails *psql.UpdateDbSystemDetails, dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) bool {
	if dbSystem.Spec.FreeFormTags == nil || reflect.DeepEqual(existing.FreeformTags, dbSystem.Spec.FreeFormTags) {
		return false
	}
	updateDetails.FreeformTags = dbSystem.Spec.FreeFormTags
	return true
}

func applyPostgresDefinedTagUpdate(updateDetails *psql.UpdateDbSystemDetails, dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) bool {
	if dbSystem.Spec.DefinedTags == nil {
		return false
	}
	desiredDefinedTags := *util.ConvertToOciDefinedTags(&dbSystem.Spec.DefinedTags)
	if reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
		return false
	}
	updateDetails.DefinedTags = desiredDefinedTags
	return true
}

func validatePostgresUnsupportedChanges(dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) error {
	validators := []func(*ociv1beta1.PostgresDbSystem, *psql.DbSystem) error{
		validatePostgresDbVersionChange,
		validatePostgresShapeChange,
		validatePostgresInstanceCountChange,
		validatePostgresInstanceOcpuCountChange,
		validatePostgresInstanceMemoryChange,
		validatePostgresSubnetChange,
	}

	for _, validate := range validators {
		if err := validate(dbSystem, existing); err != nil {
			return err
		}
	}

	return nil
}

func validatePostgresDbVersionChange(dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) error {
	if dbSystem.Spec.DbVersion == "" || safeString(existing.DbVersion) == dbSystem.Spec.DbVersion {
		return nil
	}
	return fmt.Errorf("dbVersion cannot be updated in place")
}

func validatePostgresShapeChange(dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) error {
	if dbSystem.Spec.Shape == "" || safeString(existing.Shape) == dbSystem.Spec.Shape {
		return nil
	}
	return fmt.Errorf("shape cannot be updated in place")
}

func validatePostgresInstanceCountChange(dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) error {
	if dbSystem.Spec.InstanceCount == 0 || existing.InstanceCount == nil || *existing.InstanceCount == dbSystem.Spec.InstanceCount {
		return nil
	}
	return fmt.Errorf("instanceCount cannot be updated in place")
}

func validatePostgresInstanceOcpuCountChange(dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) error {
	if dbSystem.Spec.InstanceOcpuCount == 0 ||
		(existing.InstanceOcpuCount != nil && *existing.InstanceOcpuCount == dbSystem.Spec.InstanceOcpuCount) {
		return nil
	}
	return fmt.Errorf("instanceOcpuCount cannot be updated in place")
}

func validatePostgresInstanceMemoryChange(dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) error {
	if dbSystem.Spec.InstanceMemoryInGBs == 0 ||
		(existing.InstanceMemorySizeInGBs != nil && *existing.InstanceMemorySizeInGBs == dbSystem.Spec.InstanceMemoryInGBs) {
		return nil
	}
	return fmt.Errorf("instanceMemoryInGBs cannot be updated in place")
}

func validatePostgresSubnetChange(dbSystem *ociv1beta1.PostgresDbSystem, existing *psql.DbSystem) error {
	if dbSystem.Spec.SubnetId == "" ||
		existing.NetworkDetails == nil ||
		existing.NetworkDetails.SubnetId == nil ||
		*existing.NetworkDetails.SubnetId == string(dbSystem.Spec.SubnetId) {
		return nil
	}
	return fmt.Errorf("subnetId cannot be updated in place")
}

// DeletePostgresDbSystem deletes the PostgreSQL DB system for the given OCID.
func (c *PostgresDbSystemServiceManager) DeletePostgresDbSystem(ctx context.Context, dbSystemId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := psql.DeleteDbSystemRequest{
		DbSystemId: common.String(string(dbSystemId)),
	}

	_, err = client.DeleteDbSystem(ctx, req)
	return err
}

// buildStorageDetails constructs the StorageDetails for the OCI PostgreSQL DB system.
func buildStorageDetails() psql.StorageDetails {
	isRegionallyDurable := true
	return psql.OciOptimizedStorageDetails{
		IsRegionallyDurable: common.Bool(isRegionallyDurable),
	}
}
