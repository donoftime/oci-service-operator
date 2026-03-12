/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package nosql_test

import (
	"context"
	"testing"

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
