/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/nosql"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct {
	createCalled bool
	deleteCalled bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	return true, nil
}

// mockNosqlClient implements NosqlClientInterface for unit testing.
type mockNosqlClient struct {
	createFn func(context.Context, nosql.CreateTableRequest) (nosql.CreateTableResponse, error)
	getFn    func(context.Context, nosql.GetTableRequest) (nosql.GetTableResponse, error)
	listFn   func(context.Context, nosql.ListTablesRequest) (nosql.ListTablesResponse, error)
	updateFn func(context.Context, nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error)
	deleteFn func(context.Context, nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error)
}

func (m *mockNosqlClient) CreateTable(ctx context.Context, req nosql.CreateTableRequest) (nosql.CreateTableResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nosql.CreateTableResponse{}, nil
}

func (m *mockNosqlClient) GetTable(ctx context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return nosql.GetTableResponse{}, nil
}

func (m *mockNosqlClient) ListTables(ctx context.Context, req nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return nosql.ListTablesResponse{}, nil
}

func (m *mockNosqlClient) UpdateTable(ctx context.Context, req nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nosql.UpdateTableResponse{}, nil
}

func (m *mockNosqlClient) DeleteTable(ctx context.Context, req nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, req)
	}
	return nosql.DeleteTableResponse{}, nil
}

// newTestManager creates a manager with a mock OCI client injected.
func newTestManager(mock *mockNosqlClient) *NoSQLDatabaseServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewNoSQLDatabaseServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		&fakeCredentialClient{}, nil, log)
	ExportSetClientForTest(mgr, mock)
	return mgr
}

// makeActiveTable returns a nosql.Table with ACTIVE lifecycle state.
func makeActiveTable(id, name string) nosql.Table {
	return nosql.Table{
		Id:             common.String(id),
		Name:           common.String(name),
		CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState: nosql.TableLifecycleStateActive,
	}
}

// makeTableSummary returns a TableSummary with the given state.
func makeTableSummary(id string, state nosql.TableLifecycleStateEnum) nosql.TableSummary {
	return nosql.TableSummary{
		Id:             common.String(id),
		CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState: state,
	}
}

// listResponse wraps items into a ListTablesResponse.
func listResponse(items ...nosql.TableSummary) nosql.ListTablesResponse {
	return nosql.ListTablesResponse{
		TableCollection: nosql.TableCollection{Items: items},
	}
}

const testTableOcid = "ocid1.nosqltable.oc1..aaaatest"

// ---------------------------------------------------------------------------
// Existing tests (preserved)
// ---------------------------------------------------------------------------

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewNoSQLDatabaseServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Name = "test-table"
	db.Namespace = "default"

	done, err := mgr.Delete(context.Background(), db)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called for NoSQL table")
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from a NoSQLDatabase object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewNoSQLDatabaseServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Status.OsokStatus.Ocid = "ocid1.nosqltable.oc1..xxx"

	status, err := mgr.GetCrdStatus(db)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.nosqltable.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewNoSQLDatabaseServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-NoSQLDatabase objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewNoSQLDatabaseServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// GetTableOcid tests
// ---------------------------------------------------------------------------

// TestGetTableOcid_ActiveTable verifies an ACTIVE table is found and its OCID returned.
func TestGetTableOcid_ActiveTable(t *testing.T) {
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateActive)), nil
		},
	}
	mgr := newTestManager(mock)

	db := ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetTableOcid(context.Background(), db)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID(testTableOcid), *ocid)
}

// TestGetTableOcid_CreatingTable verifies a CREATING table is returned (not nil).
func TestGetTableOcid_CreatingTable(t *testing.T) {
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateCreating)), nil
		},
	}
	mgr := newTestManager(mock)

	db := ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetTableOcid(context.Background(), db)
	assert.NoError(t, err)
	assert.NotNil(t, ocid, "CREATING table should return its OCID")
}

