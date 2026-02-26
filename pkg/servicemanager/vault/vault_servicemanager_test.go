/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/vault"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct {
	createSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	deleteSecretFn func(ctx context.Context, name, ns string) (bool, error)
	getSecretFn    func(ctx context.Context, name, ns string) (map[string][]byte, error)
	updateSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, ns)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, ns)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

// fakeVaultClient implements KmsVaultClientInterface for testing.
type fakeVaultClient struct {
	createVaultFn         func(ctx context.Context, req keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error)
	getVaultFn            func(ctx context.Context, req keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error)
	listVaultsFn          func(ctx context.Context, req keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error)
	updateVaultFn         func(ctx context.Context, req keymanagement.UpdateVaultRequest) (keymanagement.UpdateVaultResponse, error)
	scheduleVaultDeleteFn func(ctx context.Context, req keymanagement.ScheduleVaultDeletionRequest) (keymanagement.ScheduleVaultDeletionResponse, error)
}

func (f *fakeVaultClient) CreateVault(ctx context.Context, req keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
	if f.createVaultFn != nil {
		return f.createVaultFn(ctx, req)
	}
	return keymanagement.CreateVaultResponse{}, nil
}

func (f *fakeVaultClient) GetVault(ctx context.Context, req keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error) {
	if f.getVaultFn != nil {
		return f.getVaultFn(ctx, req)
	}
	return keymanagement.GetVaultResponse{}, nil
}

func (f *fakeVaultClient) ListVaults(ctx context.Context, req keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
	if f.listVaultsFn != nil {
		return f.listVaultsFn(ctx, req)
	}
	return keymanagement.ListVaultsResponse{}, nil
}

func (f *fakeVaultClient) UpdateVault(ctx context.Context, req keymanagement.UpdateVaultRequest) (keymanagement.UpdateVaultResponse, error) {
	if f.updateVaultFn != nil {
		return f.updateVaultFn(ctx, req)
	}
	return keymanagement.UpdateVaultResponse{}, nil
}

func (f *fakeVaultClient) ScheduleVaultDeletion(ctx context.Context, req keymanagement.ScheduleVaultDeletionRequest) (keymanagement.ScheduleVaultDeletionResponse, error) {
	if f.scheduleVaultDeleteFn != nil {
		return f.scheduleVaultDeleteFn(ctx, req)
	}
	return keymanagement.ScheduleVaultDeletionResponse{}, nil
}

// fakeManagementClient implements KmsManagementClientInterface for testing.
type fakeManagementClient struct {
	createKeyFn         func(ctx context.Context, req keymanagement.CreateKeyRequest) (keymanagement.CreateKeyResponse, error)
	getKeyFn            func(ctx context.Context, req keymanagement.GetKeyRequest) (keymanagement.GetKeyResponse, error)
	listKeysFn          func(ctx context.Context, req keymanagement.ListKeysRequest) (keymanagement.ListKeysResponse, error)
	scheduleKeyDeleteFn func(ctx context.Context, req keymanagement.ScheduleKeyDeletionRequest) (keymanagement.ScheduleKeyDeletionResponse, error)
}

func (f *fakeManagementClient) CreateKey(ctx context.Context, req keymanagement.CreateKeyRequest) (keymanagement.CreateKeyResponse, error) {
	if f.createKeyFn != nil {
		return f.createKeyFn(ctx, req)
	}
	return keymanagement.CreateKeyResponse{}, nil
}

func (f *fakeManagementClient) GetKey(ctx context.Context, req keymanagement.GetKeyRequest) (keymanagement.GetKeyResponse, error) {
	if f.getKeyFn != nil {
		return f.getKeyFn(ctx, req)
	}
	return keymanagement.GetKeyResponse{}, nil
}

func (f *fakeManagementClient) ListKeys(ctx context.Context, req keymanagement.ListKeysRequest) (keymanagement.ListKeysResponse, error) {
	if f.listKeysFn != nil {
		return f.listKeysFn(ctx, req)
	}
	return keymanagement.ListKeysResponse{}, nil
}

