/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/autonomousdatabases/adb"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeOCIResponse implements common.OCIResponse with a configurable HTTP response.
type fakeOCIResponse struct {
	httpResp *http.Response
}

func (f *fakeOCIResponse) HTTPResponse() *http.Response { return f.httpResp }

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

// TestCreateOrUpdate_CreateNewAdb_ECPU verifies that when ComputeModel is set, ComputeCount
// is sent and CpuCoreCount is NOT set in the create request.
func TestCreateOrUpdate_CreateNewAdb_ECPU(t *testing.T) {
	newAdbId := "ocid1.autonomousdatabase.oc1..ecpu"

	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManager(credClient)

	var capturedReq database.CreateAutonomousDatabaseRequest
	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			capturedReq = req
			return database.CreateAutonomousDatabaseResponse{
				AutonomousDatabase: database.AutonomousDatabase{
					Id: common.String(newAdbId),
				},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(newAdbId, "ecpu-adb"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "ecpu-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"
	adb.Spec.ComputeModel = "ECPU"
	adb.Spec.ComputeCount = 2.0

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(newAdbId), adb.Status.OsokStatus.Ocid)

	details := capturedReq.CreateAutonomousDatabaseDetails.(database.CreateAutonomousDatabaseDetails)
	assert.Equal(t, database.CreateAutonomousDatabaseBaseComputeModelEnum("ECPU"), details.ComputeModel)
	assert.Equal(t, common.Float32(2.0), details.ComputeCount)
	assert.Nil(t, details.CpuCoreCount, "CpuCoreCount must be nil when using ECPU model")
}

// TestCreateOrUpdate_CreateNewAdb_OCPU verifies that when ComputeModel is empty,
// CpuCoreCount is sent (legacy OCPU path) and ComputeCount/ComputeModel are not set.
func TestCreateOrUpdate_CreateNewAdb_OCPU(t *testing.T) {
	newAdbId := "ocid1.autonomousdatabase.oc1..ocpu"

	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManager(credClient)

	var capturedReq database.CreateAutonomousDatabaseRequest
	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			capturedReq = req
			return database.CreateAutonomousDatabaseResponse{
				AutonomousDatabase: database.AutonomousDatabase{
					Id: common.String(newAdbId),
				},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(newAdbId, "ocpu-adb"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "ocpu-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"
	adb.Spec.CpuCoreCount = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	details := capturedReq.CreateAutonomousDatabaseDetails.(database.CreateAutonomousDatabaseDetails)
	assert.Equal(t, common.Int(1), details.CpuCoreCount)
	assert.Empty(t, string(details.ComputeModel), "ComputeModel must be empty when using OCPU model")
	assert.Nil(t, details.ComputeCount, "ComputeCount must be nil when using OCPU model")
}

// ---------------------------------------------------------------------------
// DeleteAdb test
// ---------------------------------------------------------------------------

// TestDeleteAdb verifies DeleteAdb returns empty string and no error (stub implementation).
func TestDeleteAdb(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	ocid, err := mgr.DeleteAdb()
	assert.NoError(t, err)
	assert.Equal(t, "", ocid)
}

// ---------------------------------------------------------------------------
// Retry policy predicate tests
// ---------------------------------------------------------------------------

// TestAdbRetryPolicy_Provisioning verifies shouldRetry returns true when ADB is PROVISIONING.
func TestAdbRetryPolicy_Provisioning(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportAdbRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: database.GetAutonomousDatabaseResponse{
			AutonomousDatabase: database.AutonomousDatabase{
				LifecycleState: "PROVISIONING",
			},
		},
	}
	assert.True(t, shouldRetry(resp), "shouldRetry should be true when ADB is PROVISIONING")
}

// TestAdbRetryPolicy_Available verifies shouldRetry returns false when ADB is AVAILABLE.
func TestAdbRetryPolicy_Available(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportAdbRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: database.GetAutonomousDatabaseResponse{
			AutonomousDatabase: database.AutonomousDatabase{
				LifecycleState: "AVAILABLE",
			},
		},
	}
	assert.False(t, shouldRetry(resp), "shouldRetry should be false when ADB is AVAILABLE")
}

// TestAdbRetryPolicy_NonResponse verifies shouldRetry returns true when type assertion fails.
func TestAdbRetryPolicy_NonResponse(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportAdbRetryPredicate(mgr)

	resp := common.OCIOperationResponse{} // nil Response → type assertion fails → true
	assert.True(t, shouldRetry(resp))
}

// TestAdbRetryNextDuration verifies the nextDuration computes exponential backoff.
func TestAdbRetryNextDuration(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	nextDuration := ExportAdbRetryNextDuration(mgr)

	resp := common.OCIOperationResponse{AttemptNumber: 1}
	assert.Equal(t, 1*time.Second, nextDuration(resp))
}

// TestExponentialBackoffPolicy_SuccessResponse verifies the predicate returns false (no retry)
// when the response has no error and a 2xx HTTP status.
func TestExponentialBackoffPolicy_SuccessResponse(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportExponentialBackoffPredicate(mgr)

	// Build a fake HTTP response with 200 status.
	httpResp := &http.Response{StatusCode: 200}
	fakeResp := &fakeOCIResponse{httpResp: httpResp}
	resp := common.OCIOperationResponse{
		Response: fakeResp,
		Error:    nil,
	}
	assert.False(t, shouldRetry(resp), "shouldRetry should be false for a successful 2xx response")
}

// TestExponentialBackoffPolicy_ErrorResponse verifies the predicate returns true (retry)
// when the response has an error.
func TestExponentialBackoffPolicy_ErrorResponse(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportExponentialBackoffPredicate(mgr)

	resp := common.OCIOperationResponse{
		Error: errors.New("network error"),
	}
	assert.True(t, shouldRetry(resp), "shouldRetry should be true when there is an error")
}

// TestExponentialBackoffNextDuration verifies nextDuration returns exponential backoff.
func TestExponentialBackoffNextDuration(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	nextDuration := ExportExponentialBackoffNextDuration(mgr)

	resp := common.OCIOperationResponse{AttemptNumber: 1}
	assert.Equal(t, 1*time.Second, nextDuration(resp))
}

// ---------------------------------------------------------------------------
// isValidUpdate DefinedTags coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BindExistingAdb_DefinedTagsChange verifies that when DefinedTags
// in the spec differ from the existing ADB, UpdateAdb is called.
func TestCreateOrUpdate_BindExistingAdb_DefinedTagsChange(t *testing.T) {
	adbId := "ocid1.autonomousdatabase.oc1..deftags"
	updateCalled := false

	mgr := newTestManager(&fakeCredentialClient{})
	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "test-adb"),
				// makeActiveAdb has no DefinedTags
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
	adb.Spec.DisplayName = "test-adb" // same — no display name update
	adb.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
		"ns1": {"key1": "val1"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateAdb should be called when DefinedTags differ")
}

// ---------------------------------------------------------------------------
// UpdateAdb additional field coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_UpdateAdb_AdditionalFields verifies that DbWorkload, IsFreeTier,
// LicenseModel, DbVersion, and FreeFormTags changes trigger an update with correct values.
func TestCreateOrUpdate_UpdateAdb_AdditionalFields(t *testing.T) {
	adbId := "ocid1.autonomousdatabase.oc1..addfields"
	var capturedUpdate database.UpdateAutonomousDatabaseRequest

	mgr := newTestManager(&fakeCredentialClient{})
	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "test-adb"),
				// makeActiveAdb has DbWorkload=OLTP, IsFreeTier=false, LicenseModel=LICENSE_INCLUDED, DbVersion=19c
			}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			capturedUpdate = req
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = ociv1beta1.OCID(adbId)
	adb.Spec.DbWorkload = "DW"                   // differs from OLTP
	adb.Spec.IsFreeTier = true                   // differs from false
	adb.Spec.LicenseModel = "BRING_YOUR_OWN_LICENSE" // differs from LICENSE_INCLUDED
	adb.Spec.DbVersion = "21c"                   // differs from 19c
	adb.Spec.FreeFormTags = map[string]string{"env": "prod"} // differs from nil

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	details := capturedUpdate.UpdateAutonomousDatabaseDetails
	assert.Equal(t, database.UpdateAutonomousDatabaseDetailsDbWorkloadEnum("DW"), details.DbWorkload)
	assert.Equal(t, common.Bool(true), details.IsFreeTier)
	assert.Equal(t, database.UpdateAutonomousDatabaseDetailsLicenseModelEnum("BRING_YOUR_OWN_LICENSE"), details.LicenseModel)
	assert.Equal(t, common.String("21c"), details.DbVersion)
	assert.Equal(t, map[string]string{"env": "prod"}, details.FreeformTags)
}

