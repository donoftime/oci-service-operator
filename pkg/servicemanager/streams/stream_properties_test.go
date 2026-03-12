/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams_test

import (
	"context"
	"fmt"
	"testing"
	"testing/quick"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestStreamServiceManager_PropertyRetryableStatesRequeue(t *testing.T) {
	states := []string{"CREATING", "UPDATING", "DELETING"}

	property := func(seed uint8) bool {
		state := states[int(seed)%len(states)]
		streamID := fmt.Sprintf("ocid1.stream.oc1..retry-%d", seed)
		credClient := &fakeCredentialClient{}
		mockClient := &mockStreamAdminClient{
			getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
				stream := makeActiveStream(streamID, "retry-stream")
				stream.LifecycleState = streaming.StreamLifecycleStateEnum(state)
				return streaming.GetStreamResponse{Stream: stream}, nil
			},
		}
		mgr := makeTestManager(credClient, mockClient)

		stream := &ociv1beta1.Stream{}
		stream.Spec.StreamId = ociv1beta1.OCID(streamID)

		resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
		return err == nil && !resp.IsSuccessful && resp.ShouldRequeue && !credClient.createCalled
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestStreamServiceManager_PropertyUpdateByNameUsesResolvedID(t *testing.T) {
	property := func(seed uint16) bool {
		streamID := fmt.Sprintf("ocid1.stream.oc1..%d", seed)
		var updatedID string
		mockClient := &mockStreamAdminClient{
			listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
				return streaming.ListStreamsResponse{
					Items: []streaming.StreamSummary{
						{Id: common.String(streamID), LifecycleState: streaming.StreamSummaryLifecycleStateActive},
					},
				}, nil
			},
			getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
				return streaming.GetStreamResponse{Stream: makeActiveStream(streamID, "named-stream")}, nil
			},
			updateStreamFn: func(_ context.Context, req streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error) {
				updatedID = *req.StreamId
				return streaming.UpdateStreamResponse{}, nil
			},
		}
		mgr := makeTestManager(&fakeCredentialClient{}, mockClient)

		stream := &ociv1beta1.Stream{}
		stream.Name = "named-stream"
		stream.Namespace = "default"
		stream.Spec.Name = "named-stream"
		stream.Spec.Partitions = 1
		stream.Spec.RetentionInHours = 24
		stream.Spec.FreeFormTags = map[string]string{"env": "prop"}

		resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
		return err == nil && resp.IsSuccessful && updatedID == streamID
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}
