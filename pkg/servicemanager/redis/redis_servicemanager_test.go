/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociredis "github.com/oracle/oci-go-sdk/v65/redis"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/redis"
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

func makeActiveRedisCluster(id, displayName string) ociredis.RedisCluster {
	return ociredis.RedisCluster{
		Id:                        common.String(id),
		DisplayName:               common.String(displayName),
		LifecycleState:            ociredis.RedisClusterLifecycleStateActive,
		PrimaryFqdn:               common.String("primary.redis.example.com"),
		PrimaryEndpointIpAddress:  common.String("10.0.0.1"),
		ReplicasFqdn:              common.String("replicas.redis.example.com"),
		ReplicasEndpointIpAddress: common.String("10.0.0.2"),
		NodeCount:                 common.Int(3),
		NodeMemoryInGBs:           common.Float32(16.0),
		SoftwareVersion:           ociredis.RedisClusterSoftwareVersionV705,
		SubnetId:                  common.String("ocid1.subnet.oc1..xxx"),
		CompartmentId:             common.String("ocid1.compartment.oc1..xxx"),
		NodeCollection:            &ociredis.NodeCollection{},
	}
}

// TestGetCredentialMap verifies the secret credential map is built correctly from a RedisCluster.
func TestGetCredentialMap(t *testing.T) {
	cluster := makeActiveRedisCluster("ocid1.redis.xxx", "test-cluster")
	credMap := GetCredentialMapForTest(cluster)

	assert.Equal(t, "primary.redis.example.com", string(credMap["primaryFqdn"]))
	assert.Equal(t, "10.0.0.1", string(credMap["primaryEndpointIpAddress"]))
	assert.Equal(t, "replicas.redis.example.com", string(credMap["replicasFqdn"]))
	assert.Equal(t, "10.0.0.2", string(credMap["replicasEndpointIpAddress"]))
}

// TestGetCredentialMap_NilFields verifies nil pointer fields are handled gracefully.
func TestGetCredentialMap_NilFields(t *testing.T) {
	cluster := ociredis.RedisCluster{
		Id:             common.String("ocid1.redis.xxx"),
		DisplayName:    common.String("empty-cluster"),
		NodeCollection: &ociredis.NodeCollection{},
	}
	credMap := GetCredentialMapForTest(cluster)
	// nil fields should not appear in the map
	assert.NotContains(t, credMap, "primaryFqdn")
	assert.NotContains(t, credMap, "replicasFqdn")
}

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"

	done, err := mgr.Delete(context.Background(), cluster)
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

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Status.OsokStatus.Ocid = "ocid1.redis.oc1..xxx"

	// The OCI API call will fail with invalid config, but we exercise the path.
	// In a full integration test the OCI client would be mocked.
	_, _ = mgr.Delete(context.Background(), cluster)
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from a RedisCluster object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.redis.xxx"

	status, err := mgr.GetCrdStatus(cluster)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.redis.xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-RedisCluster objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// mockOciRedisClient implements RedisClusterClientInterface for unit testing.
type mockOciRedisClient struct {
	createFn func(ctx context.Context, req ociredis.CreateRedisClusterRequest) (ociredis.CreateRedisClusterResponse, error)
	getFn    func(ctx context.Context, req ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error)
	listFn   func(ctx context.Context, req ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error)
	updateFn func(ctx context.Context, req ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error)
	deleteFn func(ctx context.Context, req ociredis.DeleteRedisClusterRequest) (ociredis.DeleteRedisClusterResponse, error)
}

func (m *mockOciRedisClient) CreateRedisCluster(ctx context.Context, req ociredis.CreateRedisClusterRequest) (ociredis.CreateRedisClusterResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return ociredis.CreateRedisClusterResponse{}, nil
}

func (m *mockOciRedisClient) GetRedisCluster(ctx context.Context, req ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return ociredis.GetRedisClusterResponse{}, nil
}

func (m *mockOciRedisClient) ListRedisClusters(ctx context.Context, req ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return ociredis.ListRedisClustersResponse{}, nil
}

func (m *mockOciRedisClient) UpdateRedisCluster(ctx context.Context, req ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return ociredis.UpdateRedisClusterResponse{}, nil
}

func (m *mockOciRedisClient) DeleteRedisCluster(ctx context.Context, req ociredis.DeleteRedisClusterRequest) (ociredis.DeleteRedisClusterResponse, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, req)
	}
	return ociredis.DeleteRedisClusterResponse{}, nil
}

