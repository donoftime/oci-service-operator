/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// KmsVaultClientInterface defines the OCI vault operations used by OciVaultServiceManager.
type KmsVaultClientInterface interface {
	CreateVault(ctx context.Context, request keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error)
	GetVault(ctx context.Context, request keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error)
	ListVaults(ctx context.Context, request keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error)
	UpdateVault(ctx context.Context, request keymanagement.UpdateVaultRequest) (keymanagement.UpdateVaultResponse, error)
	ScheduleVaultDeletion(ctx context.Context, request keymanagement.ScheduleVaultDeletionRequest) (keymanagement.ScheduleVaultDeletionResponse, error)
}

// KmsManagementClientInterface defines the OCI key management operations used by OciVaultServiceManager.
type KmsManagementClientInterface interface {
	CreateKey(ctx context.Context, request keymanagement.CreateKeyRequest) (keymanagement.CreateKeyResponse, error)
	GetKey(ctx context.Context, request keymanagement.GetKeyRequest) (keymanagement.GetKeyResponse, error)
	ListKeys(ctx context.Context, request keymanagement.ListKeysRequest) (keymanagement.ListKeysResponse, error)
	ScheduleKeyDeletion(ctx context.Context, request keymanagement.ScheduleKeyDeletionRequest) (keymanagement.ScheduleKeyDeletionResponse, error)
}

func getKmsVaultClient(provider common.ConfigurationProvider) (keymanagement.KmsVaultClient, error) {
	return keymanagement.NewKmsVaultClientWithConfigurationProvider(provider)
}

func getKmsManagementClient(provider common.ConfigurationProvider, managementEndpoint string) (keymanagement.KmsManagementClient, error) {
	return keymanagement.NewKmsManagementClientWithConfigurationProvider(provider, managementEndpoint)
}

// getVaultClient returns the injected vault client if set, otherwise creates one from the provider.
func (c *OciVaultServiceManager) getVaultClient() (KmsVaultClientInterface, error) {
	if c.ociVaultClient != nil {
		return c.ociVaultClient, nil
	}
	return getKmsVaultClient(c.Provider)
}

// getMgmtClient returns the injected management client if set, otherwise creates one from the provider.
func (c *OciVaultServiceManager) getMgmtClient(managementEndpoint string) (KmsManagementClientInterface, error) {
	if c.ociManagementClient != nil {
		return c.ociManagementClient, nil
	}
	return getKmsManagementClient(c.Provider, managementEndpoint)
}

// CreateVault calls the OCI API to create a new Vault.
func (c *OciVaultServiceManager) CreateVault(ctx context.Context, v ociv1beta1.OciVault) (*keymanagement.Vault, error) {
	client, err := c.getVaultClient()
	if err != nil {
		return nil, err
	}

	c.Log.DebugLog("Creating OciVault", "name", v.Spec.DisplayName)

	details := keymanagement.CreateVaultDetails{
		CompartmentId: common.String(string(v.Spec.CompartmentId)),
		DisplayName:   common.String(v.Spec.DisplayName),
		VaultType:     keymanagement.CreateVaultDetailsVaultTypeEnum(v.Spec.VaultType),
		FreeformTags:  v.Spec.FreeFormTags,
	}

	if v.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&v.Spec.DefinedTags)
	}

	req := keymanagement.CreateVaultRequest{
		CreateVaultDetails: details,
	}

	resp, err := client.CreateVault(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Vault, nil
}

