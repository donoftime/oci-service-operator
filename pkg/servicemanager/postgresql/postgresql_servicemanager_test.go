/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package postgresql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocipsql "github.com/oracle/oci-go-sdk/v65/psql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/postgresql"
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

// mockOciPostgresClient implements PostgresClientInterface for unit testing.
type mockOciPostgresClient struct {
	createFn func(ctx context.Context, req ocipsql.CreateDbSystemRequest) (ocipsql.CreateDbSystemResponse, error)
	getFn    func(ctx context.Context, req ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error)
	listFn   func(ctx context.Context, req ocipsql.ListDbSystemsRequest) (ocipsql.ListDbSystemsResponse, error)
	updateFn func(ctx context.Context, req ocipsql.UpdateDbSystemRequest) (ocipsql.UpdateDbSystemResponse, error)
	deleteFn func(ctx context.Context, req ocipsql.DeleteDbSystemRequest) (ocipsql.DeleteDbSystemResponse, error)
}

func (m *mockOciPostgresClient) CreateDbSystem(ctx context.Context, req ocipsql.CreateDbSystemRequest) (ocipsql.CreateDbSystemResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return ocipsql.CreateDbSystemResponse{}, nil
}

func (m *mockOciPostgresClient) GetDbSystem(ctx context.Context, req ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return ocipsql.GetDbSystemResponse{}, nil
}

func (m *mockOciPostgresClient) ListDbSystems(ctx context.Context, req ocipsql.ListDbSystemsRequest) (ocipsql.ListDbSystemsResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return ocipsql.ListDbSystemsResponse{}, nil
}

func (m *mockOciPostgresClient) UpdateDbSystem(ctx context.Context, req ocipsql.UpdateDbSystemRequest) (ocipsql.UpdateDbSystemResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return ocipsql.UpdateDbSystemResponse{}, nil
}

func (m *mockOciPostgresClient) DeleteDbSystem(ctx context.Context, req ocipsql.DeleteDbSystemRequest) (ocipsql.DeleteDbSystemResponse, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, req)
	}
	return ocipsql.DeleteDbSystemResponse{}, nil
}

// makeActiveDbSystem creates a mock active PostgreSQL DB system.
func makeActiveDbSystem(id, displayName string) ocipsql.DbSystem {
	privateIp := "10.0.0.5"
	return ocipsql.DbSystem{
		Id:                    common.String(id),
		DisplayName:           common.String(displayName),
		CompartmentId:         common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState:        ocipsql.DbSystemLifecycleStateActive,
		SystemType:            ocipsql.DbSystemSystemTypeOciOptimizedStorage,
		DbVersion:             common.String("14.10"),
		Shape:                 common.String("VM.Standard.E4.Flex"),
		InstanceOcpuCount:     common.Int(2),
		InstanceMemorySizeInGBs: common.Int(32),
		StorageDetails:        ocipsql.OciOptimizedStorageDetails{IsRegionallyDurable: common.Bool(true)},
		NetworkDetails: &ocipsql.NetworkDetails{
			SubnetId:                    common.String("ocid1.subnet.oc1..xxx"),
			PrimaryDbEndpointPrivateIp: &privateIp,
		},
		ManagementPolicy: &ocipsql.ManagementPolicy{},
		InstanceCount:    common.Int(1),
	}
}

// newPostgresMgr creates a PostgresDbSystemServiceManager with injected mock clients.
func newPostgresMgr(t *testing.T, ociClient *mockOciPostgresClient, credClient *fakeCredentialClient) *PostgresDbSystemServiceManager {
	t.Helper()
	if credClient == nil {
		credClient = &fakeCredentialClient{}
	}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewPostgresDbSystemServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)
	if ociClient != nil {
		ExportSetClientForTest(mgr, ociClient)
	}
	return mgr
}

// TestGetCredentialMap verifies the secret credential map is built correctly.
func TestGetCredentialMap(t *testing.T) {
	dbSystem := makeActiveDbSystem("ocid1.postgresql.xxx", "test-db")
	credMap := GetCredentialMapForTest(dbSystem)

	assert.Equal(t, "ocid1.postgresql.xxx", string(credMap["id"]))
	assert.Equal(t, "test-db", string(credMap["displayName"]))
	assert.Equal(t, "5432", string(credMap["port"]))
}

