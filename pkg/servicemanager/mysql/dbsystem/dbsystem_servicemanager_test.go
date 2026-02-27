/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// mockOciDbSystemClient implements MySQLDbSystemClientInterface for testing.
type mockOciDbSystemClient struct {
	createFn func(context.Context, mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error)
	listFn   func(context.Context, mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error)
	getFn    func(context.Context, mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error)
	updateFn func(context.Context, mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error)
}

func (m *mockOciDbSystemClient) CreateDbSystem(ctx context.Context, req mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return mysql.CreateDbSystemResponse{}, nil
}

func (m *mockOciDbSystemClient) ListDbSystems(ctx context.Context, req mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return mysql.ListDbSystemsResponse{}, nil
}

func (m *mockOciDbSystemClient) GetDbSystem(ctx context.Context, req mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return mysql.GetDbSystemResponse{}, nil
}

func (m *mockOciDbSystemClient) UpdateDbSystem(ctx context.Context, req mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return mysql.UpdateDbSystemResponse{}, nil
}

// makeActiveDbSystem returns a minimal mysql.DbSystem for mock responses.
func makeActiveDbSystem(id, displayName string) mysql.DbSystem {
	port := 3306
	portX := 33060
	desc := "test description"
	hostname := "mysql.example.com"
	ip := "10.0.0.1"
	az := "AD-1"
	fd := "FAULT-DOMAIN-1"
	cfgId := "ocid1.mysqlconfiguration.oc1..xxx"
	return mysql.DbSystem{
		Id:                 common.String(id),
		DisplayName:        common.String(displayName),
		Description:        &desc,
		LifecycleState:     mysql.DbSystemLifecycleStateActive,
		Port:               &port,
		PortX:              &portX,
		HostnameLabel:      &hostname,
		IpAddress:          &ip,
		AvailabilityDomain: &az,
		FaultDomain:        &fd,
		ConfigurationId:    &cfgId,
		CompartmentId:      common.String("ocid1.compartment.oc1..xxx"),
	}
}

func newTestManager(credClient *fakeCredentialClient) *DbSystemServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return NewDbSystemServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)
}

// --- Structural tests (no OCI calls) ---

// TestGetCrdStatus_Happy verifies status is returned from a MySqlDbSystem object.
func TestGetCrdStatus_Happy(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..xxx"

	status, err := mgr.GetCrdStatus(dbSystem)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.mysqldbsystem.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert the type assertion for MySqlDbSystem")
}

