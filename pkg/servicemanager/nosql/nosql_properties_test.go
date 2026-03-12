/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/nosql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type fakeServiceError struct {
	status int
	code   string
	msg    string
}

func (e fakeServiceError) Error() string          { return e.msg }
func (e fakeServiceError) GetHTTPStatusCode() int { return e.status }
func (e fakeServiceError) GetMessage() string     { return e.msg }
func (e fakeServiceError) GetCode() string        { return e.code }
func (e fakeServiceError) GetOpcRequestID() string {
	return "opc-request-id"
}

func TestPropertyRetryableLifecycleStatesRequeue(t *testing.T) {
	retryableStates := []nosql.TableLifecycleStateEnum{
		nosql.TableLifecycleStateCreating,
		nosql.TableLifecycleStateUpdating,
	}

	for _, state := range retryableStates {
		t.Run(string(state), func(t *testing.T) {
			mock := &mockNosqlClient{
				getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
					tbl := makeActiveTable(testTableOcid, "my-table")
					tbl.LifecycleState = state
					return nosql.GetTableResponse{Table: tbl}, nil
				},
			}
			mgr := newTestManager(mock)

			db := &ociv1beta1.NoSQLDatabase{}
			db.Spec.TableId = ociv1beta1.OCID(testTableOcid)

			resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
			assert.NoError(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, ociv1beta1.Provisioning, db.Status.OsokStatus.Conditions[0].Type)
		})
	}
}

func TestPropertyBindByIDUsesExplicitSpecID(t *testing.T) {
	updateCalled := false
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			assert.Equal(t, testTableOcid, *req.TableNameOrId)
			return nosql.GetTableResponse{Table: makeActiveTable(testTableOcid, "my-table")}, nil
		},
		updateFn: func(_ context.Context, req nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error) {
			updateCalled = true
			assert.Equal(t, testTableOcid, *req.TableNameOrId)
			return nosql.UpdateTableResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.DdlStatement = "ALTER TABLE my-table ADD COLUMN col1 STRING"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
	assert.Equal(t, ociv1beta1.OCID(testTableOcid), db.Status.OsokStatus.Ocid)
}

func TestPropertyDeleteWaitsForNotFound(t *testing.T) {
	t.Run("existing table returns not done", func(t *testing.T) {
		deleteCalled := false
		mock := &mockNosqlClient{
			getFn: func(_ context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
				assert.Equal(t, testTableOcid, *req.TableNameOrId)
				return nosql.GetTableResponse{Table: makeActiveTable(testTableOcid, "my-table")}, nil
			},
			deleteFn: func(_ context.Context, req nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error) {
				deleteCalled = true
				assert.Equal(t, testTableOcid, *req.TableNameOrId)
				return nosql.DeleteTableResponse{}, nil
			},
		}
		mgr := newTestManager(mock)

		db := &ociv1beta1.NoSQLDatabase{}
		db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

		done, err := mgr.Delete(context.Background(), db)
		assert.NoError(t, err)
		assert.False(t, done)
		assert.True(t, deleteCalled)
	})

	t.Run("not found completes delete", func(t *testing.T) {
		mock := &mockNosqlClient{
			getFn: func(_ context.Context, _ nosql.GetTableRequest) (nosql.GetTableResponse, error) {
				return nosql.GetTableResponse{}, fakeServiceError{
					status: 404,
					code:   "NotFound",
					msg:    "table not found",
				}
			},
		}
		mgr := newTestManager(mock)

		db := &ociv1beta1.NoSQLDatabase{}
		db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

		done, err := mgr.Delete(context.Background(), db)
		assert.NoError(t, err)
		assert.True(t, done)
	})
}

