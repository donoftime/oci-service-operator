/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/autonomousdatabases/adb"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

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

// mockOciDbClient implements DatabaseClientInterface for testing.
type mockOciDbClient struct {
	createFn func(context.Context, database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error)
	listFn   func(context.Context, database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error)
	getFn    func(context.Context, database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error)
	updateFn func(context.Context, database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error)
}

func (m *mockOciDbClient) CreateAutonomousDatabase(ctx context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return database.CreateAutonomousDatabaseResponse{}, nil
}

func (m *mockOciDbClient) ListAutonomousDatabases(ctx context.Context, req database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return database.ListAutonomousDatabasesResponse{}, nil
}

func (m *mockOciDbClient) GetAutonomousDatabase(ctx context.Context, req database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return database.GetAutonomousDatabaseResponse{}, nil
}

func (m *mockOciDbClient) UpdateAutonomousDatabase(ctx context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return database.UpdateAutonomousDatabaseResponse{}, nil
}

// makeActiveAdb returns a minimal AutonomousDatabase suitable for mock responses.
func makeActiveAdb(id, displayName string) database.AutonomousDatabase {
	return database.AutonomousDatabase{
		Id:                   common.String(id),
		DisplayName:          common.String(displayName),
		DbName:               common.String("testdb"),
		CpuCoreCount:         common.Int(2),
		DataStorageSizeInTBs: common.Int(1),
		DbVersion:            common.String("19c"),
		DbWorkload:           database.AutonomousDatabaseDbWorkloadOltp,
		IsAutoScalingEnabled: common.Bool(false),
		IsFreeTier:           common.Bool(false),
		LicenseModel:         database.AutonomousDatabaseLicenseModelLicenseIncluded,
	}
}

func newTestManager(credClient *fakeCredentialClient) *AdbServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return NewAdbServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)
}

// --- Structural tests (no OCI calls) ---

// TestGetCrdStatus_Happy verifies status is returned from an AutonomousDatabases object.
func TestGetCrdStatus_Happy(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Status.OsokStatus.Ocid = "ocid1.autonomousdatabase.oc1..xxx"

	status, err := mgr.GetCrdStatus(adb)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.autonomousdatabase.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert the type assertion for Autonomous Databases")
}

// TestDelete_NoOcid verifies deletion is a no-op regardless of OCID state.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := newTestManager(credClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Name = "test-adb"
	adb.Namespace = "default"

	done, err := mgr.Delete(context.Background(), adb)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-AutonomousDatabases objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// --- Mock-based tests (require OCI client injection) ---

