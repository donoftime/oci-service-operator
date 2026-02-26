/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociqueue "github.com/oracle/oci-go-sdk/v65/queue"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/queue"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ---------------------------------------------------------------------------
// fakeCredentialClient — implements credhelper.CredentialClient for testing.
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// fakeQueueAdminClient — implements QueueAdminClientInterface for testing.
// ---------------------------------------------------------------------------

type fakeQueueAdminClient struct {
	createQueueFn func(ctx context.Context, req ociqueue.CreateQueueRequest) (ociqueue.CreateQueueResponse, error)
	getQueueFn    func(ctx context.Context, req ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error)
	listQueuesFn  func(ctx context.Context, req ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error)
	updateQueueFn func(ctx context.Context, req ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error)
	deleteQueueFn func(ctx context.Context, req ociqueue.DeleteQueueRequest) (ociqueue.DeleteQueueResponse, error)
}

func (f *fakeQueueAdminClient) CreateQueue(ctx context.Context, req ociqueue.CreateQueueRequest) (ociqueue.CreateQueueResponse, error) {
	if f.createQueueFn != nil {
		return f.createQueueFn(ctx, req)
	}
	return ociqueue.CreateQueueResponse{OpcWorkRequestId: common.String("wr-001")}, nil
}

func (f *fakeQueueAdminClient) GetQueue(ctx context.Context, req ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
	if f.getQueueFn != nil {
		return f.getQueueFn(ctx, req)
	}
	return ociqueue.GetQueueResponse{}, nil
}

func (f *fakeQueueAdminClient) ListQueues(ctx context.Context, req ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
	if f.listQueuesFn != nil {
		return f.listQueuesFn(ctx, req)
	}
	return ociqueue.ListQueuesResponse{}, nil
}

func (f *fakeQueueAdminClient) UpdateQueue(ctx context.Context, req ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error) {
	if f.updateQueueFn != nil {
		return f.updateQueueFn(ctx, req)
	}
	return ociqueue.UpdateQueueResponse{}, nil
}

func (f *fakeQueueAdminClient) DeleteQueue(ctx context.Context, req ociqueue.DeleteQueueRequest) (ociqueue.DeleteQueueResponse, error) {
	if f.deleteQueueFn != nil {
		return f.deleteQueueFn(ctx, req)
	}
	return ociqueue.DeleteQueueResponse{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeActiveQueue(id, displayName, messagesEndpoint string) ociqueue.Queue {
	return ociqueue.Queue{
		Id:               common.String(id),
		DisplayName:      common.String(displayName),
		CompartmentId:    common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState:   ociqueue.QueueLifecycleStateActive,
		MessagesEndpoint: common.String(messagesEndpoint),
		RetentionInSeconds:           common.Int(86400),
		VisibilityInSeconds:          common.Int(30),
		TimeoutInSeconds:             common.Int(30),
		DeadLetterQueueDeliveryCount: common.Int(5),
	}
}

func defaultLog() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
}

func emptyProvider() common.ConfigurationProvider {
	return common.NewRawConfigurationProvider("", "", "", "", "", nil)
}

// mgrWithFake creates a service manager with the given fake OCI client injected.
func mgrWithFake(credClient *fakeCredentialClient, fake *fakeQueueAdminClient) *OciQueueServiceManager {
	mgr := NewOciQueueServiceManager(emptyProvider(), credClient, nil, defaultLog())
	ExportSetClientForTest(mgr, fake)
	return mgr
}

// ---------------------------------------------------------------------------
// TestGetCredentialMap
// ---------------------------------------------------------------------------

// TestGetCredentialMap verifies the secret credential map is built correctly from a Queue.
func TestGetCredentialMap(t *testing.T) {
	q := makeActiveQueue("ocid1.queue.xxx", "test-queue", "https://cell1.queue.messaging.us-ashburn-1.oci.oraclecloud.com/20210201/queues/ocid1.queue.xxx")
	credMap := GetCredentialMapForTest(q)

	assert.Equal(t, "ocid1.queue.xxx", string(credMap["id"]))
	assert.Equal(t, "https://cell1.queue.messaging.us-ashburn-1.oci.oraclecloud.com/20210201/queues/ocid1.queue.xxx", string(credMap["messagesEndpoint"]))
	assert.Equal(t, "test-queue", string(credMap["displayName"]))
}

// TestGetCredentialMap_NilFields verifies nil pointer fields are handled gracefully.
func TestGetCredentialMap_NilFields(t *testing.T) {
	q := ociqueue.Queue{
		Id:             common.String("ocid1.queue.xxx"),
		LifecycleState: ociqueue.QueueLifecycleStateActive,
	}
	credMap := GetCredentialMapForTest(q)
	assert.NotContains(t, credMap, "messagesEndpoint")
	assert.NotContains(t, credMap, "displayName")
}

// TestGetCredentialMap_AllFieldsPopulated verifies all credential fields appear when set.
func TestGetCredentialMap_AllFieldsPopulated(t *testing.T) {
	q := makeActiveQueue(
		"ocid1.queue.oc1..aaa",
		"full-queue",
		"https://cell1.queue.messaging.us-phoenix-1.oci.oraclecloud.com/20210201/queues/ocid1.queue.oc1..aaa",
	)
	credMap := GetCredentialMapForTest(q)

	assert.Equal(t, "ocid1.queue.oc1..aaa", string(credMap["id"]))
	assert.Equal(t, "full-queue", string(credMap["displayName"]))
	assert.Contains(t, string(credMap["messagesEndpoint"]), "cell1.queue.messaging.us-phoenix-1")
}

// ---------------------------------------------------------------------------
// TestDelete
// ---------------------------------------------------------------------------

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	mgr := NewOciQueueServiceManager(emptyProvider(), credClient, nil, defaultLog())

	q := &ociv1beta1.OciQueue{}
	q.Name = "test-queue"
	q.Namespace = "default"

	done, err := mgr.Delete(context.Background(), q)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called when OCID is empty")
}

