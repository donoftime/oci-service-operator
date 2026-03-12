/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"
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
	retryableStates := []mysql.DbSystemLifecycleStateEnum{
		mysql.DbSystemLifecycleStateCreating,
		mysql.DbSystemLifecycleStateUpdating,
		mysql.DbSystemLifecycleStateInactive,
	}

	for _, state := range retryableStates {
		t.Run(string(state), func(t *testing.T) {
			mgr := newTestManager(&fakeCredentialClient{})
			mockClient := &mockOciDbSystemClient{
				getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
					dbSystem := makeActiveDbSystem("ocid1.mysqldbsystem.oc1..retry", "test")
					dbSystem.LifecycleState = state
					return mysql.GetDbSystemResponse{DbSystem: dbSystem}, nil
				},
			}
			ExportSetClientForTest(mgr, mockClient)

			dbSystem := &ociv1beta1.MySqlDbSystem{}
			dbSystem.Spec.MySqlDbSystemId = "ocid1.mysqldbsystem.oc1..retry"

			resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
			assert.NoError(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, ociv1beta1.Provisioning, dbSystem.Status.OsokStatus.Conditions[0].Type)
		})
	}
}

func TestPropertyDeleteWaitsForResourceToDisappear(t *testing.T) {
	t.Run("existing resource returns not done", func(t *testing.T) {
		deleteCalled := false
		mgr := newTestManager(&fakeCredentialClient{})
		ExportSetClientForTest(mgr, &mockOciDbSystemClient{
			getFn: func(_ context.Context, req mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
				assert.Equal(t, "ocid1.mysqldbsystem.oc1..delete", *req.DbSystemId)
				return mysql.GetDbSystemResponse{DbSystem: makeActiveDbSystem("ocid1.mysqldbsystem.oc1..delete", "db")}, nil
			},
			deleteFn: func(_ context.Context, req mysql.DeleteDbSystemRequest) (mysql.DeleteDbSystemResponse, error) {
				deleteCalled = true
				assert.Equal(t, "ocid1.mysqldbsystem.oc1..delete", *req.DbSystemId)
				return mysql.DeleteDbSystemResponse{}, nil
			},
		})

		dbSystem := &ociv1beta1.MySqlDbSystem{}
		dbSystem.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..delete"

		done, err := mgr.Delete(context.Background(), dbSystem)
		assert.NoError(t, err)
		assert.False(t, done)
		assert.True(t, deleteCalled)
	})

	t.Run("not found completes delete and removes secret", func(t *testing.T) {
		credClient := &fakeCredentialClient{}
		mgr := newTestManager(credClient)
		ExportSetClientForTest(mgr, &mockOciDbSystemClient{
			getFn: func(_ context.Context, _ mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
				return mysql.GetDbSystemResponse{}, fakeServiceError{
					status: 404,
					code:   "NotFound",
					msg:    "db system not found",
				}
			},
		})

		dbSystem := &ociv1beta1.MySqlDbSystem{}
		dbSystem.Name = "db"
		dbSystem.Namespace = "default"
		dbSystem.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..delete"

		done, err := mgr.Delete(context.Background(), dbSystem)
		assert.NoError(t, err)
		assert.True(t, done)
		assert.True(t, credClient.deleteCalled)
	})
}

