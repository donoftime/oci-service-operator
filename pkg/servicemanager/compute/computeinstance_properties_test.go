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
								ImageId:        common.String("ocid1.image.oc1..xxx"),
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
							ImageId:        common.String("ocid1.image.oc1..xxx"),
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
					ImageId:        common.String("ocid1.image.oc1..xxx"),
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

func TestPropertyComputeInstanceStatusIDUsesTrackedResourceForUpdates(t *testing.T) {
	var updatedID string
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             req.InstanceId,
					DisplayName:    common.String("old-instance"),
					CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
					ImageId:        common.String("ocid1.image.oc1..xxx"),
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
	ci := makeComputeInstanceSpec("new-instance")
	ci.Status.OsokStatus.Ocid = "ocid1.instance.oc1..tracked"

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "ocid1.instance.oc1..tracked", updatedID)
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

func TestPropertyComputeInstanceTagDriftTriggersUpdate(t *testing.T) {
	var updated core.UpdateInstanceRequest
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             req.InstanceId,
					DisplayName:    common.String("tagged-instance"),
					ImageId:        common.String("ocid1.image.oc1..xxx"),
					LifecycleState: core.InstanceLifecycleStateRunning,
					FreeformTags:   map[string]string{"team": "old"},
					DefinedTags: map[string]map[string]interface{}{
						"ops": {"env": "dev"},
					},
				},
			}, nil
		},
		updateFn: func(_ context.Context, req core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error) {
			updated = req
			return core.UpdateInstanceResponse{}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("tagged-instance")
	ci.Spec.ComputeInstanceId = ociv1beta1.OCID("ocid1.instance.oc1..tags")
	ci.Spec.FreeFormTags = map[string]string{"team": "platform"}
	ci.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
		"ops": {"env": "prod"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, map[string]string{"team": "platform"}, updated.FreeformTags)
	assert.Equal(t, map[string]map[string]interface{}{
		"ops": {"env": "prod"},
	}, updated.DefinedTags)
}

func TestPropertyComputeInstanceMatchingTagsSkipUpdate(t *testing.T) {
	updateCalled := false
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             req.InstanceId,
					DisplayName:    common.String("tagged-instance"),
					ImageId:        common.String("ocid1.image.oc1..xxx"),
					Shape:          common.String("VM.Standard.E4.Flex"),
					LifecycleState: core.InstanceLifecycleStateRunning,
					FreeformTags:   map[string]string{"team": "platform"},
					ShapeConfig:    &core.InstanceShapeConfig{Ocpus: common.Float32(1), MemoryInGBs: common.Float32(0)},
					DefinedTags: map[string]map[string]interface{}{
						"ops": {"env": "prod"},
					},
				},
			}, nil
		},
		updateFn: func(_ context.Context, req core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error) {
			updateCalled = true
			return core.UpdateInstanceResponse{}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("tagged-instance")
	ci.Spec.ComputeInstanceId = ociv1beta1.OCID("ocid1.instance.oc1..tags")
	ci.Spec.FreeFormTags = map[string]string{"team": "platform"}
	ci.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
		"ops": {"env": "prod"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, updateCalled)
}

func TestPropertyComputeInstanceCompartmentDriftTriggersMove(t *testing.T) {
	var moved core.ChangeInstanceCompartmentRequest
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             req.InstanceId,
					DisplayName:    common.String("instance"),
					CompartmentId:  common.String("ocid1.compartment.oc1..old"),
					ImageId:        common.String("ocid1.image.oc1..xxx"),
					LifecycleState: core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
		changeCompartmentFn: func(_ context.Context, req core.ChangeInstanceCompartmentRequest) (core.ChangeInstanceCompartmentResponse, error) {
			moved = req
			return core.ChangeInstanceCompartmentResponse{}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("instance")
	ci.Status.OsokStatus.Ocid = "ocid1.instance.oc1..move"
	ci.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	assert.NoError(t, mgr.UpdateInstance(context.Background(), ci))
	assert.Equal(t, "ocid1.instance.oc1..move", *moved.InstanceId)
	assert.Equal(t, string(ci.Spec.CompartmentId), *moved.CompartmentId)
}

func TestPropertyComputeInstanceShapeDriftTriggersUpdate(t *testing.T) {
	var updated core.UpdateInstanceRequest
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             req.InstanceId,
					DisplayName:    common.String("shape-instance"),
					ImageId:        common.String("ocid1.image.oc1..xxx"),
					Shape:          common.String("VM.Standard.E4.Flex"),
					ShapeConfig:    &core.InstanceShapeConfig{Ocpus: common.Float32(1), MemoryInGBs: common.Float32(16)},
					LifecycleState: core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
		updateFn: func(_ context.Context, req core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error) {
			updated = req
			return core.UpdateInstanceResponse{}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("shape-instance")
	ci.Spec.ComputeInstanceId = "ocid1.instance.oc1..shape"
	ci.Spec.Shape = "VM.Standard3.Flex"
	ci.Spec.ShapeConfig = &ociv1beta1.ComputeInstanceShapeConfig{Ocpus: 2, MemoryInGBs: 32}

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, "VM.Standard3.Flex", *updated.Shape)
	assert.NotNil(t, updated.ShapeConfig)
	assert.Equal(t, float32(2), *updated.ShapeConfig.Ocpus)
	assert.Equal(t, float32(32), *updated.ShapeConfig.MemoryInGBs)
}

func TestPropertyComputeInstanceImmutableAvailabilityDomainFailsBeforeMutation(t *testing.T) {
	updateCalled := false
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:                 req.InstanceId,
					DisplayName:        common.String("immutable-instance"),
					AvailabilityDomain: common.String("AD-1"),
					ImageId:            common.String("ocid1.image.oc1..xxx"),
					LifecycleState:     core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
		updateFn: func(_ context.Context, _ core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error) {
			updateCalled = true
			return core.UpdateInstanceResponse{}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("immutable-instance")
	ci.Spec.ComputeInstanceId = "ocid1.instance.oc1..immutable"
	ci.Spec.AvailabilityDomain = "AD-2"

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "availabilityDomain cannot be updated in place")
	assert.False(t, updateCalled)
}

func TestPropertyComputeInstanceImmutableImageDriftFailsBeforeMutation(t *testing.T) {
	updateCalled := false
	moveCalled := false
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             req.InstanceId,
					DisplayName:    common.String("image-instance"),
					CompartmentId:  common.String("ocid1.compartment.oc1..old"),
					ImageId:        common.String("ocid1.image.oc1..old"),
					LifecycleState: core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
		changeCompartmentFn: func(_ context.Context, _ core.ChangeInstanceCompartmentRequest) (core.ChangeInstanceCompartmentResponse, error) {
			moveCalled = true
			return core.ChangeInstanceCompartmentResponse{}, nil
		},
		updateFn: func(_ context.Context, _ core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error) {
			updateCalled = true
			return core.UpdateInstanceResponse{}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("image-instance")
	ci.Spec.ComputeInstanceId = "ocid1.instance.oc1..image"
	ci.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	ci.Spec.ImageId = "ocid1.image.oc1..new"

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
	assert.Contains(t, err.Error(), "imageId cannot be updated in place")
	assert.False(t, moveCalled)
	assert.False(t, updateCalled)
}