// newRedisMgr creates a RedisClusterServiceManager with injected mock clients.
func newRedisMgr(t *testing.T, ociClient *mockOciRedisClient, credClient *fakeCredentialClient) *RedisClusterServiceManager {
	t.Helper()
	if credClient == nil {
		credClient = &fakeCredentialClient{}
	}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewRedisClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)
	if ociClient != nil {
		ExportSetClientForTest(mgr, ociClient)
	}
	return mgr
}

// TestGetRedisClusterOcid_ActiveReturnsOcid verifies that an ACTIVE cluster is found by display name.
func TestGetRedisClusterOcid_ActiveReturnsOcid(t *testing.T) {
	const clusterOcid = "ocid1.redis.oc1..active"
	ociClient := &mockOciRedisClient{
		listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
			return ociredis.ListRedisClustersResponse{
				RedisClusterCollection: ociredis.RedisClusterCollection{
					Items: []ociredis.RedisClusterSummary{
						{Id: common.String(clusterOcid), LifecycleState: ociredis.RedisClusterLifecycleStateActive},
					},
				},
			}, nil
		},
	}
	mgr := newRedisMgr(t, ociClient, nil)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	cluster.Spec.DisplayName = "my-cluster"

	ocid, err := mgr.GetRedisClusterOcid(context.Background(), *cluster)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID(clusterOcid), *ocid)
}

// TestGetRedisClusterOcid_CreatingReturnsOcid verifies that a CREATING cluster OCID is returned.
func TestGetRedisClusterOcid_CreatingReturnsOcid(t *testing.T) {
	const clusterOcid = "ocid1.redis.oc1..creating"
	ociClient := &mockOciRedisClient{
		listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
			return ociredis.ListRedisClustersResponse{
				RedisClusterCollection: ociredis.RedisClusterCollection{
					Items: []ociredis.RedisClusterSummary{
						{Id: common.String(clusterOcid), LifecycleState: ociredis.RedisClusterLifecycleStateCreating},
					},
				},
			}, nil
		},
	}
	mgr := newRedisMgr(t, ociClient, nil)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	cluster.Spec.DisplayName = "my-cluster"

	ocid, err := mgr.GetRedisClusterOcid(context.Background(), *cluster)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID(clusterOcid), *ocid)
}

// TestGetRedisClusterOcid_NotFound verifies that an empty list returns nil (no cluster found).
func TestGetRedisClusterOcid_NotFound(t *testing.T) {
	ociClient := &mockOciRedisClient{
		listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
			return ociredis.ListRedisClustersResponse{}, nil
		},
	}
	mgr := newRedisMgr(t, ociClient, nil)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	cluster.Spec.DisplayName = "nonexistent"

	ocid, err := mgr.GetRedisClusterOcid(context.Background(), *cluster)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}