// TestGetTableOcid_NotFound verifies an empty list returns nil OCID without error.
func TestGetTableOcid_NotFound(t *testing.T) {
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return nosql.ListTablesResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "missing-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetTableOcid(context.Background(), db)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}

// TestGetTableOcid_DeletedState verifies a DELETED table is treated as not found.
func TestGetTableOcid_DeletedState(t *testing.T) {
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateDeleted)), nil
		},
	}
	mgr := newTestManager(mock)

	db := ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "deleted-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetTableOcid(context.Background(), db)
	assert.NoError(t, err)
	assert.Nil(t, ocid, "DELETED table should not be returned as found")
}

// TestGetTableOcid_OciError verifies errors from ListTables are propagated.
func TestGetTableOcid_OciError(t *testing.T) {
	apiErr := errors.New("OCI ListTables error")
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return nosql.ListTablesResponse{}, apiErr
		},
	}
	mgr := newTestManager(mock)

	db := ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetTableOcid(context.Background(), db)
	assert.Error(t, err)
	assert.Nil(t, ocid)
}

// ---------------------------------------------------------------------------
// CreateOrUpdate — no TableId (create / lookup by name) path
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_NoTableId_GetOcidError verifies GetTableOcid errors are propagated.
func TestCreateOrUpdate_NoTableId_GetOcidError(t *testing.T) {
	apiErr := errors.New("list error")
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return nosql.ListTablesResponse{}, apiErr
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.DdlStatement = "CREATE TABLE my-table (id INTEGER, PRIMARY KEY(id))"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_NoTableId_CreateTable_OnDemand tests the create path with no TableLimits.
func TestCreateOrUpdate_NoTableId_CreateTable_OnDemand(t *testing.T) {
	listCount := 0
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			listCount++
			if listCount == 1 {
				// First call: table not found
				return nosql.ListTablesResponse{}, nil
			}
			// Second call: table now visible after create
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateActive)), nil
		},
		createFn: func(_ context.Context, req nosql.CreateTableRequest) (nosql.CreateTableResponse, error) {
			assert.Nil(t, req.CreateTableDetails.TableLimits, "on-demand table should have no limits")
			return nosql.CreateTableResponse{}, nil
		},
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			tbl := makeActiveTable(testTableOcid, "my-table")
			return nosql.GetTableResponse{Table: tbl}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.DdlStatement = "CREATE TABLE my-table (id INTEGER, PRIMARY KEY(id))"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, 2, listCount, "ListTables should be called twice (before and after create)")
}

