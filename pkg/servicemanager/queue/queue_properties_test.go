/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue_test

import (
	"context"
	"fmt"
	"testing"
	"testing/quick"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociqueue "github.com/oracle/oci-go-sdk/v65/queue"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestOciQueue_PropertyRetryableStatesRequeue(t *testing.T) {
	states := []ociqueue.QueueLifecycleStateEnum{
		ociqueue.QueueLifecycleStateCreating,
		ociqueue.QueueLifecycleStateUpdating,
		ociqueue.QueueLifecycleStateDeleting,
	}

	property := func(seed uint8) bool {
		state := states[int(seed)%len(states)]
		queueID := fmt.Sprintf("ocid1.queue.oc1..retry-%d", seed)
		credClient := &fakeCredentialClient{}
		fake := &fakeQueueAdminClient{
			getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
				queue := makeActiveQueue(queueID, "retry-queue", "")
				queue.LifecycleState = state
				return ociqueue.GetQueueResponse{Queue: queue}, nil
			},
		}
		mgr := mgrWithFake(credClient, fake)

		q := &ociv1beta1.OciQueue{}
		q.Spec.QueueId = ociv1beta1.OCID(queueID)

		resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
		return err == nil && !resp.IsSuccessful && resp.ShouldRequeue && !credClient.createCalled
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestOciQueue_PropertyBindByIDUsesSpecIDForUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		queueID := fmt.Sprintf("ocid1.queue.oc1..%d", seed)
		var updatedID string
		fake := &fakeQueueAdminClient{
			getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
				return ociqueue.GetQueueResponse{
					Queue: makeActiveQueue(queueID, "old-name", ""),
				}, nil
			},
			updateQueueFn: func(_ context.Context, req ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error) {
				updatedID = *req.QueueId
				return ociqueue.UpdateQueueResponse{}, nil
			},
		}

		mgr := mgrWithFake(&fakeCredentialClient{}, fake)
		q := &ociv1beta1.OciQueue{}
		q.Name = "updated-queue"
		q.Namespace = "default"
		q.Spec.QueueId = ociv1beta1.OCID(queueID)
		q.Spec.DisplayName = "new-name"

		resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
		return err == nil && resp.IsSuccessful && updatedID == queueID
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestOciQueue_PropertyStatusIDUsesTrackedResourceForUpdate(t *testing.T) {
	queueID := "ocid1.queue.oc1..tracked"
	var updatedID string
	fake := &fakeQueueAdminClient{
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			return ociqueue.GetQueueResponse{Queue: makeActiveQueue(queueID, "old-queue", "")}, nil
		},
		updateQueueFn: func(_ context.Context, req ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error) {
			updatedID = *req.QueueId
			return ociqueue.UpdateQueueResponse{}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)
	q := &ociv1beta1.OciQueue{}
	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(queueID)
	q.Spec.DisplayName = "new-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, queueID, updatedID)
}

func TestOciQueue_PropertyCustomEncryptionKeyDriftTriggersUpdate(t *testing.T) {
	queueID := "ocid1.queue.oc1..enc"
	var updatedReq ociqueue.UpdateQueueRequest
	fake := &fakeQueueAdminClient{
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			queue := makeActiveQueue(queueID, "queue", "")
			queue.CustomEncryptionKeyId = common.String("ocid1.key.oc1..old")
			return ociqueue.GetQueueResponse{Queue: queue}, nil
		},
		updateQueueFn: func(_ context.Context, req ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error) {
			updatedReq = req
			return ociqueue.UpdateQueueResponse{}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)
	q := &ociv1beta1.OciQueue{}
	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(queueID)
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	q.Spec.CustomEncryptionKeyId = "ocid1.key.oc1..new"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, queueID, *updatedReq.QueueId)
	assert.NotNil(t, updatedReq.CustomEncryptionKeyId)
	assert.Equal(t, "ocid1.key.oc1..new", *updatedReq.CustomEncryptionKeyId)
}

