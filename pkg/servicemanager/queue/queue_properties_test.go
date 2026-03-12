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

	ociqueue "github.com/oracle/oci-go-sdk/v65/queue"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
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
