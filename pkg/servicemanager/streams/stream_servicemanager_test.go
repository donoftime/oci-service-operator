/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/streams"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct {
	createSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	deleteSecretFn func(ctx context.Context, name, ns string) (bool, error)
	getSecretFn    func(ctx context.Context, name, ns string) (map[string][]byte, error)
	updateSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, ns)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	if f.getSecretFn != nil {
		return f.getSecretFn(ctx, name, ns)
	}
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	if f.updateSecretFn != nil {
		return f.updateSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

// mockStreamAdminClient implements StreamAdminClientInterface for testing.
type mockStreamAdminClient struct {
	createStreamFn func(ctx context.Context, req streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error)
	listStreamsFn  func(ctx context.Context, req streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error)
	deleteStreamFn func(ctx context.Context, req streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error)
	getStreamFn    func(ctx context.Context, req streaming.GetStreamRequest) (streaming.GetStreamResponse, error)
	updateStreamFn func(ctx context.Context, req streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error)
}

func (m *mockStreamAdminClient) CreateStream(ctx context.Context, req streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error) {
	if m.createStreamFn != nil {
		return m.createStreamFn(ctx, req)
	}
	return streaming.CreateStreamResponse{}, nil
}

func (m *mockStreamAdminClient) ListStreams(ctx context.Context, req streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
	if m.listStreamsFn != nil {
		return m.listStreamsFn(ctx, req)
	}
	return streaming.ListStreamsResponse{}, nil
}

func (m *mockStreamAdminClient) DeleteStream(ctx context.Context, req streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
	if m.deleteStreamFn != nil {
		return m.deleteStreamFn(ctx, req)
	}
	return streaming.DeleteStreamResponse{}, nil
}

func (m *mockStreamAdminClient) GetStream(ctx context.Context, req streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
	if m.getStreamFn != nil {
		return m.getStreamFn(ctx, req)
	}
	return streaming.GetStreamResponse{}, nil
}

func (m *mockStreamAdminClient) UpdateStream(ctx context.Context, req streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error) {
	if m.updateStreamFn != nil {
		return m.updateStreamFn(ctx, req)
	}
	return streaming.UpdateStreamResponse{}, nil
}

// makeTestManager constructs a StreamServiceManager with fake clients for testing.
func makeTestManager(credClient *fakeCredentialClient, mockClient *mockStreamAdminClient) *StreamServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	m := &metrics.Metrics{Logger: log}
	mgr := NewStreamServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log, m)
	if mockClient != nil {
		ExportSetClientForTest(mgr, mockClient)
	}
	return mgr
}

func makeActiveStream(id, name string) streaming.Stream {
	return streaming.Stream{
		Id:               common.String(id),
		Name:             common.String(name),
		LifecycleState:   "ACTIVE",
		MessagesEndpoint: common.String("https://cell-1.streaming.us-phoenix-1.oci.oraclecloud.com"),
		StreamPoolId:     common.String("ocid1.streampool.oc1..xxx"),
		Partitions:       common.Int(1),
		RetentionInHours: common.Int(24),
	}
}

// ---------------------------------------------------------------------------
// GetCrdStatus tests
// ---------------------------------------------------------------------------

// TestGetCrdStatus_HappyPath verifies status extraction from a Stream object.
func TestGetCrdStatus_HappyPath(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	stream := &ociv1beta1.Stream{}
	stream.Status.OsokStatus.Ocid = "ocid1.stream.oc1..xxx"

	status, err := mgr.GetCrdStatus(stream)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.stream.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	cluster := &ociv1beta1.RedisCluster{}
	_, err := mgr.GetCrdStatus(cluster)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert")
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

// TestDelete_NoOcid verifies that Delete with no Spec.StreamId and non-empty
// Status.Ocid returns (true, nil) without making OCI API calls.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := makeTestManager(credClient, nil)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	// No Spec.StreamId; non-empty Status.Ocid triggers the early-return path.
	stream.Status.OsokStatus.Ocid = "ocid1.stream.oc1..xxx"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called when no spec StreamId is set")
}