// TestCreateOrUpdate_NoTableId_CreateTable_Provisioned tests create with TableLimits (provisioned mode).
func TestCreateOrUpdate_NoTableId_CreateTable_Provisioned(t *testing.T) {
	listCount := 0
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			listCount++
			if listCount == 1 {
				return nosql.ListTablesResponse{}, nil
			}
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateActive)), nil
		},
		createFn: func(_ context.Context, req nosql.CreateTableRequest) (nosql.CreateTableResponse, error) {
			assert.NotNil(t, req.CreateTableDetails.TableLimits, "provisioned table should have limits")
			assert.Equal(t, 10, *req.CreateTableDetails.TableLimits.MaxReadUnits)
			assert.Equal(t, 10, *req.CreateTableDetails.TableLimits.MaxWriteUnits)
			assert.Equal(t, 5, *req.CreateTableDetails.TableLimits.MaxStorageInGBs)
			return nosql.CreateTableResponse{}, nil
		},
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			tbl := makeActiveTable(testTableOcid, "my-table")
			return nosql.GetTableResponse{Table: tbl}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.DdlStatement = "CREATE TABLE my-table (id INTEGER, PRIMARY KEY(id))"
	db.Spec.TableLimits = &ociv1beta1.NoSQLDatabaseTableLimits{
		MaxReadUnits:    10,
		MaxWriteUnits:   10,
		MaxStorageInGBs: 5,
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_NoTableId_CreateTable_Requeue verifies requeue when table not yet visible after create.
func TestCreateOrUpdate_NoTableId_CreateTable_Requeue(t *testing.T) {
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			// Both calls return not found: first (pre-create lookup) and second (post-create poll)
			return nosql.ListTablesResponse{}, nil
		},
		createFn: func(_ context.Context, _ nosql.CreateTableRequest) (nosql.CreateTableResponse, error) {
			return nosql.CreateTableResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.DdlStatement = "CREATE TABLE my-table (id INTEGER, PRIMARY KEY(id))"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.True(t, resp.ShouldRequeue, "should requeue when table not visible after create")
	assert.Equal(t, int64(30), int64(resp.RequeueDuration.Seconds()))
}

// TestCreateOrUpdate_NoTableId_CreateTableError verifies non-BadRequest errors are propagated.
func TestCreateOrUpdate_NoTableId_CreateTableError(t *testing.T) {
	apiErr := errors.New("OCI CreateTable failed")
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return nosql.ListTablesResponse{}, nil
		},
		createFn: func(_ context.Context, _ nosql.CreateTableRequest) (nosql.CreateTableResponse, error) {
			return nosql.CreateTableResponse{}, apiErr
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.DdlStatement = "CREATE TABLE my-table (id INTEGER, PRIMARY KEY(id))"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.Error(t, err)
	assert.Equal(t, apiErr, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_NoTableId_GetTableAfterCreateError verifies GetTable errors are propagated after create.
func TestCreateOrUpdate_NoTableId_GetTableAfterCreateError(t *testing.T) {
	listCount := 0
	getErr := errors.New("GetTable failed")
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			listCount++
			if listCount == 1 {
				return nosql.ListTablesResponse{}, nil
			}
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateActive)), nil
		},
		createFn: func(_ context.Context, _ nosql.CreateTableRequest) (nosql.CreateTableResponse, error) {
			return nosql.CreateTableResponse{}, nil
		},
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			return nosql.GetTableResponse{}, getErr
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.DdlStatement = "CREATE TABLE my-table (id INTEGER, PRIMARY KEY(id))"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.Error(t, err)
	assert.Equal(t, getErr, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_NoTableId_ExistingTable verifies an existing table found by name is used as-is.
func TestCreateOrUpdate_NoTableId_ExistingTable(t *testing.T) {
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateActive)), nil
		},
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			tbl := makeActiveTable(testTableOcid, "my-table")
			return nosql.GetTableResponse{Table: tbl}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(testTableOcid), db.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_NoTableId_ExistingTable_GetTableError verifies GetTable errors are propagated when found by name.
func TestCreateOrUpdate_NoTableId_ExistingTable_GetTableError(t *testing.T) {
	getErr := errors.New("GetTable API error")
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateActive)), nil
		},
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			return nosql.GetTableResponse{}, getErr
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_NoTableId_TableFailed verifies a FAILED table returns IsSuccessful: false.
func TestCreateOrUpdate_NoTableId_TableFailed(t *testing.T) {
	mock := &mockNosqlClient{
		listFn: func(_ context.Context, _ nosql.ListTablesRequest) (nosql.ListTablesResponse, error) {
			return listResponse(makeTableSummary(testTableOcid, nosql.TableLifecycleStateActive)), nil
		},
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			tbl := nosql.Table{
				Id:             common.String(testTableOcid),
				Name:           common.String("my-table"),
				CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
				LifecycleState: nosql.TableLifecycleStateFailed,
			}
			return nosql.GetTableResponse{Table: tbl}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.Name = "my-table"
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "FAILED table should result in unsuccessful response")
}

// ---------------------------------------------------------------------------
// CreateOrUpdate — with TableId (bind by OCID) path
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_WithTableId_Success verifies binding to an existing table by OCID.
func TestCreateOrUpdate_WithTableId_Success(t *testing.T) {
	updateCalled := false
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			assert.Equal(t, testTableOcid, *req.TableNameOrId)
			tbl := makeActiveTable(testTableOcid, "my-table")
			return nosql.GetTableResponse{Table: tbl}, nil
		},
		updateFn: func(_ context.Context, req nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error) {
			updateCalled = true
			return nosql.UpdateTableResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.DdlStatement = "ALTER TABLE my-table ADD COLUMN col1 STRING"
	db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateTable should be called when DDL is provided")
}