// TestDelete_NoOcid verifies deletion is a no-op (Delete always returns true, nil).
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := newTestManager(credClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "test-dbsystem"
	dbSystem.Namespace = "default"

	done, err := mgr.Delete(context.Background(), dbSystem)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-MySqlDbSystem objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// --- getCredentialMap tests ---

// TestGetCredentialMap verifies the secret credential map is built correctly from a DbSystem.
func TestGetCredentialMap(t *testing.T) {
	dbSystem := makeActiveDbSystem("ocid1.mysqldbsystem.oc1..xxx", "test-dbsystem")
	credMap, err := GetCredentialMapForTest(dbSystem)

	assert.NoError(t, err)
	assert.Equal(t, "10.0.0.1", string(credMap["PrivateIPAddress"]))
	assert.Equal(t, "mysql.example.com", string(credMap["InternalFQDN"]))
	assert.Equal(t, "AD-1", string(credMap["AvailabilityDomain"]))
	assert.Equal(t, "FAULT-DOMAIN-1", string(credMap["FaultDomain"]))
	assert.Equal(t, "3306", string(credMap["MySQLPort"]))
	assert.Equal(t, "33060", string(credMap["MySQLXProtocolPort"]))
	assert.Contains(t, credMap, "Endpoints")
}

// TestGetCredentialMap_NilHostname verifies nil HostnameLabel is handled (empty InternalFQDN).
func TestGetCredentialMap_NilHostname(t *testing.T) {
	port := 3306
	portX := 33060
	ip := "10.0.0.2"
	az := "AD-2"
	fd := "FAULT-DOMAIN-2"
	dbSystem := mysql.DbSystem{
		Id:                 common.String("ocid1.mysqldbsystem.oc1..yyy"),
		IpAddress:          &ip,
		HostnameLabel:      nil, // nil — should produce empty InternalFQDN
		AvailabilityDomain: &az,
		FaultDomain:        &fd,
		Port:               &port,
		PortX:              &portX,
	}
	credMap, err := GetCredentialMapForTest(dbSystem)
	assert.NoError(t, err)
	assert.Equal(t, "", string(credMap["InternalFQDN"]))
}

// --- Mock-based tests (require OCI client injection) ---

// TestCreateOrUpdate_BindExistingDbSystem_NothingToUpdate verifies that when MySqlDbSystemId
// is specified and fields match, no update is issued and the manager reports success.
func TestCreateOrUpdate_BindExistingDbSystem_NothingToUpdate(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return true, nil
		},
	})

	dbSystemId := "ocid1.mysqldbsystem.oc1..xxx"
	mockClient := &mockOciDbSystemClient{
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: makeActiveDbSystem(dbSystemId, "test-dbsystem"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "test-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.MySqlDbSystemId = ociv1beta1.OCID(dbSystemId)
	dbSystem.Spec.DisplayName = "test-dbsystem" // same as returned — no update needed

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(dbSystemId), dbSystem.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_BindExistingDbSystem_UpdateNeeded verifies that when the display name
// differs from the spec, an update is issued.
func TestCreateOrUpdate_BindExistingDbSystem_UpdateNeeded(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return true, nil
		},
	})

	dbSystemId := "ocid1.mysqldbsystem.oc1..yyy"
	updateCalled := false

	mockClient := &mockOciDbSystemClient{
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: makeActiveDbSystem(dbSystemId, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error) {
			updateCalled = true
			return mysql.UpdateDbSystemResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "test-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.MySqlDbSystemId = ociv1beta1.OCID(dbSystemId)
	dbSystem.Spec.DisplayName = "new-name" // differs from "old-name" → triggers update

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateDbSystem should be called")
}

// TestCreateOrUpdate_OciGetError verifies that an OCI GetDbSystem error propagates.
func TestCreateOrUpdate_OciGetError(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	mockClient := &mockOciDbSystemClient{
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{}, errors.New("OCI API error")
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Spec.MySqlDbSystemId = "ocid1.mysqldbsystem.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_FindExisting verifies that when no MySqlDbSystemId is in the spec,
// ListDbSystems finds an existing system by display name.
func TestCreateOrUpdate_FindExisting(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return true, nil
		},
	})

	dbSystemId := "ocid1.mysqldbsystem.oc1..found"

	mockClient := &mockOciDbSystemClient{
		listFn: func(_ context.Context, _ mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
			return mysql.ListDbSystemsResponse{
				Items: []mysql.DbSystemSummary{
					{
						Id:             common.String(dbSystemId),
						LifecycleState: mysql.DbSystemLifecycleStateActive,
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: makeActiveDbSystem(dbSystemId, "my-dbsystem"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "my-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.DisplayName = "my-dbsystem"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(dbSystemId), dbSystem.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_ListError verifies that a ListDbSystems error is returned
// when no MySqlDbSystemId is in the spec.
func TestCreateOrUpdate_ListError(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})

	mockClient := &mockOciDbSystemClient{
		listFn: func(_ context.Context, _ mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
			return mysql.ListDbSystemsResponse{}, errors.New("list API error")
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Spec.DisplayName = "my-dbsystem"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreateNew verifies that when no MySqlDbSystemId is in the spec and
// no existing system is found, a new DB system is created.
func TestCreateOrUpdate_CreateNew(t *testing.T) {
	newDbSystemId := "ocid1.mysqldbsystem.oc1..new"

	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, name, _ string) (map[string][]byte, error) {
			if name == "admin-username-secret" {
				return map[string][]byte{"username": []byte("admin")}, nil
			}
			return map[string][]byte{"password": []byte("secret123")}, nil
		},
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return true, nil
		},
	}
	mgr := newTestManager(credClient)

	createCalled := false
	mockClient := &mockOciDbSystemClient{
		listFn: func(_ context.Context, _ mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
			return mysql.ListDbSystemsResponse{}, nil // empty — no existing system
		},
		createFn: func(_ context.Context, _ mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error) {
			createCalled = true
			return mysql.CreateDbSystemResponse{
				DbSystem: mysql.DbSystem{
					Id: common.String(newDbSystemId),
				},
			}, nil
		},
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: makeActiveDbSystem(newDbSystemId, "new-dbsystem"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "new-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.DisplayName = "new-dbsystem"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.AdminUsername.Secret.SecretName = "admin-username-secret"
	dbSystem.Spec.AdminPassword.Secret.SecretName = "admin-password-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default"}})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, createCalled, "CreateDbSystem should be called")
	assert.Equal(t, ociv1beta1.OCID(newDbSystemId), dbSystem.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_CreateNew_MissingUsernameKey verifies that a missing "username" key
// in the admin username secret causes an error before any OCI call.
func TestCreateOrUpdate_CreateNew_MissingUsernameKey(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return map[string][]byte{"wrongkey": []byte("value")}, nil
		},
	}
	mgr := newTestManager(credClient)

	mockClient := &mockOciDbSystemClient{
		listFn: func(_ context.Context, _ mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
			return mysql.ListDbSystemsResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Spec.DisplayName = "my-dbsystem"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.AdminUsername.Secret.SecretName = "admin-username-secret"
	dbSystem.Spec.AdminPassword.Secret.SecretName = "admin-password-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username key")
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreateNew_GetSecretError verifies that a GetSecret error
// when fetching the admin credentials is propagated correctly.
func TestCreateOrUpdate_CreateNew_GetSecretError(t *testing.T) {
	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, _, _ string) (map[string][]byte, error) {
			return nil, errors.New("secret not found")
		},
	}
	mgr := newTestManager(credClient)

	mockClient := &mockOciDbSystemClient{
		listFn: func(_ context.Context, _ mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
			return mysql.ListDbSystemsResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Spec.DisplayName = "my-dbsystem"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.AdminUsername.Secret.SecretName = "admin-username-secret"
	dbSystem.Spec.AdminPassword.Secret.SecretName = "admin-password-secret"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_LifecycleFailed verifies that when the OCI response reports
// FAILED lifecycle state, the manager status is updated accordingly.
func TestCreateOrUpdate_LifecycleFailed(t *testing.T) {
	dbSystemId := "ocid1.mysqldbsystem.oc1..failed"

	failedDbSystem := makeActiveDbSystem(dbSystemId, "failed-dbsystem")
	failedDbSystem.LifecycleState = mysql.DbSystemLifecycleStateFailed

	mgr := newTestManager(&fakeCredentialClient{})

	mockClient := &mockOciDbSystemClient{
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: failedDbSystem,
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "failed-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.MySqlDbSystemId = ociv1beta1.OCID(dbSystemId)

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	// Status should reflect the FAILED lifecycle
	assert.Equal(t, ociv1beta1.OCID(dbSystemId), dbSystem.Status.OsokStatus.Ocid)
}

// ---------------------------------------------------------------------------
// DeleteMySqlDbSystem stub coverage
// ---------------------------------------------------------------------------

// TestDeleteMySqlDbSystem verifies DeleteMySqlDbSystem returns empty string and no error.
func TestDeleteMySqlDbSystem(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	ocid, err := mgr.DeleteMySqlDbSystem()
	assert.NoError(t, err)
	assert.Equal(t, "", ocid)
}

// ---------------------------------------------------------------------------
// deleteFromSecret coverage
// ---------------------------------------------------------------------------

// TestDeleteFromSecret verifies deleteFromSecret calls CredentialClient.DeleteSecret.
func TestDeleteFromSecret(t *testing.T) {
	deleteCalled := false
	credClient := &fakeCredentialClient{
		deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
			deleteCalled = true
			return true, nil
		},
	}
	mgr := newTestManager(credClient)

	ok, err := ExportDeleteFromSecretForTest(mgr, context.Background(), "default", "my-dbsystem")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, deleteCalled, "DeleteSecret should be called by deleteFromSecret")
}

// ---------------------------------------------------------------------------
// getDbSystemRetryPolicy predicate and nextDuration coverage
// ---------------------------------------------------------------------------

// TestDbSystemRetryPolicy_Creating verifies shouldRetry returns true when state is CREATING.
func TestDbSystemRetryPolicy_Creating(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportGetDbSystemRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: mysql.GetDbSystemResponse{
			DbSystem: mysql.DbSystem{LifecycleState: "CREATING"},
		},
	}
	assert.True(t, shouldRetry(resp))
}

// TestDbSystemRetryPolicy_Active verifies shouldRetry returns false when state is ACTIVE.
func TestDbSystemRetryPolicy_Active(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportGetDbSystemRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: mysql.GetDbSystemResponse{
			DbSystem: mysql.DbSystem{LifecycleState: "ACTIVE"},
		},
	}
	assert.False(t, shouldRetry(resp))
}

// TestDbSystemRetryPolicy_NonResponse verifies shouldRetry returns true when type assertion fails.
func TestDbSystemRetryPolicy_NonResponse(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	shouldRetry := ExportGetDbSystemRetryPredicate(mgr)

	resp := common.OCIOperationResponse{} // nil Response — type assertion fails → true
	assert.True(t, shouldRetry(resp))
}

// TestDbSystemRetryNextDuration verifies nextDuration returns 1 minute.
func TestDbSystemRetryNextDuration(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	nextDuration := ExportGetDbSystemNextDuration(mgr)

	resp := common.OCIOperationResponse{AttemptNumber: 1}
	assert.Equal(t, 1*time.Minute, nextDuration(resp))
}

// ---------------------------------------------------------------------------
// CreateDbSystem optional field coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_CreateNew_WithOptionalFields verifies that optional fields
// (Description, Port, PortX, ConfigurationId, IpAddress, HostnameLabel, MysqlVersion)
// are included in the CreateDbSystem request when set.
func TestCreateOrUpdate_CreateNew_WithOptionalFields(t *testing.T) {
	newDbSystemId := "ocid1.mysqldbsystem.oc1..opts"

	credClient := &fakeCredentialClient{
		getSecretFn: func(_ context.Context, name, _ string) (map[string][]byte, error) {
			if name == "admin-username-secret" {
				return map[string][]byte{"username": []byte("admin")}, nil
			}
			return map[string][]byte{"password": []byte("secret123")}, nil
		},
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return true, nil
		},
	}
	mgr := newTestManager(credClient)

	var capturedReq mysql.CreateDbSystemRequest
	mockClient := &mockOciDbSystemClient{
		listFn: func(_ context.Context, _ mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
			return mysql.ListDbSystemsResponse{}, nil
		},
		createFn: func(_ context.Context, req mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error) {
			capturedReq = req
			return mysql.CreateDbSystemResponse{
				DbSystem: mysql.DbSystem{Id: common.String(newDbSystemId)},
			}, nil
		},
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: makeActiveDbSystem(newDbSystemId, "opts-dbsystem"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "opts-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.DisplayName = "opts-dbsystem"
	dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	dbSystem.Spec.AdminUsername.Secret.SecretName = "admin-username-secret"
	dbSystem.Spec.AdminPassword.Secret.SecretName = "admin-password-secret"
	dbSystem.Spec.Description = "test description"
	dbSystem.Spec.Port = 3307
	dbSystem.Spec.PortX = 33070
	dbSystem.Spec.ConfigurationId.Id = "ocid1.mysqlconfiguration.oc1..cfg"
	dbSystem.Spec.IpAddress = "10.0.0.5"
	dbSystem.Spec.HostnameLabel = "mysql-host"
	dbSystem.Spec.MysqlVersion = "8.0"

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default"}})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)

	d := capturedReq.CreateDbSystemDetails
	assert.Equal(t, common.String("test description"), d.Description)
	assert.Equal(t, common.Int(3307), d.Port)
	assert.Equal(t, common.Int(33070), d.PortX)
	assert.Equal(t, common.String("ocid1.mysqlconfiguration.oc1..cfg"), d.ConfigurationId)
	assert.Equal(t, common.String("10.0.0.5"), d.IpAddress)
	assert.Equal(t, common.String("mysql-host"), d.HostnameLabel)
	assert.Equal(t, common.String("8.0"), d.MysqlVersion)
}

// ---------------------------------------------------------------------------
// UpdateMySqlDbSystem Description + ConfigurationId coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BindExisting_DescriptionAndConfigIdChange verifies that Description
// and ConfigurationId changes are included in the update request.
func TestCreateOrUpdate_BindExisting_DescriptionAndConfigIdChange(t *testing.T) {
	dbSystemId := "ocid1.mysqldbsystem.oc1..descfg"
	var capturedUpdate mysql.UpdateDbSystemRequest

	mgr := newTestManager(&fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return true, nil
		},
	})
	mockClient := &mockOciDbSystemClient{
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: makeActiveDbSystem(dbSystemId, "test-dbsystem"),
			}, nil
		},
		updateFn: func(_ context.Context, req mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error) {
			capturedUpdate = req
			return mysql.UpdateDbSystemResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "test-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.MySqlDbSystemId = ociv1beta1.OCID(dbSystemId)
	dbSystem.Spec.DisplayName = "test-dbsystem" // same — no update on this field
	dbSystem.Spec.Description = "new description" // differs from "test description"
	dbSystem.Spec.ConfigurationId.Id = "ocid1.mysqlconfiguration.oc1..new" // differs from current

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, common.String("new description"), capturedUpdate.UpdateDbSystemDetails.Description)
	assert.Equal(t, common.String("ocid1.mysqlconfiguration.oc1..new"), capturedUpdate.UpdateDbSystemDetails.ConfigurationId)
}

// ---------------------------------------------------------------------------
// isValidUpdate DefinedTags coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BindExisting_DefinedTagsChange verifies that DefinedTags changes
// trigger an update call.
func TestCreateOrUpdate_BindExisting_DefinedTagsChange(t *testing.T) {
	dbSystemId := "ocid1.mysqldbsystem.oc1..deftag"
	updateCalled := false

	mgr := newTestManager(&fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return true, nil
		},
	})
	mockClient := &mockOciDbSystemClient{
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: makeActiveDbSystem(dbSystemId, "test-dbsystem"),
			}, nil
		},
		updateFn: func(_ context.Context, _ mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error) {
			updateCalled = true
			return mysql.UpdateDbSystemResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "test-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.MySqlDbSystemId = ociv1beta1.OCID(dbSystemId)
	dbSystem.Spec.DisplayName = "test-dbsystem" // same
	dbSystem.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
		"ns1": {"key1": "val1"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateDbSystem should be called when DefinedTags differ")
}

// ---------------------------------------------------------------------------
// Non-ACTIVE/FAILED lifecycle state coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_LifecycleProvisioning verifies that a non-ACTIVE, non-FAILED
// lifecycle state results in an error response indicating the system is not yet ready.
func TestCreateOrUpdate_LifecycleProvisioning(t *testing.T) {
	dbSystemId := "ocid1.mysqldbsystem.oc1..prov"

	creatingDbSystem := makeActiveDbSystem(dbSystemId, "prov-dbsystem")
	creatingDbSystem.LifecycleState = mysql.DbSystemLifecycleStateCreating

	mgr := newTestManager(&fakeCredentialClient{})
	mockClient := &mockOciDbSystemClient{
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{DbSystem: creatingDbSystem}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "prov-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.MySqlDbSystemId = ociv1beta1.OCID(dbSystemId)

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// CreatedAt != nil branch coverage
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BindExisting_CreatedAtNonNil verifies that when CreatedAt is already
// set and the bind succeeds, the CreatedAt timestamp is refreshed.
func TestCreateOrUpdate_BindExisting_CreatedAtNonNil(t *testing.T) {
	dbSystemId := "ocid1.mysqldbsystem.oc1..creat"

	mgr := newTestManager(&fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return true, nil
		},
	})
	mockClient := &mockOciDbSystemClient{
		getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
			return mysql.GetDbSystemResponse{
				DbSystem: makeActiveDbSystem(dbSystemId, "test-dbsystem"),
			}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	dbSystem := &ociv1beta1.MySqlDbSystem{}
	dbSystem.Name = "test-dbsystem"
	dbSystem.Namespace = "default"
	dbSystem.Spec.MySqlDbSystemId = ociv1beta1.OCID(dbSystemId)
	dbSystem.Spec.DisplayName = "test-dbsystem"
	// Pre-set CreatedAt so the "if CreatedAt != nil" branch is taken.
	ts := metav1.NewTime(time.Now())
	dbSystem.Status.OsokStatus.CreatedAt = &ts

	resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}