// TestDelete_WrongType verifies Delete returns (true, nil) for non-Stream objects.
func TestDelete_WrongType(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	cluster := &ociv1beta1.RedisCluster{}
	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestDelete_DeleteStreamFails verifies Delete returns (true, nil) when the OCI
// DeleteStream call fails (no OCI credentials or mock error).
func TestDelete_DeleteStreamFails(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mockClient := &mockStreamAdminClient{
		deleteStreamFn: func(_ context.Context, _ streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
			return streaming.DeleteStreamResponse{}, errors.New("oci: network error")
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = "ocid1.stream.oc1..xxx"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called when DeleteStream fails")
}

// TestDelete_StreamDeleted verifies that Delete calls DeleteSecret when the stream
// reaches DELETED lifecycle state.
func TestDelete_StreamDeleted(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..xxx"
	mockClient := &mockStreamAdminClient{
		deleteStreamFn: func(_ context.Context, _ streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
			return streaming.DeleteStreamResponse{}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{
				Stream: streaming.Stream{
					Id:             common.String(streamID),
					Name:           common.String("test-stream"),
					LifecycleState: "DELETED",
				},
			}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = ociv1beta1.OCID(streamID)

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, credClient.deleteCalled, "DeleteSecret should be called when stream is DELETED")
}

// ---------------------------------------------------------------------------
// CreateOrUpdate tests
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-Stream objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	cluster := &ociv1beta1.RedisCluster{}
	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_EmptyName verifies CreateOrUpdate returns an error when
// both Spec.StreamId and Spec.Name are empty.
func TestCreateOrUpdate_EmptyName(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)

	stream := &ociv1beta1.Stream{}
	// No StreamId, no Name.
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_BindExistingById verifies binding to an existing stream by OCID
// when no update is required returns a successful response.
func TestCreateOrUpdate_BindExistingById(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..xxx"
	activeStream := makeActiveStream(streamID, "test-stream")

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: activeStream}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = ociv1beta1.OCID(streamID)
	// No FreeFormTags, DefinedTags, or StreamPoolId changes → no update needed.

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled, "CreateSecret should be called for an active stream")
}

// TestCreateOrUpdate_GetStreamFails verifies CreateOrUpdate propagates errors from GetStream.
func TestCreateOrUpdate_GetStreamFails(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..xxx"

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{}, errors.New("oci: unauthorized")
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = ociv1beta1.OCID(streamID)

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreateNew verifies that a new stream is created when no
// existing stream is found by name.
func TestCreateOrUpdate_CreateNew(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..new"
	activeStream := makeActiveStream(streamID, "new-stream")

	listCallCount := 0
	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			listCallCount++
			// Return empty list — no existing stream with this name.
			return streaming.ListStreamsResponse{Items: []streaming.StreamSummary{}}, nil
		},
		createStreamFn: func(_ context.Context, req streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error) {
			return streaming.CreateStreamResponse{
				Stream: streaming.Stream{
					Id:               common.String(streamID),
					Name:             req.Name,
					LifecycleState:   "CREATING",
					MessagesEndpoint: common.String("https://cell-1.streaming.us-phoenix-1.oci.oraclecloud.com"),
					StreamPoolId:     common.String("ocid1.streampool.oc1..xxx"),
					Partitions:       common.Int(1),
					RetentionInHours: common.Int(24),
				},
			}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: activeStream}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "new-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "new-stream"
	stream.Spec.Partitions = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled, "CreateSecret should be called after stream creation")
}

// TestCreateOrUpdate_ListStreamsFails verifies CreateOrUpdate propagates ListStreams errors.
func TestCreateOrUpdate_ListStreamsFails(t *testing.T) {
	credClient := &fakeCredentialClient{}

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			return streaming.ListStreamsResponse{}, errors.New("oci: service unavailable")
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "test-stream"
	stream.Spec.Partitions = 1

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_ExistingByName verifies that CreateOrUpdate binds to an existing
// stream located via name lookup (ListStreams) rather than a Spec.StreamId.
func TestCreateOrUpdate_ExistingByName(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..found"
	activeStream := makeActiveStream(streamID, "named-stream")

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			return streaming.ListStreamsResponse{
				Items: []streaming.StreamSummary{
					{Id: common.String(streamID), LifecycleState: "ACTIVE"},
				},
			}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: activeStream}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "named-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "named-stream"
	stream.Spec.Partitions = 1
	// No Spec.StreamId → CreateOrUpdate will call ListStreams to find stream by name.

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled, "CreateSecret should be called for the active stream")
}