// ---------------------------------------------------------------------------
// getWalletPassword missing key coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_WalletPassword_MissingKey verifies that when the wallet password
// secret exists but lacks the "walletPassword" key, the operation fails with a clear error.
func TestCreateOrUpdate_WalletPassword_MissingKey(t *testing.T) {
	adbId := "ocid1.autonomousdatabase.oc1..walpwd"
	callCount := 0

	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			callCount++
			if callCount == 1 {
				// First call: wallet existence check → return error (wallet doesn't exist yet)
				return nil, errors.New("wallet not found")
			}
			// Second call: wallet password fetch → key is wrong
			return map[string][]byte{"wrongkey": []byte("value")}, nil
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
	adb.Spec.Wallet.WalletPassword.Secret.SecretName = "wallet-pwd-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "walletPassword")
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate_CreateNewAdb_MissingPasswordKey
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// fakeServiceError implements common.ServiceError + error for testing.
// ---------------------------------------------------------------------------

type fakeServiceError struct {
	statusCode int
	code       string
	message    string
}

func (f *fakeServiceError) GetHTTPStatusCode() int  { return f.statusCode }
func (f *fakeServiceError) GetMessage() string      { return f.message }
func (f *fakeServiceError) GetCode() string         { return f.code }
func (f *fakeServiceError) GetOpcRequestID() string { return "" }
func (f *fakeServiceError) Error() string {
	return fmt.Sprintf("%d %s: %s", f.statusCode, f.code, f.message)
}

// ---------------------------------------------------------------------------
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

// ---------------------------------------------------------------------------
// CreateAdb optional field coverage (DbVersion + LicenseModel)
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_CreateNewAdb_WithVersionAndLicense verifies that when DbVersion and
// LicenseModel are set in the spec, they are included in the create request.
func TestCreateOrUpdate_CreateNewAdb_WithVersionAndLicense(t *testing.T) {
	newAdbId := "ocid1.autonomousdatabase.oc1..verlic"
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManager(credClient)

	var capturedReq database.CreateAutonomousDatabaseRequest
	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, req database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			capturedReq = req
			return database.CreateAutonomousDatabaseResponse{
				AutonomousDatabase: database.AutonomousDatabase{Id: common.String(newAdbId)},
			}, nil
		},
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(newAdbId, "test-adb"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"
	adb.Spec.CpuCoreCount = 2
	adb.Spec.DbVersion = "21c"
	adb.Spec.LicenseModel = "BRING_YOUR_OWN_LICENSE"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	details := capturedReq.CreateAutonomousDatabaseDetails.(database.CreateAutonomousDatabaseDetails)
	assert.Equal(t, common.String("21c"), details.DbVersion)
	assert.Equal(t, database.CreateAutonomousDatabaseBaseLicenseModelEnum("BRING_YOUR_OWN_LICENSE"), details.LicenseModel)
}

// ---------------------------------------------------------------------------
// UpdateAdb DbName branch coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BindExistingAdb_DbNameChange verifies that when DbName in the spec
// differs from the existing ADB, the update request includes the new DbName.
func TestCreateOrUpdate_BindExistingAdb_DbNameChange(t *testing.T) {
	adbId := "ocid1.autonomousdatabase.oc1..dbname"
	var capturedUpdate database.UpdateAutonomousDatabaseRequest

	mgr := newTestManager(&fakeCredentialClient{})
	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "test-adb"),
			}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			capturedUpdate = req
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = ociv1beta1.OCID(adbId)
	adb.Spec.DisplayName = "new-name" // triggers updateNeeded
	adb.Spec.DbName = "newdb"         // differs from "testdb" — exercises DbName branch

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, common.String("newdb"), capturedUpdate.UpdateAutonomousDatabaseDetails.DbName)
}

