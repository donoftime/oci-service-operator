/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams_test

import (
	"context"
	"errors"
	"testing"

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