// GetVault retrieves a Vault by OCID.
func (c *OciVaultServiceManager) GetVault(ctx context.Context, vaultId ociv1beta1.OCID) (*keymanagement.Vault, error) {
	client, err := c.getVaultClient()
	if err != nil {
		return nil, err
	}

	req := keymanagement.GetVaultRequest{
		VaultId: common.String(string(vaultId)),
	}

	resp, err := client.GetVault(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Vault, nil
}

// GetVaultOcid looks up an existing Vault by display name and returns its OCID if found.
// Returns nil if no matching vault in CREATING or ACTIVE state is found.
func (c *OciVaultServiceManager) GetVaultOcid(ctx context.Context, v ociv1beta1.OciVault) (*ociv1beta1.OCID, error) {
	client, err := c.getVaultClient()
	if err != nil {
		return nil, err
	}

	req := keymanagement.ListVaultsRequest{
		CompartmentId: common.String(string(v.Spec.CompartmentId)),
		Limit:         common.Int(100),
	}

	resp, err := client.ListVaults(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Vaults")
		return nil, err
	}

	for _, item := range resp.Items {
		if item.DisplayName == nil || *item.DisplayName != v.Spec.DisplayName {
			continue
		}
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("OciVault %s exists with OCID %s", v.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciVault %s does not exist", v.Spec.DisplayName))
	return nil, nil
}

// UpdateVault updates the display name and tags of an existing Vault.
func (c *OciVaultServiceManager) UpdateVault(ctx context.Context, v *ociv1beta1.OciVault) error {
	client, err := c.getVaultClient()
	if err != nil {
		return err
	}

	updateDetails := keymanagement.UpdateVaultDetails{
		DisplayName: common.String(v.Spec.DisplayName),
	}
	if v.Spec.FreeFormTags != nil {
		updateDetails.FreeformTags = v.Spec.FreeFormTags
	}
	if v.Spec.DefinedTags != nil {
		updateDetails.DefinedTags = *util.ConvertToOciDefinedTags(&v.Spec.DefinedTags)
	}

	req := keymanagement.UpdateVaultRequest{
		VaultId:            common.String(string(v.Status.OsokStatus.Ocid)),
		UpdateVaultDetails: updateDetails,
	}

	_, err = client.UpdateVault(ctx, req)
	return err
}

// ScheduleVaultDeletion schedules the Vault for deletion (minimum 7-day grace period).
func (c *OciVaultServiceManager) ScheduleVaultDeletion(ctx context.Context, vaultId ociv1beta1.OCID) error {
	client, err := c.getVaultClient()
	if err != nil {
		return err
	}

	req := keymanagement.ScheduleVaultDeletionRequest{
		VaultId:                      common.String(string(vaultId)),
		ScheduleVaultDeletionDetails: keymanagement.ScheduleVaultDeletionDetails{},
	}
	_, err = client.ScheduleVaultDeletion(ctx, req)
	return err
}

// CreateKey creates a new key in the vault using the vault's management endpoint.
func (c *OciVaultServiceManager) CreateKey(ctx context.Context, v ociv1beta1.OciVault, managementEndpoint string) (*keymanagement.Key, error) {
	if v.Spec.Key == nil {
		return nil, fmt.Errorf("no key spec provided")
	}

	client, err := c.getMgmtClient(managementEndpoint)
	if err != nil {
		return nil, err
	}

	keyShape := &keymanagement.KeyShape{
		Algorithm: keymanagement.KeyShapeAlgorithmEnum(v.Spec.Key.KeyShape.Algorithm),
		Length:    common.Int(v.Spec.Key.KeyShape.Length),
	}

	details := keymanagement.CreateKeyDetails{
		CompartmentId: common.String(string(v.Spec.CompartmentId)),
		DisplayName:   common.String(v.Spec.Key.DisplayName),
		KeyShape:      keyShape,
		FreeformTags:  v.Spec.FreeFormTags,
	}

	if v.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&v.Spec.DefinedTags)
	}

	req := keymanagement.CreateKeyRequest{
		CreateKeyDetails: details,
	}

	resp, err := client.CreateKey(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Key, nil
}

// GetKey retrieves a key by OCID using the vault's management endpoint.
func (c *OciVaultServiceManager) GetKey(ctx context.Context, keyId ociv1beta1.OCID, managementEndpoint string) (*keymanagement.Key, error) {
	client, err := c.getMgmtClient(managementEndpoint)
	if err != nil {
		return nil, err
	}

	req := keymanagement.GetKeyRequest{
		KeyId: common.String(string(keyId)),
	}

	resp, err := client.GetKey(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Key, nil
}

// GetKeyOcid looks up an existing key by display name within a vault.
func (c *OciVaultServiceManager) GetKeyOcid(ctx context.Context, v ociv1beta1.OciVault, managementEndpoint string) (*ociv1beta1.OCID, error) {
	if v.Spec.Key == nil {
		return nil, nil
	}

	client, err := c.getMgmtClient(managementEndpoint)
	if err != nil {
		return nil, err
	}

	req := keymanagement.ListKeysRequest{
		CompartmentId: common.String(string(v.Spec.CompartmentId)),
		Limit:         common.Int(100),
	}

	resp, err := client.ListKeys(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Keys")
		return nil, err
	}

	for _, item := range resp.Items {
		if item.DisplayName == nil || *item.DisplayName != v.Spec.Key.DisplayName {
			continue
		}
		state := string(item.LifecycleState)
		if state == "ENABLED" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("Key %s exists with OCID %s", v.Spec.Key.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("Key %s does not exist", v.Spec.Key.DisplayName))
	return nil, nil
}

// ScheduleKeyDeletion schedules a key for deletion.
func (c *OciVaultServiceManager) ScheduleKeyDeletion(ctx context.Context, keyId ociv1beta1.OCID, managementEndpoint string) error {
	client, err := c.getMgmtClient(managementEndpoint)
	if err != nil {
		return err
	}

	req := keymanagement.ScheduleKeyDeletionRequest{
		KeyId:                      common.String(string(keyId)),
		ScheduleKeyDeletionDetails: keymanagement.ScheduleKeyDeletionDetails{},
	}

	_, err = client.ScheduleKeyDeletion(ctx, req)
	return err
}
