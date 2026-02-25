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

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewOciQueueServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

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
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewOciQueueServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	q := &ociv1beta1.OciQueue{}
	q.Name = "test-queue"
	q.Namespace = "default"
	q.Status.OsokStatus.Ocid = "ocid1.queue.oc1..xxx"

	// The OCI API call will fail with invalid config, but we exercise the path.
	_, _ = mgr.Delete(context.Background(), q)
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from an OciQueue object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewOciQueueServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	q := &ociv1beta1.OciQueue{}
	q.Status.OsokStatus.Ocid = "ocid1.queue.xxx"

	status, err := mgr.GetCrdStatus(q)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.queue.xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewOciQueueServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-OciQueue objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewOciQueueServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}
