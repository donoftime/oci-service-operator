/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicontainerinstances "github.com/oracle/oci-go-sdk/v65/containerinstances"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestPropertyContainerInstancePendingStatesRequestRequeue(t *testing.T) {
	for _, state := range []ocicontainerinstances.ContainerInstanceLifecycleStateEnum{
		ocicontainerinstances.ContainerInstanceLifecycleStateCreating,
		ocicontainerinstances.ContainerInstanceLifecycleStateUpdating,
	} {
		t.Run(string(state), func(t *testing.T) {
			ociClient := &fakeOciClient{
				listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
					return ocicontainerinstances.ListContainerInstancesResponse{
						ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
							Items: []ocicontainerinstances.ContainerInstanceSummary{
								{
									Id:             common.String("ocid1.containerinstance.oc1..pending"),
									DisplayName:    common.String("pending-ci"),
									LifecycleState: state,
								},
							},
						},
					}, nil
				},
				getFn: func(_ context.Context, _ ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
					return ocicontainerinstances.GetContainerInstanceResponse{
						ContainerInstance: ocicontainerinstances.ContainerInstance{
							Id:             common.String("ocid1.containerinstance.oc1..pending"),
							DisplayName:    common.String("pending-ci"),
							LifecycleState: state,
						},
					}, nil
				},
			}
			mgr := newTestManager(ociClient)
			ci := makeContainerInstanceSpec("pending-ci")

			resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
			assert.NoError(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
		})
	}
}

func TestPropertyContainerInstanceBindByIDUsesSpecIDWhenStatusIsEmpty(t *testing.T) {
	var updatedID string
	ociClient := &fakeOciClient{
		getFn: func(_ context.Context, req ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             req.ContainerInstanceId,
					DisplayName:    common.String("old-bound-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
		updateFn: func(_ context.Context, req ocicontainerinstances.UpdateContainerInstanceRequest) (ocicontainerinstances.UpdateContainerInstanceResponse, error) {
			updatedID = *req.ContainerInstanceId
			return ocicontainerinstances.UpdateContainerInstanceResponse{}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("new-bound-ci")
	ci.Spec.ContainerInstanceId = ociv1beta1.OCID("ocid1.containerinstance.oc1..bind")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, string(ci.Spec.ContainerInstanceId), updatedID)
}

func TestPropertyContainerInstanceDeleteWaitsForConfirmedDisappearance(t *testing.T) {
	ociClient := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error) {
			return ocicontainerinstances.DeleteContainerInstanceResponse{}, nil
		},
		getFn: func(_ context.Context, req ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             req.ContainerInstanceId,
					DisplayName:    common.String("still-there"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("still-there")
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..delete"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.False(t, done)
}