func TestPropertyDeleteUsesMySQLWorkRequestProgress(t *testing.T) {
	t.Run("in-progress work request keeps finalizer", func(t *testing.T) {
		mgr := newTestManager(&fakeCredentialClient{})
		ExportSetClientForTest(mgr, &mockOciDbSystemClient{
			listWorkRequestsFn: func(_ context.Context, req mysql.ListWorkRequestsRequest) (mysql.ListWorkRequestsResponse, error) {
				assert.Equal(t, "ocid1.compartment.oc1..mysql", *req.CompartmentId)
				workRequestID := "ocid1.mysqlworkrequest.oc1..inprogress"
				return mysql.ListWorkRequestsResponse{
					Items: []mysql.WorkRequestSummary{{
						Id:            &workRequestID,
						OperationType: mysql.WorkRequestOperationTypeDeleteDbsystem,
					}},
				}, nil
			},
			getWorkRequestFn: func(_ context.Context, req mysql.GetWorkRequestRequest) (mysql.GetWorkRequestResponse, error) {
				assert.Equal(t, "ocid1.mysqlworkrequest.oc1..inprogress", *req.WorkRequestId)
				return mysql.GetWorkRequestResponse{
					WorkRequest: mysql.WorkRequest{
						Status: mysql.WorkRequestOperationStatusInProgress,
						Resources: []mysql.WorkRequestResource{{
							Identifier: common.String("ocid1.mysqldbsystem.oc1..delete"),
						}},
					},
				}, nil
			},
			getFn: func(_ context.Context, req mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
				assert.Equal(t, "ocid1.mysqldbsystem.oc1..delete", *req.DbSystemId)
				dbSystem := makeActiveDbSystem("ocid1.mysqldbsystem.oc1..delete", "db")
				dbSystem.CompartmentId = common.String("ocid1.compartment.oc1..mysql")
				return mysql.GetDbSystemResponse{DbSystem: dbSystem}, nil
			},
			deleteFn: func(_ context.Context, _ mysql.DeleteDbSystemRequest) (mysql.DeleteDbSystemResponse, error) {
				t.Fatal("DeleteDbSystem should not be called when delete work request is in progress")
				return mysql.DeleteDbSystemResponse{}, nil
			},
		})

		dbSystem := &ociv1beta1.MySqlDbSystem{}
		dbSystem.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..delete"

		done, err := mgr.Delete(context.Background(), dbSystem)
		assert.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("succeeded work request completes delete", func(t *testing.T) {
		credClient := &fakeCredentialClient{}
		mgr := newTestManager(credClient)
		ExportSetClientForTest(mgr, &mockOciDbSystemClient{
			listWorkRequestsFn: func(_ context.Context, req mysql.ListWorkRequestsRequest) (mysql.ListWorkRequestsResponse, error) {
				assert.Equal(t, "ocid1.compartment.oc1..mysql", *req.CompartmentId)
				workRequestID := "ocid1.mysqlworkrequest.oc1..done"
				return mysql.ListWorkRequestsResponse{
					Items: []mysql.WorkRequestSummary{{
						Id:            &workRequestID,
						OperationType: mysql.WorkRequestOperationTypeDeleteDbsystem,
					}},
				}, nil
			},
			getWorkRequestFn: func(_ context.Context, req mysql.GetWorkRequestRequest) (mysql.GetWorkRequestResponse, error) {
				assert.Equal(t, "ocid1.mysqlworkrequest.oc1..done", *req.WorkRequestId)
				return mysql.GetWorkRequestResponse{
					WorkRequest: mysql.WorkRequest{
						Status: mysql.WorkRequestOperationStatusSucceeded,
						Resources: []mysql.WorkRequestResource{{
							Identifier: common.String("ocid1.mysqldbsystem.oc1..delete"),
						}},
					},
				}, nil
			},
			getFn: func(_ context.Context, req mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
				assert.Equal(t, "ocid1.mysqldbsystem.oc1..delete", *req.DbSystemId)
				dbSystem := makeActiveDbSystem("ocid1.mysqldbsystem.oc1..delete", "db")
				dbSystem.CompartmentId = common.String("ocid1.compartment.oc1..mysql")
				return mysql.GetDbSystemResponse{DbSystem: dbSystem}, nil
			},
			deleteFn: func(_ context.Context, _ mysql.DeleteDbSystemRequest) (mysql.DeleteDbSystemResponse, error) {
				t.Fatal("DeleteDbSystem should not be called when delete work request already succeeded")
				return mysql.DeleteDbSystemResponse{}, nil
			},
		})

		dbSystem := &ociv1beta1.MySqlDbSystem{}
		dbSystem.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..delete"

		done, err := mgr.Delete(context.Background(), dbSystem)
		assert.NoError(t, err)
		assert.True(t, done)
		assert.True(t, credClient.deleteCalled)
	})

	t.Run("failed work request surfaces an error", func(t *testing.T) {
		mgr := newTestManager(&fakeCredentialClient{})
		ExportSetClientForTest(mgr, &mockOciDbSystemClient{
			listWorkRequestsFn: func(_ context.Context, req mysql.ListWorkRequestsRequest) (mysql.ListWorkRequestsResponse, error) {
				assert.Equal(t, "ocid1.compartment.oc1..mysql", *req.CompartmentId)
				workRequestID := "ocid1.mysqlworkrequest.oc1..failed"
				return mysql.ListWorkRequestsResponse{
					Items: []mysql.WorkRequestSummary{{
						Id:            &workRequestID,
						OperationType: mysql.WorkRequestOperationTypeDeleteDbsystem,
					}},
				}, nil
			},
			getWorkRequestFn: func(_ context.Context, req mysql.GetWorkRequestRequest) (mysql.GetWorkRequestResponse, error) {
				assert.Equal(t, "ocid1.mysqlworkrequest.oc1..failed", *req.WorkRequestId)
				return mysql.GetWorkRequestResponse{
					WorkRequest: mysql.WorkRequest{
						Status: mysql.WorkRequestOperationStatusFailed,
						Resources: []mysql.WorkRequestResource{{
							Identifier: common.String("ocid1.mysqldbsystem.oc1..delete"),
						}},
					},
				}, nil
			},
			getFn: func(_ context.Context, req mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
				assert.Equal(t, "ocid1.mysqldbsystem.oc1..delete", *req.DbSystemId)
				dbSystem := makeActiveDbSystem("ocid1.mysqldbsystem.oc1..delete", "db")
				dbSystem.CompartmentId = common.String("ocid1.compartment.oc1..mysql")
				return mysql.GetDbSystemResponse{DbSystem: dbSystem}, nil
			},
			deleteFn: func(_ context.Context, _ mysql.DeleteDbSystemRequest) (mysql.DeleteDbSystemResponse, error) {
				t.Fatal("DeleteDbSystem should not be reissued while surfacing a failed delete work request")
				return mysql.DeleteDbSystemResponse{}, nil
			},
		})

		dbSystem := &ociv1beta1.MySqlDbSystem{}
		dbSystem.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..delete"

		done, err := mgr.Delete(context.Background(), dbSystem)
		assert.Error(t, err)
		assert.False(t, done)
	})
}

func TestPropertyTransientReadFailuresRequestRequeue(t *testing.T) {
	t.Run("bound get throttling requeues without surfacing an error", func(t *testing.T) {
		mgr := newTestManager(&fakeCredentialClient{})
		ExportSetClientForTest(mgr, &mockOciDbSystemClient{
			getFn: func(_ context.Context, req mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
				assert.Equal(t, "ocid1.mysqldbsystem.oc1..bound", *req.DbSystemId)
				return mysql.GetDbSystemResponse{}, fakeServiceError{
					status: 429,
					code:   "TooManyRequests",
					msg:    "throttled",
				}
			},
		})

		dbSystem := &ociv1beta1.MySqlDbSystem{}
		dbSystem.Spec.MySqlDbSystemId = "ocid1.mysqldbsystem.oc1..bound"
		dbSystem.Spec.DisplayName = "bound-db"

		resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
		assert.NoError(t, err)
		assert.False(t, resp.IsSuccessful)
		assert.True(t, resp.ShouldRequeue)
		assert.Equal(t, ociv1beta1.Provisioning, dbSystem.Status.OsokStatus.Conditions[0].Type)
	})

	t.Run("managed list server error requeues without surfacing an error", func(t *testing.T) {
		mgr := newTestManager(&fakeCredentialClient{})
		ExportSetClientForTest(mgr, &mockOciDbSystemClient{
			listFn: func(_ context.Context, req mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error) {
				assert.Equal(t, "ocid1.compartment.oc1..mysql", *req.CompartmentId)
				return mysql.ListDbSystemsResponse{}, fakeServiceError{
					status: 503,
					code:   "ServiceUnavailable",
					msg:    "temporary outage",
				}
			},
		})

		dbSystem := &ociv1beta1.MySqlDbSystem{}
		dbSystem.Spec.CompartmentId = "ocid1.compartment.oc1..mysql"
		dbSystem.Spec.DisplayName = "managed-db"

		resp, err := mgr.CreateOrUpdate(context.Background(), dbSystem, ctrl.Request{})
		assert.NoError(t, err)
		assert.False(t, resp.IsSuccessful)
		assert.True(t, resp.ShouldRequeue)
		assert.Equal(t, ociv1beta1.Provisioning, dbSystem.Status.OsokStatus.Conditions[0].Type)
	})

	t.Run("delete get throttling requeues without surfacing an error", func(t *testing.T) {
		mgr := newTestManager(&fakeCredentialClient{})
		ExportSetClientForTest(mgr, &mockOciDbSystemClient{
			getFn: func(_ context.Context, req mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error) {
				assert.Equal(t, "ocid1.mysqldbsystem.oc1..delete", *req.DbSystemId)
				return mysql.GetDbSystemResponse{}, fakeServiceError{
					status: 429,
					code:   "TooManyRequests",
					msg:    "throttled",
				}
			},
		})

		dbSystem := &ociv1beta1.MySqlDbSystem{}
		dbSystem.Status.OsokStatus.Ocid = "ocid1.mysqldbsystem.oc1..delete"

		done, err := mgr.Delete(context.Background(), dbSystem)
		assert.NoError(t, err)
		assert.False(t, done)
	})
}