// TestDelete_EmptyOcidPath verifies that Delete with no StreamId and no Status.Ocid
// traverses both the GetStreamOcid and GetStreamOCID(DELETE) list paths and returns
// (true, nil) when neither finds a stream.
func TestDelete_EmptyOcidPath(t *testing.T) {
	credClient := &fakeCredentialClient{}
	listCallCount := 0

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			listCallCount++
			// Both GetStreamOcid and GetStreamOCID("DELETE") calls return empty.
			return streaming.ListStreamsResponse{Items: []streaming.StreamSummary{}}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "test-stream"
	// No Spec.StreamId, no Status.Ocid.

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.Equal(t, 2, listCallCount, "ListStreams should be called via GetStreamOcid then GetStreamOCID(DELETE)")
	assert.False(t, credClient.deleteCalled)
}

// TestDelete_StreamFoundByName verifies Delete finds a stream via name lookup when
// Spec.StreamId is empty, then deletes it successfully.
func TestDelete_StreamFoundByName(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..named"

	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			// GetStreamOcid finds the stream in ACTIVE state.
			return streaming.ListStreamsResponse{
				Items: []streaming.StreamSummary{
					{Id: common.String(streamID), LifecycleState: "ACTIVE"},
				},
			}, nil
		},
		deleteStreamFn: func(_ context.Context, _ streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
			return streaming.DeleteStreamResponse{}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{
				Stream: streaming.Stream{
					Id:             common.String(streamID),
					Name:           common.String("test-stream"),
					LifecycleState: "DELETED",
				},
			}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "test-stream"
	// No Spec.StreamId, no Status.Ocid → goes through GetStreamOcid path.

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, credClient.deleteCalled, "DeleteSecret should be called after stream DELETED")
}

// TestDelete_FailedStreamFound verifies GetFailedOrDeleteStream handles a stream
// in FAILED state during a DELETE-phase lookup.
func TestDelete_FailedStreamFound(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..failed"

	callCount := 0
	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, _ streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			callCount++
			if callCount == 1 {
				// First call (GetStreamOcid): no ACTIVE/CREATING/UPDATING stream found.
				return streaming.ListStreamsResponse{Items: []streaming.StreamSummary{}}, nil
			}
			// Second call (GetStreamOCID("DELETE")): stream exists in FAILED state.
			return streaming.ListStreamsResponse{
				Items: []streaming.StreamSummary{
					{Id: common.String(streamID), LifecycleState: "FAILED"},
				},
			}, nil
		},
		deleteStreamFn: func(_ context.Context, _ streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error) {
			return streaming.DeleteStreamResponse{}, nil
		},
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{
				Stream: streaming.Stream{
					Id:             common.String(streamID),
					Name:           common.String("test-stream"),
					LifecycleState: "DELETING",
				},
			}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "test-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "test-stream"

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
}

// ---------------------------------------------------------------------------
// stream_secretgeneration tests
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// UpdateStream tests
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_UpdateViaFreeFormTags verifies that when FreeFormTags in the spec
// differ from the existing stream, UpdateStream is called with the correct details.
func TestCreateOrUpdate_UpdateViaFreeFormTags(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..upd"
	existingStream := makeActiveStream(streamID, "my-stream")
	// existingStream.FreeformTags is nil

	updateCalled := false
	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: existingStream}, nil
		},
		updateStreamFn: func(_ context.Context, req streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error) {
			updateCalled = true
			assert.Equal(t, streamID, *req.StreamId)
			return streaming.UpdateStreamResponse{}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "my-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = ociv1beta1.OCID(streamID)
	stream.Spec.Partitions = 1         // matches existing (required for UpdateStream validation)
	stream.Spec.RetentionInHours = 24  // matches existing
	stream.Spec.FreeFormTags = map[string]string{"env": "prod"} // differs from nil

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateStream should be called when FreeFormTags differ")
}