func (f *fakeManagementClient) ScheduleKeyDeletion(ctx context.Context, req keymanagement.ScheduleKeyDeletionRequest) (keymanagement.ScheduleKeyDeletionResponse, error) {
	if f.scheduleKeyDeleteFn != nil {
		return f.scheduleKeyDeleteFn(ctx, req)
	}
	return keymanagement.ScheduleKeyDeletionResponse{}, nil
}

// makeActiveVault returns a Vault in ACTIVE state with all fields set.
func makeActiveVault(id, displayName, managementEndpoint, cryptoEndpoint string) keymanagement.Vault {
	return keymanagement.Vault{
		Id:                 common.String(id),
		DisplayName:        common.String(displayName),
		CompartmentId:      common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState:     keymanagement.VaultLifecycleStateActive,
		ManagementEndpoint: common.String(managementEndpoint),
		CryptoEndpoint:     common.String(cryptoEndpoint),
		VaultType:          keymanagement.VaultVaultTypeDefault,
		WrappingkeyId:      common.String("ocid1.key.oc1..wrapping"),
	}
}

// makeCreatingVault returns a Vault in CREATING state.
func makeCreatingVault(id, displayName string) keymanagement.Vault {
	return keymanagement.Vault{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState: keymanagement.VaultLifecycleStateCreating,
	}
}

// newMgr creates a test service manager with the given credential client.
func newMgr(credClient *fakeCredentialClient) *OciVaultServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return NewOciVaultServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)
}

// newMgrWithVaultClient creates a service manager with an injected vault client.
func newMgrWithVaultClient(credClient *fakeCredentialClient, vc KmsVaultClientInterface) *OciVaultServiceManager {
	mgr := newMgr(credClient)
	ExportSetVaultClientForTest(mgr, vc)
	return mgr
}

// newMgrWithBothClients creates a service manager with injected vault and management clients.
func newMgrWithBothClients(credClient *fakeCredentialClient, vc KmsVaultClientInterface, mc KmsManagementClientInterface) *OciVaultServiceManager {
	mgr := newMgr(credClient)
	ExportSetVaultClientForTest(mgr, vc)
	ExportSetManagementClientForTest(mgr, mc)
	return mgr
}

// newVaultCR builds a minimal OciVault custom resource with no VaultId (create path).
func newVaultCR(name, ns string) *ociv1beta1.OciVault {
	v := &ociv1beta1.OciVault{}
	v.Name = name
	v.Namespace = ns
	v.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	v.Spec.DisplayName = name
	v.Spec.VaultType = "DEFAULT"
	return v
}

// ---------------------------------------------------------------------------
// Credential map tests
// ---------------------------------------------------------------------------

// TestGetCredentialMap verifies the secret credential map is built correctly from a Vault.
func TestGetCredentialMap(t *testing.T) {
	v := makeActiveVault(
		"ocid1.vault.oc1..xxx",
		"test-vault",
		"https://abc-management.kms.us-ashburn-1.oraclecloud.com",
		"https://abc-crypto.kms.us-ashburn-1.oraclecloud.com",
	)
	credMap := GetCredentialMapForTest(v)

	assert.Equal(t, "ocid1.vault.oc1..xxx", string(credMap["id"]))
	assert.Equal(t, "test-vault", string(credMap["displayName"]))
	assert.Equal(t, "https://abc-management.kms.us-ashburn-1.oraclecloud.com", string(credMap["managementEndpoint"]))
	assert.Equal(t, "https://abc-crypto.kms.us-ashburn-1.oraclecloud.com", string(credMap["cryptoEndpoint"]))
}

// TestGetCredentialMap_NilFields verifies nil pointer fields are handled gracefully.
func TestGetCredentialMap_NilFields(t *testing.T) {
	v := keymanagement.Vault{
		Id:             common.String("ocid1.vault.oc1..xxx"),
		LifecycleState: keymanagement.VaultLifecycleStateActive,
	}
	credMap := GetCredentialMapForTest(v)
	assert.NotContains(t, credMap, "managementEndpoint")
	assert.NotContains(t, credMap, "cryptoEndpoint")
	assert.NotContains(t, credMap, "displayName")
}