// TestCreateOrUpdate_CreateNew_ListEmpty verifies CreateRedisCluster is called when no cluster exists.
func TestCreateOrUpdate_CreateNew_ListEmpty(t *testing.T) {
	const newOcid = "ocid1.redis.oc1..new"
	createCalled := false
	activeCluster := makeActiveRedisCluster(newOcid, "test-cluster")
	credClient := &fakeCredentialClient{}

	ociClient := &mockOciRedisClient{
		listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
			return ociredis.ListRedisClustersResponse{}, nil
		},
		createFn: func(_ context.Context, req ociredis.CreateRedisClusterRequest) (ociredis.CreateRedisClusterResponse, error) {
			createCalled = true
			assert.Equal(t, "test-cluster", *req.CreateRedisClusterDetails.DisplayName)
			return ociredis.CreateRedisClusterResponse{RedisCluster: activeCluster}, nil
		},
		getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{RedisCluster: activeCluster}, nil
		},
	}
	mgr := newRedisMgr(t, ociClient, credClient)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Spec.DisplayName = "test-cluster"
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	cluster.Spec.NodeCount = 3
	cluster.Spec.NodeMemoryInGBs = 16
	cluster.Spec.SoftwareVersion = "V7_0_5"
	cluster.Spec.SubnetId = "ocid1.subnet.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, createCalled, "CreateRedisCluster should have been called")
	assert.True(t, credClient.createCalled, "CreateSecret should have been called on new cluster")
	assert.Equal(t, ociv1beta1.OCID(newOcid), cluster.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_CreateNew_ExistingByDisplayName verifies Create is NOT called when cluster found by name.
func TestCreateOrUpdate_CreateNew_ExistingByDisplayName(t *testing.T) {
	const existingOcid = "ocid1.redis.oc1..existing"
	createCalled := false
	activeCluster := makeActiveRedisCluster(existingOcid, "test-cluster")

	ociClient := &mockOciRedisClient{
		listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
			return ociredis.ListRedisClustersResponse{
				RedisClusterCollection: ociredis.RedisClusterCollection{
					Items: []ociredis.RedisClusterSummary{
						{Id: common.String(existingOcid), LifecycleState: ociredis.RedisClusterLifecycleStateActive},
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ ociredis.CreateRedisClusterRequest) (ociredis.CreateRedisClusterResponse, error) {
			createCalled = true
			return ociredis.CreateRedisClusterResponse{}, nil
		},
		getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{RedisCluster: activeCluster}, nil
		},
	}
	mgr := newRedisMgr(t, ociClient, nil)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Spec.DisplayName = "test-cluster"
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, createCalled, "CreateRedisCluster should NOT be called when cluster exists by display name")
}

// TestCreateOrUpdate_UpdatePath_FieldDiffTriggersUpdate verifies UpdateRedisCluster is called when spec differs.
func TestCreateOrUpdate_UpdatePath_FieldDiffTriggersUpdate(t *testing.T) {
	const clusterOcid = "ocid1.redis.oc1..update"
	updateCalled := false
	existingCluster := makeActiveRedisCluster(clusterOcid, "old-name")

	ociClient := &mockOciRedisClient{
		getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{RedisCluster: existingCluster}, nil
		},
		updateFn: func(_ context.Context, req ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error) {
			updateCalled = true
			assert.Equal(t, "new-name", *req.UpdateRedisClusterDetails.DisplayName)
			return ociredis.UpdateRedisClusterResponse{}, nil
		},
	}
	mgr := newRedisMgr(t, ociClient, nil)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Spec.RedisClusterId = clusterOcid
	cluster.Spec.DisplayName = "new-name"
	cluster.Status.OsokStatus.Ocid = clusterOcid

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateRedisCluster should have been called when display name differs")
}

// TestCreateOrUpdate_UpdatePath_NoOpWhenUnchanged verifies no OCI update call when spec matches existing.
func TestCreateOrUpdate_UpdatePath_NoOpWhenUnchanged(t *testing.T) {
	const clusterOcid = "ocid1.redis.oc1..noop"
	updateCalled := false
	existingCluster := makeActiveRedisCluster(clusterOcid, "same-name")

	ociClient := &mockOciRedisClient{
		getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{RedisCluster: existingCluster}, nil
		},
		updateFn: func(_ context.Context, _ ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error) {
			updateCalled = true
			return ociredis.UpdateRedisClusterResponse{}, nil
		},
	}
	mgr := newRedisMgr(t, ociClient, nil)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Spec.RedisClusterId = clusterOcid
	cluster.Spec.DisplayName = "same-name" // matches existing
	cluster.Spec.NodeCount = 3             // matches existing
	cluster.Spec.NodeMemoryInGBs = 16.0    // matches existing
	cluster.Status.OsokStatus.Ocid = clusterOcid

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, updateCalled, "UpdateRedisCluster should NOT be called when spec matches existing state")
}

// TestDelete_WithOcid_CallsOCIDelete verifies that DeleteRedisCluster OCI call is made when OCID is set.
func TestDelete_WithOcid_CallsOCIDelete(t *testing.T) {
	const clusterOcid = "ocid1.redis.oc1..todelete"
	deleteCalled := false

	ociClient := &mockOciRedisClient{
		deleteFn: func(_ context.Context, req ociredis.DeleteRedisClusterRequest) (ociredis.DeleteRedisClusterResponse, error) {
			deleteCalled = true
			assert.Equal(t, clusterOcid, *req.RedisClusterId)
			return ociredis.DeleteRedisClusterResponse{}, nil
		},
	}
	mgr := newRedisMgr(t, ociClient, nil)

	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Status.OsokStatus.Ocid = clusterOcid

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled, "DeleteRedisCluster should have been called with the cluster OCID")
}