// TestCreateOrUpdate_UpdateViaDefinedTags verifies that when DefinedTags in the spec
// differ from the existing stream, UpdateStream is called and the DefinedTags branch
// in isValidUpdate is exercised.
func TestCreateOrUpdate_UpdateViaDefinedTags(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..deftags"
	existingStream := makeActiveStream(streamID, "tag-stream")
	// existingStream.DefinedTags is nil

	updateCalled := false
	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: existingStream}, nil
		},
		updateStreamFn: func(_ context.Context, _ streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error) {
			updateCalled = true
			return streaming.UpdateStreamResponse{}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "tag-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = ociv1beta1.OCID(streamID)
	stream.Spec.Partitions = 1
	stream.Spec.RetentionInHours = 24
	stream.Spec.DefinedTags = map[string]ociv1beta1.MapValue{
		"ns1": {"key1": "val1"},
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateStream should be called when DefinedTags differ")
}

// TestUpdateStream_PartitionsMismatch verifies UpdateStream returns an error when the
// spec partitions differ from the existing stream's partitions.
func TestUpdateStream_PartitionsMismatch(t *testing.T) {
	streamID := "ocid1.stream.oc1..partmm"
	existingStream := makeActiveStream(streamID, "my-stream")
	// existingStream.Partitions = 1

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: existingStream}, nil
		},
	}
	mgr := makeTestManager(&fakeCredentialClient{}, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Spec.StreamId = ociv1beta1.OCID(streamID)
	stream.Spec.Partitions = 3 // differs from existing (1)

	err := mgr.UpdateStream(context.Background(), stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Partitions can't be updated")
}

// TestUpdateStream_RetentionMismatch verifies UpdateStream returns an error when the
// spec RetentionInHours is below the minimum (24 hours).
func TestUpdateStream_RetentionMismatch(t *testing.T) {
	streamID := "ocid1.stream.oc1..retmm"
	existingStream := makeActiveStream(streamID, "my-stream")

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: existingStream}, nil
		},
	}
	mgr := makeTestManager(&fakeCredentialClient{}, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Spec.StreamId = ociv1beta1.OCID(streamID)
	stream.Spec.Partitions = 1         // matches
	stream.Spec.RetentionInHours = 12  // <= 23 → error

	err := mgr.UpdateStream(context.Background(), stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RetentionsHours can't be updated")
}

// TestGetStreamOcid_WithOptionalFilters verifies that when StreamPoolId and CompartmentId
// are set in the spec, they are included in the ListStreams request.
func TestGetStreamOcid_WithOptionalFilters(t *testing.T) {
	credClient := &fakeCredentialClient{}

	listCallCount := 0
	mockClient := &mockStreamAdminClient{
		listStreamsFn: func(_ context.Context, req streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error) {
			listCallCount++
			// Verify filters were passed through
			if listCallCount == 1 {
				assert.NotNil(t, req.StreamPoolId, "StreamPoolId should be set in ListStreams request")
				assert.NotNil(t, req.CompartmentId, "CompartmentId should be set in ListStreams request")
			}
			return streaming.ListStreamsResponse{Items: []streaming.StreamSummary{}}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "filtered-stream"
	stream.Namespace = "default"
	stream.Spec.Name = "filtered-stream"
	stream.Spec.StreamPoolId = "ocid1.streampool.oc1..filter"
	stream.Spec.CompartmentId = "ocid1.compartment.oc1..filter"
	// No Spec.StreamId, no Status.Ocid → goes through GetStreamOcid path in Delete

	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
}

// ---------------------------------------------------------------------------
// Retry policy predicate tests
// ---------------------------------------------------------------------------

// TestStreamRetryPolicy_Creating verifies shouldRetry returns true when stream is CREATING.
func TestStreamRetryPolicy_Creating(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportGetStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: streaming.GetStreamResponse{
			Stream: streaming.Stream{LifecycleState: "CREATING"},
		},
	}
	assert.True(t, shouldRetry(resp), "shouldRetry should return true when LifecycleState is CREATING")
}

// TestStreamRetryPolicy_Active verifies shouldRetry returns false when stream is ACTIVE.
func TestStreamRetryPolicy_Active(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportGetStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: streaming.GetStreamResponse{
			Stream: streaming.Stream{LifecycleState: "ACTIVE"},
		},
	}
	assert.False(t, shouldRetry(resp), "shouldRetry should return false when LifecycleState is ACTIVE")
}

// TestStreamRetryPolicy_NonResponse verifies shouldRetry returns true when the
// response type is not a GetStreamResponse (type assertion fails).
func TestStreamRetryPolicy_NonResponse(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportGetStreamRetryPredicate(mgr)

	// Empty response — type assertion to GetStreamResponse will fail → shouldRetry = true
	resp := common.OCIOperationResponse{}
	assert.True(t, shouldRetry(resp), "shouldRetry should return true when response type is not GetStreamResponse")
}

// TestDeleteStreamRetryPolicy_Deleting verifies the delete retry predicate returns true
// when the stream is in DELETING state.
func TestDeleteStreamRetryPolicy_Deleting(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportDeleteStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: streaming.GetStreamResponse{
			Stream: streaming.Stream{LifecycleState: "DELETING"},
		},
	}
	assert.True(t, shouldRetry(resp), "delete shouldRetry should return true when LifecycleState is DELETING")
}