// TestGetCredentialMap_KeyManagementEndpoint verifies the managementEndpoint key
// is included in the credential map and contains the correct value.
func TestGetCredentialMap_KeyManagementEndpoint(t *testing.T) {
	mgmtEndpoint := "https://abc-management.kms.us-ashburn-1.oraclecloud.com"
	v := makeActiveVault("ocid1.vault.oc1..xxx", "test-vault", mgmtEndpoint,
		"https://abc-crypto.kms.us-ashburn-1.oraclecloud.com")
	credMap := GetCredentialMapForTest(v)

	assert.Contains(t, credMap, "managementEndpoint")
	assert.Equal(t, mgmtEndpoint, string(credMap["managementEndpoint"]))
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := newMgr(credClient)

	v := newVaultCR("test-vault", "default")

	done, err := mgr.Delete(context.Background(), v)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called when OCID is empty")
}

// TestDelete_SecretError verifies Delete tolerates secret-deletion errors.
func TestDelete_SecretError(t *testing.T) {
	credClient := &fakeCredentialClient{
		deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, errors.New("secret not found")
		},
	}
	mgr := newMgr(credClient)

	v := newVaultCR("test-vault", "default")
	v.Status.OsokStatus.Ocid = "ocid1.vault.oc1..xxx"

	// The OCI API call will fail with invalid config, but we exercise the path.
	_, _ = mgr.Delete(context.Background(), v)
}

// TestDelete_WithOcid_SchedulesVaultDeletion verifies that Delete with an OCID calls
// ScheduleVaultDeletion on the OCI API and then removes the secret.
func TestDelete_WithOcid_SchedulesVaultDeletion(t *testing.T) {
	vaultID := "ocid1.vault.oc1..deleteme"

	deleteCalled := false
	fakeVC := &fakeVaultClient{
		getVaultFn: func(_ context.Context, req keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error) {
			v := makeActiveVault(*req.VaultId, "test-vault", "https://mgmt.example.com", "https://crypto.example.com")
			return keymanagement.GetVaultResponse{Vault: v}, nil
		},
		scheduleVaultDeleteFn: func(_ context.Context, req keymanagement.ScheduleVaultDeletionRequest) (keymanagement.ScheduleVaultDeletionResponse, error) {
			deleteCalled = true
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagement.ScheduleVaultDeletionResponse{}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)

	v := newVaultCR("test-vault", "default")
	v.Status.OsokStatus.Ocid = ociv1beta1.OCID(vaultID)

	done, err := mgr.Delete(context.Background(), v)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled, "ScheduleVaultDeletion should have been called")
	assert.True(t, credClient.deleteCalled, "DeleteSecret should have been called")
}