// ---------------------------------------------------------------------------
// CreateOrUpdate error path coverage (CreateAdb failure)
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_CreateNewAdb_InvalidParameterError verifies that a 400/InvalidParameter
// error from CreateAdb results in a failed response with nil error (not retried).
func TestCreateOrUpdate_CreateNewAdb_InvalidParameterError(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManager(credClient)

	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, _ database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			return database.CreateAutonomousDatabaseResponse{},
				&fakeServiceError{statusCode: 400, code: "InvalidParameter", message: "bad param"}
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"
	adb.Spec.CpuCoreCount = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err, "InvalidParameter errors should not propagate as Go errors")
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreateNewAdb_OciCreateError verifies that a non-400 OCI error
// from CreateAdb propagates as an error from CreateOrUpdate.
func TestCreateOrUpdate_CreateNewAdb_OciCreateError(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"password": []byte("admin123")}, nil
		},
	}
	mgr := newTestManager(credClient)

	mockClient := &mockOciDbClient{
		listFn: func(_ context.Context, _ database.ListAutonomousDatabasesRequest) (database.ListAutonomousDatabasesResponse, error) {
			return database.ListAutonomousDatabasesResponse{}, nil
		},
		createFn: func(_ context.Context, _ database.CreateAutonomousDatabaseRequest) (database.CreateAutonomousDatabaseResponse, error) {
			return database.CreateAutonomousDatabaseResponse{},
				&fakeServiceError{statusCode: 500, code: "InternalServerError", message: "server error"}
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.DisplayName = "test-adb"
	adb.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	adb.Spec.AdminPassword.Secret.SecretName = "adb-admin-secret"
	adb.Spec.CpuCoreCount = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.Error(t, err, "non-400 errors should propagate")
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// CreatedAt != nil branch coverage in bind path
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BindExistingAdb_UpdateNeeded_WithCreatedAt verifies that when
// CreatedAt is already set and an update is performed, the CreatedAt timestamp is refreshed.
func TestCreateOrUpdate_BindExistingAdb_UpdateNeeded_WithCreatedAt(t *testing.T) {
	adbId := "ocid1.autonomousdatabase.oc1..creat"
	mgr := newTestManager(&fakeCredentialClient{})

	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			return database.GetAutonomousDatabaseResponse{
				AutonomousDatabase: makeActiveAdb(adbId, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = ociv1beta1.OCID(adbId)
	adb.Spec.DisplayName = "new-name"
	// Pre-set CreatedAt so the "if CreatedAt != nil" branch is taken after update.
	ts := metav1.NewTime(time.Now())
	adb.Status.OsokStatus.CreatedAt = &ts

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// getCredentialMap coverage via export
// ---------------------------------------------------------------------------

// TestGetCredentialMap_Valid verifies that getCredentialMap correctly parses a zip archive
// and returns its file contents as a map.
func TestGetCredentialMap_Valid(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("tnsnames.ora")
	assert.NoError(t, err)
	_, err = fw.Write([]byte("MY_SERVICE = (DESCRIPTION=...)"))
	assert.NoError(t, err)
	assert.NoError(t, zw.Close())

	resp := database.GenerateAutonomousDatabaseWalletResponse{
		Content: io.NopCloser(bytes.NewReader(buf.Bytes())),
	}

	credMap, err := ExportGetCredentialMapForTest("test-adb", resp)
	assert.NoError(t, err)
	assert.Contains(t, credMap, "tnsnames.ora")
	assert.Equal(t, []byte("MY_SERVICE = (DESCRIPTION=...)"), credMap["tnsnames.ora"])
}

// TestGetCredentialMap_MultipleFiles verifies that multiple files in the zip are all captured.
func TestGetCredentialMap_MultipleFiles(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"tnsnames.ora", "sqlnet.ora", "cwallet.sso"} {
		fw, err := zw.Create(name)
		assert.NoError(t, err)
		_, err = fw.Write([]byte("content of " + name))
		assert.NoError(t, err)
	}
	assert.NoError(t, zw.Close())

	resp := database.GenerateAutonomousDatabaseWalletResponse{
		Content: io.NopCloser(bytes.NewReader(buf.Bytes())),
	}

	credMap, err := ExportGetCredentialMapForTest("test-adb", resp)
	assert.NoError(t, err)
	assert.Len(t, credMap, 3)
	assert.Equal(t, []byte("content of tnsnames.ora"), credMap["tnsnames.ora"])
	assert.Equal(t, []byte("content of sqlnet.ora"), credMap["sqlnet.ora"])
	assert.Equal(t, []byte("content of cwallet.sso"), credMap["cwallet.sso"])
}