func TestPropertyDeleteUsesTableWorkRequestProgress(t *testing.T) {
	t.Run("in-progress work request keeps finalizer", func(t *testing.T) {
		mock := &mockNosqlClient{
			listWorkRequestsFn: func(_ context.Context, req nosql.ListWorkRequestsRequest) (nosql.ListWorkRequestsResponse, error) {
				assert.Equal(t, "ocid1.compartment.oc1..nosql", *req.CompartmentId)
				workRequestID := "ocid1.nosqlworkrequest.oc1..progress"
				return nosql.ListWorkRequestsResponse{
					WorkRequestCollection: nosql.WorkRequestCollection{
						Items: []nosql.WorkRequestSummary{{
							Id:            &workRequestID,
							OperationType: nosql.WorkRequestSummaryOperationTypeDeleteTable,
							Resources: []nosql.WorkRequestResource{{
								Identifier: common.String(testTableOcid),
							}},
						}},
					},
				}, nil
			},
			getWorkRequestFn: func(_ context.Context, req nosql.GetWorkRequestRequest) (nosql.GetWorkRequestResponse, error) {
				assert.Equal(t, "ocid1.nosqlworkrequest.oc1..progress", *req.WorkRequestId)
				return nosql.GetWorkRequestResponse{
					WorkRequest: nosql.WorkRequest{
						Status: nosql.WorkRequestStatusInProgress,
					},
				}, nil
			},
			getFn: func(_ context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
				assert.Equal(t, testTableOcid, *req.TableNameOrId)
				table := makeActiveTable(testTableOcid, "my-table")
				table.CompartmentId = common.String("ocid1.compartment.oc1..nosql")
				return nosql.GetTableResponse{Table: table}, nil
			},
			deleteFn: func(_ context.Context, _ nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error) {
				t.Fatal("DeleteTable should not be called when delete work request is in progress")
				return nosql.DeleteTableResponse{}, nil
			},
		}
		mgr := newTestManager(mock)

		db := &ociv1beta1.NoSQLDatabase{}
		db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

		done, err := mgr.Delete(context.Background(), db)
		assert.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("succeeded work request completes delete", func(t *testing.T) {
		mock := &mockNosqlClient{
			listWorkRequestsFn: func(_ context.Context, req nosql.ListWorkRequestsRequest) (nosql.ListWorkRequestsResponse, error) {
				assert.Equal(t, "ocid1.compartment.oc1..nosql", *req.CompartmentId)
				workRequestID := "ocid1.nosqlworkrequest.oc1..done"
				return nosql.ListWorkRequestsResponse{
					WorkRequestCollection: nosql.WorkRequestCollection{
						Items: []nosql.WorkRequestSummary{{
							Id:            &workRequestID,
							OperationType: nosql.WorkRequestSummaryOperationTypeDeleteTable,
							Resources: []nosql.WorkRequestResource{{
								Identifier: common.String(testTableOcid),
							}},
						}},
					},
				}, nil
			},
			getWorkRequestFn: func(_ context.Context, req nosql.GetWorkRequestRequest) (nosql.GetWorkRequestResponse, error) {
				assert.Equal(t, "ocid1.nosqlworkrequest.oc1..done", *req.WorkRequestId)
				return nosql.GetWorkRequestResponse{
					WorkRequest: nosql.WorkRequest{
						Status: nosql.WorkRequestStatusSucceeded,
					},
				}, nil
			},
			getFn: func(_ context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
				assert.Equal(t, testTableOcid, *req.TableNameOrId)
				table := makeActiveTable(testTableOcid, "my-table")
				table.CompartmentId = common.String("ocid1.compartment.oc1..nosql")
				return nosql.GetTableResponse{Table: table}, nil
			},
			deleteFn: func(_ context.Context, _ nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error) {
				t.Fatal("DeleteTable should not be called when delete work request already succeeded")
				return nosql.DeleteTableResponse{}, nil
			},
		}
		mgr := newTestManager(mock)

		db := &ociv1beta1.NoSQLDatabase{}
		db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

		done, err := mgr.Delete(context.Background(), db)
		assert.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("failed work request surfaces an error", func(t *testing.T) {
		mock := &mockNosqlClient{
			listWorkRequestsFn: func(_ context.Context, req nosql.ListWorkRequestsRequest) (nosql.ListWorkRequestsResponse, error) {
				assert.Equal(t, "ocid1.compartment.oc1..nosql", *req.CompartmentId)
				workRequestID := "ocid1.nosqlworkrequest.oc1..failed"
				return nosql.ListWorkRequestsResponse{
					WorkRequestCollection: nosql.WorkRequestCollection{
						Items: []nosql.WorkRequestSummary{{
							Id:            &workRequestID,
							OperationType: nosql.WorkRequestSummaryOperationTypeDeleteTable,
							Resources: []nosql.WorkRequestResource{{
								Identifier: common.String(testTableOcid),
							}},
						}},
					},
				}, nil
			},
			getWorkRequestFn: func(_ context.Context, req nosql.GetWorkRequestRequest) (nosql.GetWorkRequestResponse, error) {
				assert.Equal(t, "ocid1.nosqlworkrequest.oc1..failed", *req.WorkRequestId)
				return nosql.GetWorkRequestResponse{
					WorkRequest: nosql.WorkRequest{
						Status: nosql.WorkRequestStatusFailed,
					},
				}, nil
			},
			getFn: func(_ context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
				assert.Equal(t, testTableOcid, *req.TableNameOrId)
				table := makeActiveTable(testTableOcid, "my-table")
				table.CompartmentId = common.String("ocid1.compartment.oc1..nosql")
				return nosql.GetTableResponse{Table: table}, nil
			},
			deleteFn: func(_ context.Context, _ nosql.DeleteTableRequest) (nosql.DeleteTableResponse, error) {
				t.Fatal("DeleteTable should not be reissued while surfacing a failed delete work request")
				return nosql.DeleteTableResponse{}, nil
			},
		}
		mgr := newTestManager(mock)

		db := &ociv1beta1.NoSQLDatabase{}
		db.Status.OsokStatus.Ocid = ociv1beta1.OCID(testTableOcid)

		done, err := mgr.Delete(context.Background(), db)
		assert.Error(t, err)
		assert.False(t, done)
	})
}

func TestPropertyUpdateSkipsNoSQLNoOpChanges(t *testing.T) {
	updateCalled := false
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			assert.Equal(t, testTableOcid, *req.TableNameOrId)
			table := makeActiveTable(testTableOcid, "my-table")
			table.DdlStatement = common.String("CREATE TABLE my-table (id STRING, PRIMARY KEY(id))")
			table.TableLimits = &nosql.TableLimits{
				MaxReadUnits:    common.Int(10),
				MaxWriteUnits:   common.Int(20),
				MaxStorageInGBs: common.Int(5),
			}
			table.FreeformTags = map[string]string{"team": "platform"}
			table.DefinedTags = map[string]map[string]interface{}{
				"ops": {"env": "dev"},
			}
			return nosql.GetTableResponse{Table: table}, nil
		},
		updateFn: func(_ context.Context, _ nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error) {
			updateCalled = true
			return nosql.UpdateTableResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.CompartmentId = "ocid1.compartment.oc1..nosql"
	db.Spec.DdlStatement = "CREATE TABLE my-table (id STRING, PRIMARY KEY(id))"
	db.Spec.TableLimits = &ociv1beta1.NoSQLDatabaseTableLimits{
		MaxReadUnits:    10,
		MaxWriteUnits:   20,
		MaxStorageInGBs: 5,
	}
	db.Spec.FreeFormTags = map[string]string{"team": "platform"}
	db.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
		"ops": {"env": "dev"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, updateCalled)
}

func TestPropertyUpdateCapturesNoSQLTagDrift(t *testing.T) {
	updateCalled := false
	mock := &mockNosqlClient{
		getFn: func(_ context.Context, req nosql.GetTableRequest) (nosql.GetTableResponse, error) {
			assert.Equal(t, testTableOcid, *req.TableNameOrId)
			table := makeActiveTable(testTableOcid, "my-table")
			table.FreeformTags = map[string]string{"team": "legacy"}
			table.DefinedTags = map[string]map[string]interface{}{
				"ops": {"env": "dev"},
			}
			return nosql.GetTableResponse{Table: table}, nil
		},
		updateFn: func(_ context.Context, req nosql.UpdateTableRequest) (nosql.UpdateTableResponse, error) {
			updateCalled = true
			assert.Equal(t, map[string]string{"team": "platform"}, req.FreeformTags)
			assert.Equal(t, map[string]map[string]interface{}{"ops": {"env": "prod"}}, req.DefinedTags)
			return nosql.UpdateTableResponse{}, nil
		},
	}
	mgr := newTestManager(mock)

	db := &ociv1beta1.NoSQLDatabase{}
	db.Spec.TableId = ociv1beta1.OCID(testTableOcid)
	db.Spec.FreeFormTags = map[string]string{"team": "platform"}
	db.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
		"ops": {"env": "prod"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}