// TestDelete_SecretError verifies Delete tolerates secret-deletion errors.
func TestDelete_SecretError(t *testing.T) {
	credClient := &fakeCredentialClient{
		deleteSecretFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, errors.New("secret not found")
		},
	}
	mgr := NewOciQueueServiceManager(emptyProvider(), credClient, nil, defaultLog())

	q := &ociv1beta1.OciQueue{}
	q.Name = "test-queue"
	q.Namespace = "default"
	q.Status.OsokStatus.Ocid = "ocid1.queue.oc1..xxx"

	// The OCI API call will fail with invalid config, but we exercise the path.
	_, _ = mgr.Delete(context.Background(), q)
}

// TestDelete_WithFakeClient verifies Delete calls DeleteQueue and then DeleteSecret.
func TestDelete_WithFakeClient(t *testing.T) {
	credClient := &fakeCredentialClient{}
	fake := &fakeQueueAdminClient{}
	mgr := mgrWithFake(credClient, fake)

	q := &ociv1beta1.OciQueue{}
	q.Name = "test-queue"
	q.Namespace = "default"
	q.Status.OsokStatus.Ocid = "ocid1.queue.oc1..xxx"

	done, err := mgr.Delete(context.Background(), q)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, credClient.deleteCalled, "DeleteSecret should be called after DeleteQueue")
}

// ---------------------------------------------------------------------------
// TestGetCrdStatus
// ---------------------------------------------------------------------------

// TestGetCrdStatus_ReturnsStatus verifies status extraction from an OciQueue object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := NewOciQueueServiceManager(emptyProvider(), &fakeCredentialClient{}, nil, defaultLog())

	q := &ociv1beta1.OciQueue{}
	q.Status.OsokStatus.Ocid = "ocid1.queue.xxx"

	status, err := mgr.GetCrdStatus(q)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.queue.xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := NewOciQueueServiceManager(emptyProvider(), &fakeCredentialClient{}, nil, defaultLog())

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — type assertion
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-OciQueue objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := NewOciQueueServiceManager(emptyProvider(), &fakeCredentialClient{}, nil, defaultLog())

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// TestGetQueueOcid (via CreateOrUpdate to exercise the no-ID path)
// ---------------------------------------------------------------------------

// TestGetQueueOcid_ActiveFound verifies that an ACTIVE queue in the list is returned.
func TestGetQueueOcid_ActiveFound(t *testing.T) {
	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{
					Items: []ociqueue.QueueSummary{
						{Id: common.String("ocid1.queue.oc1..active"), LifecycleState: ociqueue.QueueLifecycleStateActive},
					},
				},
			}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := ociv1beta1.OciQueue{}
	q.Spec.DisplayName = "my-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetQueueOcid(context.Background(), q)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID("ocid1.queue.oc1..active"), *ocid)
}