// TestDelete_WithKey_SchedulesKeyDeletion verifies that Delete also schedules key deletion
// when the vault spec includes a key by ID.
func TestDelete_WithKey_SchedulesKeyDeletion(t *testing.T) {
	vaultID := "ocid1.vault.oc1..deleteme"
	keyID := "ocid1.key.oc1..deletekey"
	mgmtEndpoint := "https://mgmt.example.com"

	keyDeleteCalled := false
	fakeVC := &fakeVaultClient{
		getVaultFn: func(_ context.Context, req keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error) {
			v := makeActiveVault(*req.VaultId, "test-vault", mgmtEndpoint, "https://crypto.example.com")
			return keymanagement.GetVaultResponse{Vault: v}, nil
		},
		scheduleVaultDeleteFn: func(_ context.Context, _ keymanagement.ScheduleVaultDeletionRequest) (keymanagement.ScheduleVaultDeletionResponse, error) {
			return keymanagement.ScheduleVaultDeletionResponse{}, nil
		},
	}

	fakeMC := &fakeManagementClient{
		scheduleKeyDeleteFn: func(_ context.Context, req keymanagement.ScheduleKeyDeletionRequest) (keymanagement.ScheduleKeyDeletionResponse, error) {
			keyDeleteCalled = true
			assert.Equal(t, keyID, *req.KeyId)
			return keymanagement.ScheduleKeyDeletionResponse{}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithBothClients(credClient, fakeVC, fakeMC)

	v := newVaultCR("test-vault", "default")
	v.Status.OsokStatus.Ocid = ociv1beta1.OCID(vaultID)
	v.Spec.Key = &ociv1beta1.OciVaultKeySpec{
		KeyId:       ociv1beta1.OCID(keyID),
		DisplayName: "test-key",
		KeyShape:    ociv1beta1.OciVaultKeyShape{Algorithm: "AES", Length: 32},
	}

	done, err := mgr.Delete(context.Background(), v)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, keyDeleteCalled, "ScheduleKeyDeletion should have been called")
}

// ---------------------------------------------------------------------------
// GetCrdStatus tests
// ---------------------------------------------------------------------------

// TestGetCrdStatus_ReturnsStatus verifies status extraction from an OciVault object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := newMgr(credClient)

	v := &ociv1beta1.OciVault{}
	v.Status.OsokStatus.Ocid = "ocid1.vault.oc1..xxx"

	status, err := mgr.GetCrdStatus(v)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.vault.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := newMgr(credClient)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// ---------------------------------------------------------------------------
// CreateOrUpdate type assertion tests
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-OciVault objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := newMgr(credClient)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// GetVaultOcid path tests (exercised via CreateOrUpdate)
// ---------------------------------------------------------------------------

// TestGetVaultOcid_NotFound verifies that when ListVaults returns no matching vault,
// CreateOrUpdate proceeds to create a new vault.
func TestGetVaultOcid_NotFound(t *testing.T) {
	createdVault := makeActiveVault("ocid1.vault.oc1..new", "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	createCalled := false
	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, req keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			createCalled = true
			assert.Equal(t, "test-vault", *req.CreateVaultDetails.DisplayName)
			return keymanagement.CreateVaultResponse{Vault: createdVault}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, createCalled, "CreateVault should have been called")
	assert.True(t, credClient.createCalled, "CreateSecret should have been called")
}

// TestGetVaultOcid_Found_Active verifies that when ListVaults finds an ACTIVE vault,
// CreateOrUpdate binds to it instead of creating a new one.
func TestGetVaultOcid_Found_Active(t *testing.T) {
	existingID := "ocid1.vault.oc1..existing"
	existingVault := makeActiveVault(existingID, "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	createCalled := false
	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{
				Items: []keymanagement.VaultSummary{
					{
						Id:             common.String(existingID),
						DisplayName:    common.String("test-vault"),
						LifecycleState: keymanagement.VaultSummaryLifecycleStateActive,
					},
				},
			}, nil
		},
		getVaultFn: func(_ context.Context, _ keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error) {
			return keymanagement.GetVaultResponse{Vault: existingVault}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			createCalled = true
			return keymanagement.CreateVaultResponse{}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, createCalled, "CreateVault should NOT be called when vault is found by name")
}

// TestGetVaultOcid_Found_Creating verifies that a vault found in CREATING state
// returns a provisioning response (not yet successful, no error).
func TestGetVaultOcid_Found_Creating(t *testing.T) {
	existingID := "ocid1.vault.oc1..creating"
	creatingVault := makeCreatingVault(existingID, "test-vault")

	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{
				Items: []keymanagement.VaultSummary{
					{
						Id:             common.String(existingID),
						DisplayName:    common.String("test-vault"),
						LifecycleState: keymanagement.VaultSummaryLifecycleStateCreating,
					},
				},
			}, nil
		},
		getVaultFn: func(_ context.Context, _ keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error) {
			return keymanagement.GetVaultResponse{Vault: creatingVault}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "CREATING vault should return provisioning, not success")
	assert.False(t, credClient.createCalled, "secret should not be written while vault is CREATING")
}

// TestGetVaultOcid_ListError verifies that a ListVaults error is propagated.
func TestGetVaultOcid_ListError(t *testing.T) {
	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, errors.New("OCI API unavailable")
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// Lifecycle state transition tests
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_CREATING_State verifies that a newly created vault in CREATING
// state returns a provisioning response without writing a secret.
func TestCreateOrUpdate_CREATING_State(t *testing.T) {
	creatingVault := makeCreatingVault("ocid1.vault.oc1..creating", "test-vault")

	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: creatingVault}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "CREATING vault should not be IsSuccessful")
	assert.False(t, credClient.createCalled, "secret should not be created while vault is CREATING")
}

