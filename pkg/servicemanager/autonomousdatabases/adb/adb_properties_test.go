/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/autonomousdatabases/adb"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

type propertyServiceError struct {
	status int
	code   string
	msg    string
}

func (e propertyServiceError) Error() string          { return e.msg }
func (e propertyServiceError) GetHTTPStatusCode() int { return e.status }
func (e propertyServiceError) GetMessage() string     { return e.msg }
func (e propertyServiceError) GetCode() string        { return e.code }
func (e propertyServiceError) GetOpcRequestID() string {
	return "opc-request-id"
}

func TestPropertyRetryableLifecycleStatesRequeue(t *testing.T) {
	retryableStates := []database.AutonomousDatabaseLifecycleStateEnum{
		database.AutonomousDatabaseLifecycleStateProvisioning,
		database.AutonomousDatabaseLifecycleStateUpdating,
		database.AutonomousDatabaseLifecycleStateStarting,
		database.AutonomousDatabaseLifecycleStateStopping,
		database.AutonomousDatabaseLifecycleStateMaintenanceInProgress,
	}

	for _, state := range retryableStates {
		t.Run(string(state), func(t *testing.T) {
			mgr := newTestManager(&fakeCredentialClient{})
			mockClient := &mockOciDbClient{
				getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
					adb := makeActiveAdb("ocid1.autonomousdatabase.oc1..retry", "test-adb")
					adb.LifecycleState = state
					return database.GetAutonomousDatabaseResponse{AutonomousDatabase: adb}, nil
				},
			}
			ExportSetClientForTest(mgr, mockClient)

			adb := &ociv1beta1.AutonomousDatabases{}
			adb.Spec.AdbId = "ocid1.autonomousdatabase.oc1..retry"

			resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
			assert.NoError(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
			assert.Equal(t, ociv1beta1.Provisioning, adb.Status.OsokStatus.Conditions[0].Type)
		})
	}
}

func TestPropertyExplicitFalseBooleansTriggerUpdate(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	updateCalled := false
	var captured database.UpdateAutonomousDatabaseRequest

	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			adb := makeActiveAdb("ocid1.autonomousdatabase.oc1..bool", "test-adb")
			adb.IsAutoScalingEnabled = common.Bool(true)
			adb.IsFreeTier = common.Bool(true)
			return database.GetAutonomousDatabaseResponse{AutonomousDatabase: adb}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			captured = req
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = "ocid1.autonomousdatabase.oc1..bool"
	adb.Spec.SetIsAutoScalingEnabled(false)
	adb.Spec.SetIsFreeTier(false)

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
	assert.NotNil(t, captured.IsAutoScalingEnabled)
	assert.False(t, *captured.IsAutoScalingEnabled)
	assert.NotNil(t, captured.IsFreeTier)
	assert.False(t, *captured.IsFreeTier)
}

func TestPropertyOmittedFalseBooleansDoNotTriggerUpdate(t *testing.T) {
	mgr := newTestManager(&fakeCredentialClient{})
	updateCalled := false

	mockClient := &mockOciDbClient{
		getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
			adb := makeActiveAdb("ocid1.autonomousdatabase.oc1..bool", "test-adb")
			adb.IsAutoScalingEnabled = common.Bool(true)
			adb.IsFreeTier = common.Bool(true)
			return database.GetAutonomousDatabaseResponse{AutonomousDatabase: adb}, nil
		},
		updateFn: func(_ context.Context, req database.UpdateAutonomousDatabaseRequest) (database.UpdateAutonomousDatabaseResponse, error) {
			updateCalled = true
			return database.UpdateAutonomousDatabaseResponse{}, nil
		},
	}
	ExportSetClientForTest(mgr, mockClient)

	adb := &ociv1beta1.AutonomousDatabases{}
	adb.Spec.AdbId = "ocid1.autonomousdatabase.oc1..bool"

	resp, err := mgr.CreateOrUpdate(context.Background(), adb, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, updateCalled)
}