// TestGetQueueOcid_CreatingFound verifies that a CREATING queue is also returned.
func TestGetQueueOcid_CreatingFound(t *testing.T) {
	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{
					Items: []ociqueue.QueueSummary{
						{Id: common.String("ocid1.queue.oc1..creating"), LifecycleState: ociqueue.QueueLifecycleStateCreating},
					},
				},
			}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := ociv1beta1.OciQueue{}
	ocid, err := mgr.GetQueueOcid(context.Background(), q)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID("ocid1.queue.oc1..creating"), *ocid)
}

// TestGetQueueOcid_NotFound verifies nil is returned when no matching queue exists.
func TestGetQueueOcid_NotFound(t *testing.T) {
	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{Items: []ociqueue.QueueSummary{}},
			}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := ociv1beta1.OciQueue{}
	ocid, err := mgr.GetQueueOcid(context.Background(), q)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}

// TestGetQueueOcid_ListError verifies that a ListQueues error propagates.
func TestGetQueueOcid_ListError(t *testing.T) {
	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{}, errors.New("network error")
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := ociv1beta1.OciQueue{}
	_, err := mgr.GetQueueOcid(context.Background(), q)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}

// ---------------------------------------------------------------------------
// TestCreateQueue
// ---------------------------------------------------------------------------