func TestOciQueue_PropertyTagDriftTriggersUpdate(t *testing.T) {
	property := func(seed uint16) bool {
		queueID := fmt.Sprintf("ocid1.queue.oc1..tags-%d", seed)
		var updatedReq ociqueue.UpdateQueueRequest
		fake := &fakeQueueAdminClient{
			getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
				queue := makeActiveQueue(queueID, "tagged-queue", "")
				queue.FreeformTags = map[string]string{"team": "old"}
				queue.DefinedTags = map[string]map[string]interface{}{"ops": {"env": "dev"}}
				return ociqueue.GetQueueResponse{Queue: queue}, nil
			},
			updateQueueFn: func(_ context.Context, req ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error) {
				updatedReq = req
				return ociqueue.UpdateQueueResponse{}, nil
			},
		}

		mgr := mgrWithFake(&fakeCredentialClient{}, fake)
		q := &ociv1beta1.OciQueue{}
		q.Spec.QueueId = ociv1beta1.OCID(queueID)
		q.Spec.FreeFormTags = map[string]string{"team": "platform"}
		q.Spec.DefinedTags = map[string]ociv1beta1.MapValue{"ops": {"env": "prod"}}

		resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
		return err == nil &&
			resp.IsSuccessful &&
			updatedReq.FreeformTags["team"] == "platform" &&
			updatedReq.DefinedTags["ops"]["env"] == "prod"
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestOciQueue_PropertyCompartmentDriftTriggersMove(t *testing.T) {
	queueID := "ocid1.queue.oc1..move"
	var moved ociqueue.ChangeQueueCompartmentRequest
	fake := &fakeQueueAdminClient{
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			queue := makeActiveQueue(queueID, "queue", "")
			queue.CompartmentId = common.String("ocid1.compartment.oc1..old")
			return ociqueue.GetQueueResponse{Queue: queue}, nil
		},
		changeQueueCompartmentFn: func(_ context.Context, req ociqueue.ChangeQueueCompartmentRequest) (ociqueue.ChangeQueueCompartmentResponse, error) {
			moved = req
			return ociqueue.ChangeQueueCompartmentResponse{}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)
	q := &ociv1beta1.OciQueue{}
	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(queueID)
	q.Spec.CompartmentId = "ocid1.compartment.oc1..new"

	assert.NoError(t, mgr.UpdateQueue(context.Background(), q))
	assert.Equal(t, queueID, *moved.QueueId)
	assert.Equal(t, string(q.Spec.CompartmentId), *moved.CompartmentId)
}

func TestOciQueue_PropertyEmptyCustomEncryptionKeyDoesNotClear(t *testing.T) {
	queueID := "ocid1.queue.oc1..enc-no-clear"
	updateCalled := false
	fake := &fakeQueueAdminClient{
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			queue := makeActiveQueue(queueID, "queue", "")
			queue.CustomEncryptionKeyId = common.String("ocid1.key.oc1..existing")
			return ociqueue.GetQueueResponse{Queue: queue}, nil
		},
		updateQueueFn: func(_ context.Context, _ ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error) {
			updateCalled = true
			return ociqueue.UpdateQueueResponse{}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)
	q := &ociv1beta1.OciQueue{}
	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(queueID)
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	assert.NoError(t, mgr.UpdateQueue(context.Background(), q))
	assert.False(t, updateCalled)
}

func TestOciQueue_PropertyRetentionDriftFailsBeforeMutation(t *testing.T) {
	queueID := "ocid1.queue.oc1..ret"
	moveCalled := false
	updateCalled := false
	fake := &fakeQueueAdminClient{
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			queue := makeActiveQueue(queueID, "queue", "")
			queue.CompartmentId = common.String("ocid1.compartment.oc1..old")
			queue.RetentionInSeconds = common.Int(86400)
			return ociqueue.GetQueueResponse{Queue: queue}, nil
		},
		changeQueueCompartmentFn: func(_ context.Context, _ ociqueue.ChangeQueueCompartmentRequest) (ociqueue.ChangeQueueCompartmentResponse, error) {
			moveCalled = true
			return ociqueue.ChangeQueueCompartmentResponse{}, nil
		},
		updateQueueFn: func(_ context.Context, _ ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error) {
			updateCalled = true
			return ociqueue.UpdateQueueResponse{}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)
	q := &ociv1beta1.OciQueue{}
	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(queueID)
	q.Spec.CompartmentId = "ocid1.compartment.oc1..new"
	q.Spec.RetentionInSeconds = 3600

	err := mgr.UpdateQueue(context.Background(), q)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retentionInSeconds cannot be updated in place")
	assert.False(t, moveCalled)
	assert.False(t, updateCalled)
}