func TestPropertySpecJSONTracksExplicitADBBooleans(t *testing.T) {
	var spec ociv1beta1.AutonomousDatabasesSpec

	err := json.Unmarshal([]byte(`{"isAutoScalingEnabled":false,"isFreeTier":false}`), &spec)
	assert.NoError(t, err)
	assert.True(t, spec.HasExplicitIsAutoScalingEnabled())
	assert.True(t, spec.HasExplicitIsFreeTier())
	assert.False(t, spec.IsAutoScalingEnabled)
	assert.False(t, spec.IsFreeTier)
}

func TestPropertyDeleteWaitsForResourceToDisappear(t *testing.T) {
	t.Run("existing resource keeps finalizer", func(t *testing.T) {
		deleteCalled := false
		mgr := newTestManager(&fakeCredentialClient{})
		mockClient := &mockOciDbClient{
			getFn: func(_ context.Context, req database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
				assert.Equal(t, "ocid1.autonomousdatabase.oc1..delete", *req.AutonomousDatabaseId)
				return database.GetAutonomousDatabaseResponse{
					AutonomousDatabase: makeActiveAdb("ocid1.autonomousdatabase.oc1..delete", "test-adb"),
				}, nil
			},
			deleteFn: func(_ context.Context, req database.DeleteAutonomousDatabaseRequest) (database.DeleteAutonomousDatabaseResponse, error) {
				deleteCalled = true
				assert.Equal(t, "ocid1.autonomousdatabase.oc1..delete", *req.AutonomousDatabaseId)
				return database.DeleteAutonomousDatabaseResponse{}, nil
			},
		}
		ExportSetClientForTest(mgr, mockClient)

		adb := &ociv1beta1.AutonomousDatabases{}
		adb.Status.OsokStatus.Ocid = "ocid1.autonomousdatabase.oc1..delete"

		done, err := mgr.Delete(context.Background(), adb)
		assert.NoError(t, err)
		assert.False(t, done)
		assert.True(t, deleteCalled)
	})

	t.Run("not found completes delete and removes wallet secret", func(t *testing.T) {
		credClient := &fakeCredentialClient{}
		mgr := newTestManager(credClient)
		mockClient := &mockOciDbClient{
			getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
				return database.GetAutonomousDatabaseResponse{}, propertyServiceError{
					status: 404,
					code:   "NotFound",
					msg:    "adb not found",
				}
			},
		}
		ExportSetClientForTest(mgr, mockClient)

		adb := &ociv1beta1.AutonomousDatabases{}
		adb.Name = "adb"
		adb.Namespace = "default"
		adb.Status.OsokStatus.Ocid = "ocid1.autonomousdatabase.oc1..delete"

		done, err := mgr.Delete(context.Background(), adb)
		assert.NoError(t, err)
		assert.True(t, done)
		assert.False(t, credClient.deleteCalled)
	})

	t.Run("legacy unowned wallet secret is preserved", func(t *testing.T) {
		credClient := &fakeCredentialClient{
			getSecretFn: func(_ context.Context, name, _ string) (map[string][]byte, error) {
				assert.Equal(t, "adb-wallet", name)
				return map[string][]byte{"tnsnames.ora": []byte("legacy")}, nil
			},
			deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
				t.Fatal("DeleteSecret should not be called for an unowned legacy wallet secret")
				return false, nil
			},
		}
		mgr := newTestManager(credClient)
		mockClient := &mockOciDbClient{
			getFn: func(_ context.Context, _ database.GetAutonomousDatabaseRequest) (database.GetAutonomousDatabaseResponse, error) {
				return database.GetAutonomousDatabaseResponse{}, propertyServiceError{
					status: 404,
					code:   "NotFound",
					msg:    "adb not found",
				}
			},
		}
		ExportSetClientForTest(mgr, mockClient)

		adb := &ociv1beta1.AutonomousDatabases{}
		adb.Name = "adb"
		adb.Namespace = "default"
		adb.Status.OsokStatus.Ocid = "ocid1.autonomousdatabase.oc1..delete"

		done, err := mgr.Delete(context.Background(), adb)
		assert.NoError(t, err)
		assert.True(t, done)
		assert.False(t, credClient.deleteCalled)
	})
}