// TestCreateOrUpdate_WithTableId_UpdateNoOp verifies no UpdateTable call when no DDL or limits.
func TestCreateOrUpdate_WithTableId_UpdateNoOp(t *testing.T) {
	updateCalled := false
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			tbl := makeActiveTable(testTableOcid, "my-table")
			return nosql.GetTableResponse{Table: tbl}, nil
		},
		updateFn: func(_ context.Context, _ nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error) {
			updateCalled = true
			return nosql.UpdateTableResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	// No DDL statement and no TableLimits → update is a no-op

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, updateCalled, "UpdateTable should not be called when there's nothing to update")
}

// TestCreateOrUpdate_WithTableId_GetTableError verifies GetTable errors are propagated when binding by ID.
func TestCreateOrUpdate_WithTableId_GetTableError(t *testing.T) {
	getErr := errors.New("table not found")
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			return nosql.GetTableResponse{}, getErr
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.Error(t, err)
	assert.Equal(t, getErr, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_WithTableId_UpdateError verifies UpdateTable errors are propagated.
func TestCreateOrUpdate_WithTableId_UpdateError(t *testing.T) {
	updateErr := errors.New("UpdateTable API error")
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			tbl := makeActiveTable(testTableOcid, "my-table")
			return nosql.GetTableResponse{Table: tbl}, nil
		},
		updateFn: func(_ context.Context, _ nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error) {
			return nosql.UpdateTableResponse{}, updateErr
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.DdlStatement = "ALTER TABLE my-table ADD COLUMN col1 STRING"
	db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.Error(t, err)
	assert.Equal(t, updateErr, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_WithTableId_TableFailed verifies FAILED table returns unsuccessful when bound by ID.
func TestCreateOrUpdate_WithTableId_TableFailed(t *testing.T) {
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			tbl := nosql.Table{
				Id:             common.String(testTableOcid),
				Name:           common.String("my-table"),
				CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
				LifecycleState: nosql.TableLifecycleStateFailed,
			}
			return nosql.GetTableResponse{Table: tbl}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	// No DDL/limits to keep update as no-op

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "FAILED table should return unsuccessful")
}

// TestCreateOrUpdate_WithTableId_UpdateLimits verifies TableLimits change triggers UpdateTable call.
func TestCreateOrUpdate_WithTableId_UpdateLimits(t *testing.T) {
	updateCalled := false
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			tbl := makeActiveTable(testTableOcid, "my-table")
			return nosql.GetTableResponse{Table: tbl}, nil
		},
		updateFn: func(_ context.Context, req nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error) {
			updateCalled = true
			assert.NotNil(t, req.UpdateTableDetails.TableLimits)
			assert.Equal(t, 20, *req.UpdateTableDetails.TableLimits.MaxReadUnits)
			return nosql.UpdateTableResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	db.Spec.TableLimits = &ociv1beta1.NoSQLDatabaseTableLimits{
		MaxReadUnits:    20,
		MaxWriteUnits:   20,
		MaxStorageInGBs: 10,
	}
	db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateTable should be called when TableLimits changed")
}

// ---------------------------------------------------------------------------
// Delete tests with mock client
// ---------------------------------------------------------------------------

// TestDelete_WithOcid_Success verifies DeleteTable is called with the correct OCID.
func TestDelete_WithOcid_Success(t *testing.T) {
	deleteCalled := false
	mock := &mockNosqlClient{
		deleteFn: func(_ context.Context, req nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error) {
			deleteCalled = true
			assert.Equal(t, testTableOcid, *req.TableNameOrId)
			assert.True(t, *req.IsIfExists)
			return nosql.DeleteTableResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

	done, err := mgr.Delete(context.Background(), db)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// TestDelete_WithOcid_Error verifies DeleteTable errors are propagated.
func TestDelete_WithOcid_Error(t *testing.T) {
	deleteErr := errors.New("DeleteTable API error")
	mock := &mockNosqlClient{
		deleteFn: func(_ context.Context, _ nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error) {
			return nosql.DeleteTableResponse{}, deleteErr
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

	done, err := mgr.Delete(context.Background(), db)
	assert.Error(t, err)
	assert.Equal(t, deleteErr, err)
	assert.False(t, done)
}