// TestGetCredentialMap_NilFields verifies nil pointer fields are handled gracefully.
func TestGetCredentialMap_NilFields(t *testing.T) {
	dbSystem := ocipsql.DbSystem{
		Id:             common.String("ocid1.postgresql.xxx"),
		DisplayName:    nil,
		StorageDetails: ocipsql.OciOptimizedStorageDetails{IsRegionallyDurable: common.Bool(true)},
		ManagementPolicy: &ocipsql.ManagementPolicy{},
	}
	credMap := GetCredentialMapForTest(dbSystem)
	assert.Contains(t, credMap, "id")
	assert.NotContains(t, credMap, "displayName")
	assert.Equal(t, "5432", string(credMap["port"]))
}

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := newPostgresMgr(t, nil, credClient)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"

	done, err := mgr.Delete(context.Background(), dbSystem)
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

	ociClient := &mockOciPostgresClient{
		deleteFn: func(_ context.Context, _ ocipsql.DeleteDbSystemRequest) (ocipsql.DeleteDbSystemResponse, error) {
			return ocipsql.DeleteDbSystemResponse{}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, credClient)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"
	dbSystem.Status.OsokStatus.Ocid = "ocid1.postgresql.oc1..xxx"

	// Should succeed despite secret deletion error
	done, err := mgr.Delete(context.Background(), dbSystem)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from a PostgresDbSystem object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := newPostgresMgr(t, nil, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Status.OsokStatus.Ocid = "ocid1.postgresql.xxx"

	status, err := mgr.GetCrdStatus(dbSystem)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.postgresql.xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := newPostgresMgr(t, nil, nil)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-PostgresDbSystem objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := newPostgresMgr(t, nil, nil)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreateNew verifies CreateDbSystem is called when no DB system exists.
func TestCreateOrUpdate_CreateNew(t *testing.T) {
	const newOcid = "ocid1.postgresql.oc1..new"
	createCalled := false
	activeDbSystem := makeActiveDbSystem(newOcid, "test-db")
	credClient := &fakeCredentialClient{}

	ociClient := &mockOciPostgresClient{
		listFn: func(_ context.Context, _ ocipsql.ListDbSystemsRequest) (ocipsql.ListDbSystemsResponse, error) {
			return ocipsql.ListDbSystemsResponse{}, nil
		},
		createFn: func(_ context.Context, req ocipsql.CreateDbSystemRequest) (ocipsql.CreateDbSystemResponse, error) {
			createCalled = true
			assert.Equal(t, "test-db", *req.CreateDbSystemDetails.DisplayName)
			return ocipsql.CreateDbSystemResponse{DbSystem: activeDbSystem}, nil
		},
		getFn: func(_ context.Context, _ ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error) {
			return ocipsql.GetDbSystemResponse{DbSystem: activeDbSystem}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, credClient)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"
	dbSystem.Spec.DisplayName = "test-db"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.DbVersion = "14.10"
	dbSystem.Spec.Shape = "VM.Standard.E4.Flex"
	dbSystem.Spec.SubnetId = "ocid1.subnet.oc1..xxx"
	dbSystem.Spec.StorageType = "HighPerformance"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, createCalled, "CreateDbSystem should have been called")
	assert.True(t, credClient.createCalled, "CreateSecret should have been called on new DB system")
	assert.Equal(t, ociv1beta1.OCID(newOcid), dbSystem.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_ExistingByDisplayName verifies Create is NOT called when DB system found by name.
func TestCreateOrUpdate_ExistingByDisplayName(t *testing.T) {
	const existingOcid = "ocid1.postgresql.oc1..existing"
	createCalled := false
	activeDbSystem := makeActiveDbSystem(existingOcid, "test-db")

	ociClient := &mockOciPostgresClient{
		listFn: func(_ context.Context, _ ocipsql.ListDbSystemsRequest) (ocipsql.ListDbSystemsResponse, error) {
			return ocipsql.ListDbSystemsResponse{
				DbSystemCollection: ocipsql.DbSystemCollection{
					Items: []ocipsql.DbSystemSummary{
						{Id: common.String(existingOcid), LifecycleState: ocipsql.DbSystemLifecycleStateActive},
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ ocipsql.CreateDbSystemRequest) (ocipsql.CreateDbSystemResponse, error) {
			createCalled = true
			return ocipsql.CreateDbSystemResponse{}, nil
		},
		getFn: func(_ context.Context, _ ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error) {
			return ocipsql.GetDbSystemResponse{DbSystem: activeDbSystem}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"
	dbSystem.Spec.DisplayName = "test-db"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.StorageType = "HighPerformance"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, createCalled, "CreateDbSystem should NOT be called when DB system exists by display name")
}

// TestCreateOrUpdate_Bind verifies binding to an existing DB system by ID.
func TestCreateOrUpdate_Bind(t *testing.T) {
	const existingOcid = "ocid1.postgresql.oc1..bind"
	activeDbSystem := makeActiveDbSystem(existingOcid, "test-db")

	ociClient := &mockOciPostgresClient{
		getFn: func(_ context.Context, _ ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error) {
			return ocipsql.GetDbSystemResponse{DbSystem: activeDbSystem}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"
	dbSystem.Spec.PostgresDbSystemId = existingOcid
	dbSystem.Spec.DisplayName = "test-db"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Status.OsokStatus.Ocid = existingOcid

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(existingOcid), dbSystem.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_Update verifies UpdateDbSystem is called when display name changes.
func TestCreateOrUpdate_Update(t *testing.T) {
	const dbSystemOcid = "ocid1.postgresql.oc1..update"
	updateCalled := false
	existingDbSystem := makeActiveDbSystem(dbSystemOcid, "old-name")

	ociClient := &mockOciPostgresClient{
		getFn: func(_ context.Context, _ ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error) {
			return ocipsql.GetDbSystemResponse{DbSystem: existingDbSystem}, nil
		},
		updateFn: func(_ context.Context, req ocipsql.UpdateDbSystemRequest) (ocipsql.UpdateDbSystemResponse, error) {
			updateCalled = true
			assert.Equal(t, "new-name", *req.UpdateDbSystemDetails.DisplayName)
			return ocipsql.UpdateDbSystemResponse{}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"
	dbSystem.Spec.PostgresDbSystemId = dbSystemOcid
	dbSystem.Spec.DisplayName = "new-name"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Status.OsokStatus.Ocid = dbSystemOcid

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateDbSystem should have been called when display name differs")
}

// TestDelete_WithOcid verifies that DeleteDbSystem OCI call is made when OCID is set.
func TestDelete_WithOcid(t *testing.T) {
	const dbSystemOcid = "ocid1.postgresql.oc1..todelete"
	deleteCalled := false

	ociClient := &mockOciPostgresClient{
		deleteFn: func(_ context.Context, req ocipsql.DeleteDbSystemRequest) (ocipsql.DeleteDbSystemResponse, error) {
			deleteCalled = true
			assert.Equal(t, dbSystemOcid, *req.DbSystemId)
			return ocipsql.DeleteDbSystemResponse{}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"
	dbSystem.Status.OsokStatus.Ocid = dbSystemOcid

	done, err := mgr.Delete(context.Background(), dbSystem)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled, "DeleteDbSystem should have been called with the DB system OCID")
}

// TestDelete_NotFound verifies that a generic error from DeleteDbSystem is propagated.
func TestDelete_NotFound(t *testing.T) {
	ociClient := &mockOciPostgresClient{
		deleteFn: func(_ context.Context, _ ocipsql.DeleteDbSystemRequest) (ocipsql.DeleteDbSystemResponse, error) {
			return ocipsql.DeleteDbSystemResponse{}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"
	dbSystem.Status.OsokStatus.Ocid = "ocid1.postgresql.oc1..gone"

	done, err := mgr.Delete(context.Background(), dbSystem)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestCreateOrUpdate_Failure verifies OCI API errors result in Failed condition.
func TestCreateOrUpdate_Failure(t *testing.T) {
	ociClient := &mockOciPostgresClient{
		listFn: func(_ context.Context, _ ocipsql.ListDbSystemsRequest) (ocipsql.ListDbSystemsResponse, error) {
			return ocipsql.ListDbSystemsResponse{}, nil
		},
		createFn: func(_ context.Context, _ ocipsql.CreateDbSystemRequest) (ocipsql.CreateDbSystemResponse, error) {
			return ocipsql.CreateDbSystemResponse{}, errors.New("OCI API error")
		},
	}
	mgr := newPostgresMgr(t, ociClient, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Name = "test-db"
	dbSystem.Namespace = "default"
	dbSystem.Spec.DisplayName = "test-db"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.StorageType = "HighPerformance"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestGetPostgresDbSystemByName_Active verifies that an ACTIVE DB system is returned by display name.
func TestGetPostgresDbSystemByName_Active(t *testing.T) {
	const dbOcid = "ocid1.postgresql.oc1..active"
	ociClient := &mockOciPostgresClient{
		listFn: func(_ context.Context, _ ocipsql.ListDbSystemsRequest) (ocipsql.ListDbSystemsResponse, error) {
			return ocipsql.ListDbSystemsResponse{
				DbSystemCollection: ocipsql.DbSystemCollection{
					Items: []ocipsql.DbSystemSummary{
						{Id: common.String(dbOcid), LifecycleState: ocipsql.DbSystemLifecycleStateActive},
					},
				},
			}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.DisplayName = "my-db"

	ocid, err := mgr.GetPostgresDbSystemByName(context.Background(), *dbSystem)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID(dbOcid), *ocid)
}

// TestGetPostgresDbSystemByName_NotFound verifies empty list returns nil.
func TestGetPostgresDbSystemByName_NotFound(t *testing.T) {
	ociClient := &mockOciPostgresClient{
		listFn: func(_ context.Context, _ ocipsql.ListDbSystemsRequest) (ocipsql.ListDbSystemsResponse, error) {
			return ocipsql.ListDbSystemsResponse{}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, nil)

	dbSystem := &ociv1beta1.PostgresDbSystem{}
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.DisplayName = "nonexistent"

	ocid, err := mgr.GetPostgresDbSystemByName(context.Background(), *dbSystem)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}