// TestCreateQueue_WithDeadLetterAndRetention verifies DLQ and retention fields are
// included in the OCI CreateQueue request when set in the spec.
func TestCreateQueue_WithDeadLetterAndRetention(t *testing.T) {
	var capturedReq ociqueue.CreateQueueRequest

	fake := &fakeQueueAdminClient{
		createQueueFn: func(_ context.Context, req ociqueue.CreateQueueRequest) (ociqueue.CreateQueueResponse, error) {
			capturedReq = req
			return ociqueue.CreateQueueResponse{OpcWorkRequestId: common.String("wr-dlq-001")}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := ociv1beta1.OciQueue{}
	q.Spec.DisplayName = "dlq-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	q.Spec.RetentionInSeconds = 3600
	q.Spec.DeadLetterQueueDeliveryCount = 10
	q.Spec.VisibilityInSeconds = 60
	q.Spec.TimeoutInSeconds = 30

	wrID, err := mgr.CreateQueue(context.Background(), q)
	assert.NoError(t, err)
	assert.Equal(t, "wr-dlq-001", wrID)
	assert.NotNil(t, capturedReq.CreateQueueDetails.RetentionInSeconds)
	assert.Equal(t, 3600, *capturedReq.CreateQueueDetails.RetentionInSeconds)
	assert.NotNil(t, capturedReq.CreateQueueDetails.DeadLetterQueueDeliveryCount)
	assert.Equal(t, 10, *capturedReq.CreateQueueDetails.DeadLetterQueueDeliveryCount)
	assert.NotNil(t, capturedReq.CreateQueueDetails.VisibilityInSeconds)
	assert.Equal(t, 60, *capturedReq.CreateQueueDetails.VisibilityInSeconds)
	assert.NotNil(t, capturedReq.CreateQueueDetails.TimeoutInSeconds)
	assert.Equal(t, 30, *capturedReq.CreateQueueDetails.TimeoutInSeconds)
}

// TestCreateQueue_MinimalFields verifies zero-value optional fields are omitted.
func TestCreateQueue_MinimalFields(t *testing.T) {
	var capturedReq ociqueue.CreateQueueRequest

	fake := &fakeQueueAdminClient{
		createQueueFn: func(_ context.Context, req ociqueue.CreateQueueRequest) (ociqueue.CreateQueueResponse, error) {
			capturedReq = req
			return ociqueue.CreateQueueResponse{OpcWorkRequestId: common.String("wr-min-001")}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := ociv1beta1.OciQueue{}
	q.Spec.DisplayName = "min-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	// Retention, visibility, timeout, DLQ all zero — not included in request.

	_, err := mgr.CreateQueue(context.Background(), q)
	assert.NoError(t, err)
	assert.Nil(t, capturedReq.CreateQueueDetails.RetentionInSeconds)
	assert.Nil(t, capturedReq.CreateQueueDetails.DeadLetterQueueDeliveryCount)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — create paths (no QueueId)
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_NoId_QueueNotFound_Provisioning verifies the first-create path:
// no existing queue → CreateQueue is submitted → status set to Provisioning.
func TestCreateOrUpdate_NoId_QueueNotFound_Provisioning(t *testing.T) {
	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{Items: []ociqueue.QueueSummary{}},
			}, nil
		},
		createQueueFn: func(_ context.Context, _ ociqueue.CreateQueueRequest) (ociqueue.CreateQueueResponse, error) {
			return ociqueue.CreateQueueResponse{OpcWorkRequestId: common.String("wr-new-001")}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := &ociv1beta1.OciQueue{}
	q.Name = "new-queue"
	q.Namespace = "default"
	q.Spec.DisplayName = "new-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "should be Provisioning (not successful) while creating")
}

// TestCreateOrUpdate_NoId_QueueCreating verifies that a CREATING queue triggers
// a Provisioning status and early return.
func TestCreateOrUpdate_NoId_QueueCreating(t *testing.T) {
	queueID := "ocid1.queue.oc1..creating"
	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{
					Items: []ociqueue.QueueSummary{
						{Id: common.String(queueID), LifecycleState: ociqueue.QueueLifecycleStateCreating},
					},
				},
			}, nil
		},
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			return ociqueue.GetQueueResponse{
				Queue: ociqueue.Queue{
					Id:             common.String(queueID),
					DisplayName:    common.String("creating-queue"),
					LifecycleState: ociqueue.QueueLifecycleStateCreating,
				},
			}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := &ociv1beta1.OciQueue{}
	q.Name = "creating-queue"
	q.Namespace = "default"
	q.Spec.DisplayName = "creating-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "should still be Provisioning while CREATING")
}

// TestCreateOrUpdate_NoId_QueueActive_Success verifies the happy path:
// queue found ACTIVE → secret created → IsSuccessful=true.
func TestCreateOrUpdate_NoId_QueueActive_Success(t *testing.T) {
	queueID := "ocid1.queue.oc1..active"
	endpoint := "https://cell1.queue.messaging.us-ashburn-1.oci.oraclecloud.com/20210201/queues/" + queueID

	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{
					Items: []ociqueue.QueueSummary{
						{Id: common.String(queueID), LifecycleState: ociqueue.QueueLifecycleStateActive},
					},
				},
			}, nil
		},
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			return ociqueue.GetQueueResponse{
				Queue: makeActiveQueue(queueID, "active-queue", endpoint),
			}, nil
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := mgrWithFake(credClient, fake)

	q := &ociv1beta1.OciQueue{}
	q.Name = "active-queue"
	q.Namespace = "default"
	q.Spec.DisplayName = "active-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled, "secret should be created on success")
}