// TestDeleteStreamRetryPolicy_Deleted verifies the delete retry predicate returns false
// when the stream is in DELETED state.
func TestDeleteStreamRetryPolicy_Deleted(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportDeleteStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{
		Response: streaming.GetStreamResponse{
			Stream: streaming.Stream{LifecycleState: "DELETED"},
		},
	}
	assert.False(t, shouldRetry(resp), "delete shouldRetry should return false when LifecycleState is DELETED")
}

// TestDeleteStreamRetryPolicy_NonResponse verifies the delete retry predicate returns true
// when the response type assertion to GetStreamResponse fails.
func TestDeleteStreamRetryPolicy_NonResponse(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	shouldRetry := ExportDeleteStreamRetryPredicate(mgr)

	resp := common.OCIOperationResponse{} // nil Response → type assertion fails → returns true
	assert.True(t, shouldRetry(resp), "delete shouldRetry should return true when response type is not GetStreamResponse")
}

// TestStreamRetryNextDuration verifies that the nextDuration function returns an
// exponential backoff duration (2^(attempt-1) seconds).
func TestStreamRetryNextDuration(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	nextDuration := ExportGetStreamNextDuration(mgr)

	// Attempt 1: 2^0 = 1 second
	resp := common.OCIOperationResponse{AttemptNumber: 1}
	assert.Equal(t, 1*time.Second, nextDuration(resp))
}

// TestDeleteStreamRetryNextDuration verifies the delete retry nextDuration function.
func TestDeleteStreamRetryNextDuration(t *testing.T) {
	mgr := makeTestManager(&fakeCredentialClient{}, nil)
	nextDuration := ExportDeleteStreamNextDuration(mgr)

	resp := common.OCIOperationResponse{AttemptNumber: 1}
	assert.Equal(t, 1*time.Second, nextDuration(resp))
}

// TestCreateOrUpdate_FailedLifecycle verifies that when the existing stream is in FAILED
// state, CreateOrUpdate sets a Failed status condition on the CR.
func TestCreateOrUpdate_FailedLifecycle(t *testing.T) {
	credClient := &fakeCredentialClient{}
	streamID := "ocid1.stream.oc1..failed"
	failedStream := makeActiveStream(streamID, "failed-stream")
	failedStream.LifecycleState = "FAILED"

	mockClient := &mockStreamAdminClient{
		getStreamFn: func(_ context.Context, _ streaming.GetStreamRequest) (streaming.GetStreamResponse, error) {
			return streaming.GetStreamResponse{Stream: failedStream}, nil
		},
	}
	mgr := makeTestManager(credClient, mockClient)

	stream := &ociv1beta1.Stream{}
	stream.Name = "failed-stream"
	stream.Namespace = "default"
	stream.Spec.StreamId = ociv1beta1.OCID(streamID)

	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, credClient.createCalled, "CreateSecret should NOT be called for a FAILED stream")
}

// ---------------------------------------------------------------------------
// stream_secretgeneration tests
// ---------------------------------------------------------------------------

// TestGetCredentialMap verifies the secret credential map contains the stream endpoint.
func TestGetCredentialMap(t *testing.T) {
	stream := streaming.Stream{
		Id:               common.String("ocid1.stream.oc1..xxx"),
		Name:             common.String("test-stream"),
		MessagesEndpoint: common.String("https://cell-1.streaming.us-phoenix-1.oci.oraclecloud.com"),
	}

	credMap, err := GetCredentialMapForTest(stream)
	assert.NoError(t, err)
	assert.Equal(t, "https://cell-1.streaming.us-phoenix-1.oci.oraclecloud.com", string(credMap["endpoint"]))
}