// TestCreateOrUpdate_ACTIVE_State verifies that an ACTIVE vault completes successfully
// and writes the secret.
func TestCreateOrUpdate_ACTIVE_State(t *testing.T) {
	activeVault := makeActiveVault("ocid1.vault.oc1..active", "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: activeVault}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled, "secret should be created for ACTIVE vault")
}

// TestCreateOrUpdate_DELETING_State verifies that a DELETING vault results in
// a Failed OSOK status and IsSuccessful=false.
func TestCreateOrUpdate_DELETING_State(t *testing.T) {
	deletingVault := keymanagement.Vault{
		Id:             common.String("ocid1.vault.oc1..deleting"),
		DisplayName:    common.String("test-vault"),
		LifecycleState: keymanagement.VaultLifecycleStateDeleting,
	}

	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: deletingVault}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "DELETING vault should not be IsSuccessful")
}

// ---------------------------------------------------------------------------
// Update path tests (VaultId already set in spec)
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_UpdatePath verifies that when VaultId is set, GetVault and
// UpdateVault are called (the create path is skipped).
func TestCreateOrUpdate_UpdatePath(t *testing.T) {
	vaultID := "ocid1.vault.oc1..existing"
	activeVault := makeActiveVault(vaultID, "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	updateCalled := false
	fakeVC := &fakeVaultClient{
		getVaultFn: func(_ context.Context, req keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error) {
			return keymanagement.GetVaultResponse{Vault: activeVault}, nil
		},
		updateVaultFn: func(_ context.Context, req keymanagement.UpdateVaultRequest) (keymanagement.UpdateVaultResponse, error) {
			updateCalled = true
			assert.Equal(t, vaultID, *req.VaultId)
			return keymanagement.UpdateVaultResponse{Vault: activeVault}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")
	v.Spec.VaultId = ociv1beta1.OCID(vaultID)
	v.Status.OsokStatus.Ocid = ociv1beta1.OCID(vaultID)

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateVault should have been called")
	assert.True(t, credClient.createCalled, "secret should be written after update")
}

// TestCreateOrUpdate_UpdatePath_GetVaultError verifies that a GetVault error in
// the update path is propagated.
func TestCreateOrUpdate_UpdatePath_GetVaultError(t *testing.T) {
	vaultID := "ocid1.vault.oc1..existing"

	fakeVC := &fakeVaultClient{
		getVaultFn: func(_ context.Context, _ keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error) {
			return keymanagement.GetVaultResponse{}, errors.New("vault not found")
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")
	v.Spec.VaultId = ociv1beta1.OCID(vaultID)

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_UpdatePath_UpdateVaultError verifies that an UpdateVault error
// is propagated.
func TestCreateOrUpdate_UpdatePath_UpdateVaultError(t *testing.T) {
	vaultID := "ocid1.vault.oc1..existing"
	activeVault := makeActiveVault(vaultID, "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	fakeVC := &fakeVaultClient{
		getVaultFn: func(_ context.Context, _ keymanagement.GetVaultRequest) (keymanagement.GetVaultResponse, error) {
			return keymanagement.GetVaultResponse{Vault: activeVault}, nil
		},
		updateVaultFn: func(_ context.Context, _ keymanagement.UpdateVaultRequest) (keymanagement.UpdateVaultResponse, error) {
			return keymanagement.UpdateVaultResponse{}, errors.New("update rejected")
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")
	v.Spec.VaultId = ociv1beta1.OCID(vaultID)
	v.Status.OsokStatus.Ocid = ociv1beta1.OCID(vaultID)

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// Key management tests
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_WithKey_NotFound_CreatesKey verifies that when the vault spec
// includes a key without an ID, and no matching key exists, a new key is created.
func TestCreateOrUpdate_WithKey_NotFound_CreatesKey(t *testing.T) {
	activeVault := makeActiveVault("ocid1.vault.oc1..xxx", "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	keyCreated := false
	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: activeVault}, nil
		},
	}

	fakeMC := &fakeManagementClient{
		listKeysFn: func(_ context.Context, _ keymanagement.ListKeysRequest) (keymanagement.ListKeysResponse, error) {
			return keymanagement.ListKeysResponse{}, nil
		},
		createKeyFn: func(_ context.Context, req keymanagement.CreateKeyRequest) (keymanagement.CreateKeyResponse, error) {
			keyCreated = true
			assert.Equal(t, "test-key", *req.CreateKeyDetails.DisplayName)
			newKey := keymanagement.Key{Id: common.String("ocid1.key.oc1..new")}
			return keymanagement.CreateKeyResponse{Key: newKey}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithBothClients(credClient, fakeVC, fakeMC)
	v := newVaultCR("test-vault", "default")
	v.Spec.Key = &ociv1beta1.OciVaultKeySpec{
		DisplayName: "test-key",
		KeyShape:    ociv1beta1.OciVaultKeyShape{Algorithm: "AES", Length: 32},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, keyCreated, "CreateKey should have been called")
}

// TestCreateOrUpdate_WithKey_Found_SkipsCreate verifies that when a key already exists
// by display name, CreateKey is not called.
func TestCreateOrUpdate_WithKey_Found_SkipsCreate(t *testing.T) {
	activeVault := makeActiveVault("ocid1.vault.oc1..xxx", "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	keyCreated := false
	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: activeVault}, nil
		},
	}

	fakeMC := &fakeManagementClient{
		listKeysFn: func(_ context.Context, _ keymanagement.ListKeysRequest) (keymanagement.ListKeysResponse, error) {
			return keymanagement.ListKeysResponse{
				Items: []keymanagement.KeySummary{
					{
						Id:             common.String("ocid1.key.oc1..existing"),
						DisplayName:    common.String("test-key"),
						LifecycleState: keymanagement.KeySummaryLifecycleStateEnabled,
					},
				},
			}, nil
		},
		createKeyFn: func(_ context.Context, _ keymanagement.CreateKeyRequest) (keymanagement.CreateKeyResponse, error) {
			keyCreated = true
			return keymanagement.CreateKeyResponse{}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithBothClients(credClient, fakeVC, fakeMC)
	v := newVaultCR("test-vault", "default")
	v.Spec.Key = &ociv1beta1.OciVaultKeySpec{
		DisplayName: "test-key",
		KeyShape:    ociv1beta1.OciVaultKeyShape{Algorithm: "AES", Length: 32},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, keyCreated, "CreateKey should NOT be called when key already exists")
}

// TestCreateOrUpdate_WithKey_ByID_BindsExisting verifies that when a key ID is set
// in the spec, GetKey is called (not ListKeys or CreateKey).
func TestCreateOrUpdate_WithKey_ByID_BindsExisting(t *testing.T) {
	activeVault := makeActiveVault("ocid1.vault.oc1..xxx", "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	getKeyCalled := false
	listKeysCalled := false
	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: activeVault}, nil
		},
	}

	fakeMC := &fakeManagementClient{
		getKeyFn: func(_ context.Context, req keymanagement.GetKeyRequest) (keymanagement.GetKeyResponse, error) {
			getKeyCalled = true
			assert.Equal(t, "ocid1.key.oc1..existingkey", *req.KeyId)
			k := keymanagement.Key{Id: req.KeyId}
			return keymanagement.GetKeyResponse{Key: k}, nil
		},
		listKeysFn: func(_ context.Context, _ keymanagement.ListKeysRequest) (keymanagement.ListKeysResponse, error) {
			listKeysCalled = true
			return keymanagement.ListKeysResponse{}, nil
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithBothClients(credClient, fakeVC, fakeMC)
	v := newVaultCR("test-vault", "default")
	v.Spec.Key = &ociv1beta1.OciVaultKeySpec{
		KeyId:       "ocid1.key.oc1..existingkey",
		DisplayName: "test-key",
		KeyShape:    ociv1beta1.OciVaultKeyShape{Algorithm: "AES", Length: 32},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, getKeyCalled, "GetKey should be called when key ID is set")
	assert.False(t, listKeysCalled, "ListKeys should NOT be called when key ID is set")
}

// TestCreateOrUpdate_WithKey_CreateError verifies that key creation errors propagate.
func TestCreateOrUpdate_WithKey_CreateError(t *testing.T) {
	activeVault := makeActiveVault("ocid1.vault.oc1..xxx", "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: activeVault}, nil
		},
	}

	fakeMC := &fakeManagementClient{
		listKeysFn: func(_ context.Context, _ keymanagement.ListKeysRequest) (keymanagement.ListKeysResponse, error) {
			return keymanagement.ListKeysResponse{}, nil
		},
		createKeyFn: func(_ context.Context, _ keymanagement.CreateKeyRequest) (keymanagement.CreateKeyResponse, error) {
			return keymanagement.CreateKeyResponse{}, errors.New("quota exceeded")
		},
	}

	credClient := &fakeCredentialClient{}
	mgr := newMgrWithBothClients(credClient, fakeVC, fakeMC)
	v := newVaultCR("test-vault", "default")
	v.Spec.Key = &ociv1beta1.OciVaultKeySpec{
		DisplayName: "test-key",
		KeyShape:    ociv1beta1.OciVaultKeyShape{Algorithm: "AES", Length: 32},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// Secret writing tests
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_SecretAlreadyExists verifies that when the secret already exists,
// CreateOrUpdate treats it as success (idempotent operation).
func TestCreateOrUpdate_SecretAlreadyExists(t *testing.T) {
	activeVault := makeActiveVault("ocid1.vault.oc1..xxx", "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: activeVault}, nil
		},
	}

	alreadyExistsErr := apierrors.NewAlreadyExists(schema.GroupResource{Group: "", Resource: "secrets"}, "test-vault")
	credClient := &fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return false, alreadyExistsErr
		},
	}

	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful, "AlreadyExists error should be treated as success")
}

// TestCreateOrUpdate_SecretCreateFails verifies that a genuine secret creation
// failure causes CreateOrUpdate to return an error.
func TestCreateOrUpdate_SecretCreateFails(t *testing.T) {
	activeVault := makeActiveVault("ocid1.vault.oc1..xxx", "test-vault",
		"https://mgmt.example.com", "https://crypto.example.com")

	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: activeVault}, nil
		},
	}

	credClient := &fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return false, errors.New("secret backend unavailable")
		},
	}

	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_SecretWritten_ContainsVaultData verifies the exact content of
// the secret written to the credential store after a successful vault creation.
func TestCreateOrUpdate_SecretWritten_ContainsVaultData(t *testing.T) {
	mgmtEndpoint := "https://abc-management.kms.us-ashburn-1.oraclecloud.com"
	cryptoEndpoint := "https://abc-crypto.kms.us-ashburn-1.oraclecloud.com"
	vaultID := "ocid1.vault.oc1..xxx"
	activeVault := makeActiveVault(vaultID, "test-vault", mgmtEndpoint, cryptoEndpoint)

	fakeVC := &fakeVaultClient{
		listVaultsFn: func(_ context.Context, _ keymanagement.ListVaultsRequest) (keymanagement.ListVaultsResponse, error) {
			return keymanagement.ListVaultsResponse{}, nil
		},
		createVaultFn: func(_ context.Context, _ keymanagement.CreateVaultRequest) (keymanagement.CreateVaultResponse, error) {
			return keymanagement.CreateVaultResponse{Vault: activeVault}, nil
		},
	}

	var capturedData map[string][]byte
	credClient := &fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, data map[string][]byte) (bool, error) {
			capturedData = data
			return true, nil
		},
	}

	mgr := newMgrWithVaultClient(credClient, fakeVC)
	v := newVaultCR("test-vault", "default")

	resp, err := mgr.CreateOrUpdate(context.Background(), v, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	assert.Equal(t, vaultID, string(capturedData["id"]))
	assert.Equal(t, "test-vault", string(capturedData["displayName"]))
	assert.Equal(t, mgmtEndpoint, string(capturedData["managementEndpoint"]))
	assert.Equal(t, cryptoEndpoint, string(capturedData["cryptoEndpoint"]))
}