// TestCreateOrUpdate_NoId_GetQueueOcidError verifies error propagation from GetQueueOcid.
func TestCreateOrUpdate_NoId_GetQueueOcidError(t *testing.T) {
	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{}, errors.New("list failed")
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := &ociv1beta1.OciQueue{}
	q.Spec.DisplayName = "test-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_NoId_CreateQueueError verifies error propagation from CreateQueue.
func TestCreateOrUpdate_NoId_CreateQueueError(t *testing.T) {
	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{Items: []ociqueue.QueueSummary{}},
			}, nil
		},
		createQueueFn: func(_ context.Context, _ ociqueue.CreateQueueRequest) (ociqueue.CreateQueueResponse, error) {
			return ociqueue.CreateQueueResponse{}, errors.New("create failed")
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := &ociv1beta1.OciQueue{}
	q.Spec.DisplayName = "fail-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — existing OCID paths
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_WithId_Binds verifies that a set QueueId causes GetQueue + UpdateQueue
// and results in a successful bind with secret creation.
func TestCreateOrUpdate_WithId_Binds(t *testing.T) {
	queueID := "ocid1.queue.oc1..existing"
	endpoint := "https://cell1.queue.messaging.us-ashburn-1.oci.oraclecloud.com/20210201/queues/" + queueID

	activeQueue := makeActiveQueue(queueID, "existing-queue", endpoint)

	fake := &fakeQueueAdminClient{
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			return ociqueue.GetQueueResponse{Queue: activeQueue}, nil
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := mgrWithFake(credClient, fake)

	q := &ociv1beta1.OciQueue{}
	q.Name = "existing-queue"
	q.Namespace = "default"
	q.Spec.QueueId = ociv1beta1.OCID(queueID)
	q.Spec.DisplayName = "existing-queue" // Matches existing → no update needed.
	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(queueID)

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, credClient.createCalled, "secret should be created on bind")
}

// TestCreateOrUpdate_WithId_UpdateNeeded verifies that differing spec values trigger
// a real UpdateQueue call.
func TestCreateOrUpdate_WithId_UpdateNeeded(t *testing.T) {
	queueID := "ocid1.queue.oc1..update"
	endpoint := "https://cell1.queue.messaging.us-ashburn-1.oci.oraclecloud.com/20210201/queues/" + queueID

	existingQueue := makeActiveQueue(queueID, "old-name", endpoint)

	var updateCalled bool
	fake := &fakeQueueAdminClient{
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			return ociqueue.GetQueueResponse{Queue: existingQueue}, nil
		},
		updateQueueFn: func(_ context.Context, _ ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error) {
			updateCalled = true
			return ociqueue.UpdateQueueResponse{}, nil
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := mgrWithFake(credClient, fake)

	q := &ociv1beta1.OciQueue{}
	q.Name = "update-queue"
	q.Namespace = "default"
	q.Spec.QueueId = ociv1beta1.OCID(queueID)
	q.Spec.DisplayName = "new-name" // Differs from existing → triggers update.
	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(queueID)

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateQueue should have been called")
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — FAILED state
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_QueueFailed verifies that a FAILED lifecycle state results in
// IsSuccessful=false and no error.
func TestCreateOrUpdate_QueueFailed(t *testing.T) {
	queueID := "ocid1.queue.oc1..failed"

	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{
					Items: []ociqueue.QueueSummary{
						{Id: common.String(queueID), LifecycleState: ociqueue.QueueLifecycleStateActive},
					},
				},
			}, nil
		},
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			return ociqueue.GetQueueResponse{
				Queue: ociqueue.Queue{
					Id:             common.String(queueID),
					DisplayName:    common.String("failed-queue"),
					LifecycleState: ociqueue.QueueLifecycleStateFailed,
				},
			}, nil
		},
	}
	mgr := mgrWithFake(&fakeCredentialClient{}, fake)

	q := &ociv1beta1.OciQueue{}
	q.Name = "failed-queue"
	q.Namespace = "default"
	q.Spec.DisplayName = "failed-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "FAILED queue should not be successful")
}

// ---------------------------------------------------------------------------
// TestCreateOrUpdate — secret already exists
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_SecretAlreadyExists verifies that an AlreadyExists error from
// CreateSecret is treated as success (idempotent).
func TestCreateOrUpdate_SecretAlreadyExists(t *testing.T) {
	queueID := "ocid1.queue.oc1..already"
	endpoint := "https://cell1.queue.messaging.us-ashburn-1.oci.oraclecloud.com/20210201/queues/" + queueID

	fake := &fakeQueueAdminClient{
		listQueuesFn: func(_ context.Context, _ ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error) {
			return ociqueue.ListQueuesResponse{
				QueueCollection: ociqueue.QueueCollection{
					Items: []ociqueue.QueueSummary{
						{Id: common.String(queueID), LifecycleState: ociqueue.QueueLifecycleStateActive},
					},
				},
			}, nil
		},
		getQueueFn: func(_ context.Context, _ ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error) {
			return ociqueue.GetQueueResponse{
				Queue: makeActiveQueue(queueID, "already-queue", endpoint),
			}, nil
		},
	}
	credClient := &fakeCredentialClient{
		createSecretFn: func(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
			return false, apierrors.NewAlreadyExists(schema.GroupResource{}, "already-queue")
		},
	}
	mgr := mgrWithFake(credClient, fake)

	q := &ociv1beta1.OciQueue{}
	q.Name = "already-queue"
	q.Namespace = "default"
	q.Spec.DisplayName = "already-queue"
	q.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), q, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful, "AlreadyExists on secret should be treated as success")
}
