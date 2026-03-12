/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compute_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestPropertyComputeInstancePendingStatesRequestRequeue(t *testing.T) {
	for _, state := range []core.InstanceLifecycleStateEnum{
		core.InstanceLifecycleStateProvisioning,
		core.InstanceLifecycleStateStarting,
		core.InstanceLifecycleStateStopping,
	} {
		t.Run(string(state), func(t *testing.T) {
			ociClient := &fakeComputeClient{
				listFn: func(_ context.Context, _ core.ListInstancesRequest) (core.ListInstancesResponse, error) {
					return core.ListInstancesResponse{
						Items: []core.Instance{
							{
								Id:             common.String("ocid1.instance.oc1..pending"),
								DisplayName:    common.String("pending-instance"),
								LifecycleState: state,
							},
						},
					}, nil
				},
				getFn: func(_ context.Context, _ core.GetInstanceRequest) (core.GetInstanceResponse, error) {
					return core.GetInstanceResponse{
						Instance: core.Instance{
							Id:             common.String("ocid1.instance.oc1..pending"),
							DisplayName:    common.String("pending-instance"),
							LifecycleState: state,
						},
					}, nil
				},
			}
			mgr := newTestManager(ociClient)
			ci := makeComputeInstanceSpec("pending-instance")

			resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
			assert.NoError(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
		})
	}
}

func TestPropertyComputeInstanceBindByIDUsesSpecIDWhenStatusIsEmpty(t *testing.T) {
	var updatedID string
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             req.InstanceId,
					DisplayName:    common.String("old-bound-instance"),
					LifecycleState: core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
		updateFn: func(_ context.Context, req core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error) {
			updatedID = *req.InstanceId
			return core.UpdateInstanceResponse{}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("new-bound-instance")
	ci.Spec.ComputeInstanceId = ociv1beta1.OCID("ocid1.instance.oc1..bind")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, string(ci.Spec.ComputeInstanceId), updatedID)
}

func TestPropertyComputeInstanceDeleteWaitsForConfirmedDisappearance(t *testing.T) {
	ociClient := &fakeComputeClient{
		terminateFn: func(_ context.Context, _ core.TerminateInstanceRequest) (core.TerminateInstanceResponse, error) {
			return core.TerminateInstanceResponse{}, nil
		},
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             req.InstanceId,
					DisplayName:    common.String("still-there"),
					LifecycleState: core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("still-there")
	ci.Status.OsokStatus.Ocid = "ocid1.instance.oc1..delete"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.False(t, done)
}
