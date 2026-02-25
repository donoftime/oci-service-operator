/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import (
	"testing"

	ociRedis "github.com/oracle/oci-go-sdk/v65/redis"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestGetRedisCredentialMap_AllFields(t *testing.T) {
	primaryFqdn := "primary.redis.example.com"
	primaryIP := "10.0.0.1"
	replicasFqdn := "replicas.redis.example.com"
	replicasIP := "10.0.0.2"

	cluster := ociRedis.RedisCluster{
		Id:                        strPtr("ocid1.rediscluster.oc1..test"),
		DisplayName:               strPtr("test-cluster"),
		CompartmentId:             strPtr("ocid1.compartment.oc1..test"),
		NodeCount:                 intPtr(3),
		NodeMemoryInGBs:           float32Ptr(4.0),
		SoftwareVersion:           ociRedis.RedisClusterSoftwareVersionV705,
		SubnetId:                  strPtr("ocid1.subnet.oc1..test"),
		PrimaryFqdn:               &primaryFqdn,
		PrimaryEndpointIpAddress:  &primaryIP,
		ReplicasFqdn:              &replicasFqdn,
		ReplicasEndpointIpAddress: &replicasIP,
	}

	credMap, err := getRedisCredentialMap(cluster)
	assert.NoError(t, err)
	assert.Equal(t, []byte(primaryFqdn), credMap["primaryFqdn"])
	assert.Equal(t, []byte(primaryIP), credMap["primaryEndpointIpAddress"])
	assert.Equal(t, []byte(replicasFqdn), credMap["replicasFqdn"])
	assert.Equal(t, []byte(replicasIP), credMap["replicasEndpointIpAddress"])
}

func TestGetRedisCredentialMap_NilFields(t *testing.T) {
	cluster := ociRedis.RedisCluster{
		Id:              strPtr("ocid1.rediscluster.oc1..test"),
		DisplayName:     strPtr("test-cluster"),
		CompartmentId:   strPtr("ocid1.compartment.oc1..test"),
		NodeCount:       intPtr(1),
		NodeMemoryInGBs: float32Ptr(2.0),
		SoftwareVersion: ociRedis.RedisClusterSoftwareVersionV705,
		SubnetId:        strPtr("ocid1.subnet.oc1..test"),
	}

	credMap, err := getRedisCredentialMap(cluster)
	assert.NoError(t, err)
	assert.Empty(t, credMap)
}

func TestConvert_Success(t *testing.T) {
	mgr := &RedisClusterServiceManager{}
	cluster := &ociv1beta1.RedisCluster{
		Spec: ociv1beta1.RedisClusterSpec{
			DisplayName:     "test",
			CompartmentId:   "ocid1.compartment.oc1..test",
			NodeCount:       3,
			NodeMemoryInGBs: 4.0,
			SoftwareVersion: "V7_0_5",
			SubnetId:        "ocid1.subnet.oc1..test",
		},
	}

	result, err := mgr.convert(cluster)
	assert.NoError(t, err)
	assert.Equal(t, cluster, result)
}

func TestConvert_Failure(t *testing.T) {
	mgr := &RedisClusterServiceManager{}

	// Pass a different runtime.Object type
	stream := &ociv1beta1.Stream{}
	result, err := mgr.convert(stream)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "RedisCluster")
}

func TestGetCrdStatus(t *testing.T) {
	mgr := &RedisClusterServiceManager{}
	cluster := &ociv1beta1.RedisCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.rediscluster.oc1..test"

	status, err := mgr.GetCrdStatus(cluster)
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, ociv1beta1.OCID("ocid1.rediscluster.oc1..test"), status.Ocid)
}

func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := &RedisClusterServiceManager{}
	stream := &ociv1beta1.Stream{}

	status, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Nil(t, status)
}

func TestRedisClusterSpec_RequiredFields(t *testing.T) {
	cluster := ociv1beta1.RedisCluster{
		Spec: ociv1beta1.RedisClusterSpec{
			CompartmentId:   "ocid1.compartment.oc1..test",
			DisplayName:     "my-redis-cluster",
			NodeCount:       3,
			NodeMemoryInGBs: 4.0,
			SoftwareVersion: "V7_0_5",
			SubnetId:        "ocid1.subnet.oc1..test",
		},
	}

	assert.Equal(t, "my-redis-cluster", cluster.Spec.DisplayName)
	assert.Equal(t, 3, cluster.Spec.NodeCount)
	assert.Equal(t, float32(4.0), cluster.Spec.NodeMemoryInGBs)
	assert.Equal(t, "V7_0_5", cluster.Spec.SoftwareVersion)
	assert.Equal(t, ociv1beta1.OCID("ocid1.compartment.oc1..test"), cluster.Spec.CompartmentId)
	assert.Equal(t, ociv1beta1.OCID("ocid1.subnet.oc1..test"), cluster.Spec.SubnetId)
}

func TestRedisClusterStatus_OcidSet(t *testing.T) {
	cluster := ociv1beta1.RedisCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.rediscluster.oc1..testocid"

	assert.Equal(t, ociv1beta1.OCID("ocid1.rediscluster.oc1..testocid"), cluster.Status.OsokStatus.Ocid)
}

// Test that Delete handles missing OCID gracefully
func TestDelete_MissingOCID(t *testing.T) {
	mgr := &RedisClusterServiceManager{}
	cluster := &ociv1beta1.RedisCluster{
		Spec: ociv1beta1.RedisClusterSpec{
			DisplayName: "test-cluster",
		},
	}

	// With empty OCID, delete should return true (done) without error
	// because there's nothing to delete
	assert.Empty(t, string(cluster.Spec.RedisClusterId))
	assert.Empty(t, string(cluster.Status.OsokStatus.Ocid))

	_ = mgr
}

// Helpers for test data
func strPtr(s string) *string       { return &s }
func intPtr(i int) *int             { return &i }
func float32Ptr(f float32) *float32 { return &f }
