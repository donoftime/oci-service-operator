/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/opensearch"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct{}

func (f *fakeCredentialClient) CreateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}
func (f *fakeCredentialClient) DeleteSecret(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
func (f *fakeCredentialClient) GetSecret(_ context.Context, _, _ string) (map[string][]byte, error) {
	return nil, nil
}
func (f *fakeCredentialClient) UpdateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}

func makeManager() *OpenSearchClusterServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	return NewOpenSearchClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		&fakeCredentialClient{}, nil, log, nil)
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from an OpenSearchCluster object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := makeManager()

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..xxx"

	status, err := mgr.GetCrdStatus(cluster)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.opensearchcluster.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := makeManager()

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-OpenSearchCluster objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := makeManager()

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	mgr := makeManager()

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestDelete_WithStatusOcid verifies deletion is attempted when status OCID is set.
func TestDelete_WithStatusOcid(t *testing.T) {
	mgr := makeManager()

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..xxx"

	// OCI API call will fail with invalid config, but we exercise the delete path.
	done, _ := mgr.Delete(context.Background(), cluster)
	assert.True(t, done)
}

// TestDelete_WrongType verifies Delete handles wrong object type gracefully.
func TestDelete_WrongType(t *testing.T) {
	mgr := makeManager()

	stream := &ociv1beta1.Stream{}
	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
}
