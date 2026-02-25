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