// TestCreateOrUpdate_BindExistingAdb_NothingToUpdate verifies that when AdbId is specified
// and the ADB fields match the spec, no update is issued and the manager reports success.
func TestCreateOrUpdate_BindExistingAdb_NothingToUpdate(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	adbId := "ocid1.autonomousdatabase.oc1..xxx"
	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "test-adb"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = ociv1beta1.OCID(adbId)
	adb.Spec.DisplayName = "test-adb" // same as returned — no update needed

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(adbId), adb.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_BindExistingAdb_UpdateNeeded verifies that when the display name
// differs from the spec, an update is issued.
func TestCreateOrUpdate_BindExistingAdb_UpdateNeeded(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	adbId := "ocid1.autonomousdatabase.oc1..yyy"
	updateCalled := false

	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = ociv1beta1.OCID(adbId)
	adb.Spec.DisplayName = "new-name" // differs from returned "old-name" → triggers update

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateAutonomousDatabase should be called")
}

// TestCreateOrUpdate_BindExistingAdb_UpdateMultipleFields verifies that when multiple
// spec fields differ from the current ADB state, all changed fields are included in
// the update request.
func TestCreateOrUpdate_BindExistingAdb_UpdateMultipleFields(t *testing.T) {
	adbId := "ocid1.autonomousdatabase.oc1..multi"
	updateCalled := false

	mgr := newTestManager(&fakeCredentialClient{})

	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = ociv1beta1.OCID(adbId)
	adb.Spec.DisplayName = "new-name"    // differs from "old-name"
	adb.Spec.CpuCoreCount = 4            // differs from 2
	adb.Spec.DataStorageSizeInTBs = 2    // differs from 1
	adb.Spec.IsAutoScalingEnabled = true // differs from false

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateAutonomousDatabase should be called")
}

// TestCreateOrUpdate_FindExistingAdb verifies that when no AdbId is in the spec,
// ListAutonomousDatabases finds an existing ADB by display name.
func TestCreateOrUpdate_FindExistingAdb(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	adbId := "ocid1.autonomousdatabase.oc1..found"

	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{
				Items: []database.AutonomousDatabaseSummary{
					{
						Id:             common.String(adbId),
						LifecycleState: database.AutonomousDatabaseSummaryLifecycleStateAvailable,
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "my-adb"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	// No AdbId in spec — should discover via ListAutonomousDatabases
	adb.Spec.DisplayName = "my-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(adbId), adb.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_OciGetError verifies that an OCI GetAutonomousDatabase error
// propagates as a failure from CreateOrUpdate.
func TestCreateOrUpdate_OciGetError(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{}, errors.New("OCI API error")
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = "ocid1.autonomousdatabase.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_OciListError verifies that a ListAutonomousDatabases error
// is returned when no AdbId is in the spec.
func TestCreateOrUpdate_OciListError(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, errors.New("list API error")
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	// No AdbId — triggers ListAutonomousDatabases
	adb.Spec.DisplayName = "my-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreateNewAdb verifies that when no AdbId is in the spec and no
// existing ADB is found by name, a new ADB is created and its OCID is recorded.
func TestCreateOrUpdate_CreateNewAdb(t *testing.T) {
	newAdbId := "ocid1.autonomousdatabase.oc1..new"

	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManager(credClient)

	createCalled := false
	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil // empty — no existing ADB
		},
		createFn: func(_ context.Context, _ database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			createCalled = true
			return database.CreateAutonomousDatabaseResponse{
				AutonomousDatabase: database.AutonomousDatabase{
					Id: common.String(newAdbId),
				},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(newAdbId, "new-adb"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "new-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, createCalled, "CreateAutonomousDatabase should be called")
	assert.Equal(t, ociv1beta1.OCID(newAdbId), adb.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_CreateNewAdb_GetSecretError verifies that a GetSecret error
// when fetching the admin password is propagated correctly.
func TestCreateOrUpdate_CreateNewAdb_GetSecretError(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return nil, errors.New("secret not found")
		},
	}
	mgr := newTestManager(credClient)

	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "my-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_WithWallet_AlreadyExists verifies that when the wallet secret
// already exists, GenerateWallet returns success without re-generating.
func TestCreateOrUpdate_WithWallet_AlreadyExists(t *testing.T) {
	adbId := "ocid1.autonomousdatabase.oc1..wallet"

	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return nil, nil // nil error = wallet secret exists
		},
	}
	mgr := newTestManager(credClient)

	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "test-adb"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Name = "test-adb"
	adb.Namespace = "default"
	adb.Spec.AdbId = ociv1beta1.OCID(adbId)
	adb.Spec.DisplayName = "test-adb"                                  // same — no update
	adb.Spec.Wallet.WalletPassword.Secret.SecretName = "wallet-secret" // triggers GenerateWallet

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_WithWallet_PasswordSecretError verifies that when the wallet secret
// does not exist and fetching the wallet password secret fails, the error propagates.
func TestCreateOrUpdate_WithWallet_PasswordSecretError(t *testing.T) {
	adbId := "ocid1.autonomousdatabase.oc1..wallerr"
	callCount := 0

	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			callCount++
			if callCount == 1 {
				// First call checks whether the wallet already exists — return error (doesn't exist)
				return nil, errors.New("not found")
			}
			// Second call fetches the wallet password — also fails
			return nil, errors.New("wallet password secret not found")
		},
	}
	mgr := newTestManager(credClient)

	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "test-adb"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Name = "test-adb"
	adb.Namespace = "default"
	adb.Spec.AdbId = ociv1beta1.OCID(adbId)
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.Wallet.WalletPassword.Secret.SecretName = "wallet-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreateNewAdb_MissingPasswordKey verifies that a missing "password"
// key in the admin secret causes a clear error before any OCI call is made.
func TestCreateOrUpdate_CreateNewAdb_MissingPasswordKey(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"wrongkey": []byte("value")}, nil
		},
	}
	mgr := newTestManager(credClient)

	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "my-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password key")
	assert.False(t, resp.IsSuccessful)
}
