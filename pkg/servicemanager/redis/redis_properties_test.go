/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociredis "github.com/oracle/oci-go-sdk/v65/redis"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func makePendingRedisCluster(id, displayName string, state ociredis.RedisClusterLifecycleStateEnum) ociredis.RedisCluster {
	cluster := makeActiveRedisCluster(id, displayName)
	cluster.LifecycleState = state
	return cluster
}

func makeRedisSpec(name string) *ociv1beta1.RedisCluster {
	cluster := &ociv1beta1.RedisCluster{}
	cluster.Name = name
	cluster.Namespace = "default"
	cluster.Spec.DisplayName = name
	cluster.Spec.CompartmentId = "ocid1.compartment.oc1..x"
	cluster.Spec.SubnetId = "ocid1.subnet.oc1..x"
	cluster.Spec.NodeCount = 3
	cluster.Spec.NodeMemoryInGBs = 16
	cluster.Spec.SoftwareVersion = "V7_0_5"
	return cluster
}

func TestPropertyRedisPendingStatesRequestRequeue(t *testing.T) {
	for _, state := range []ociredis.RedisClusterLifecycleStateEnum{
		ociredis.RedisClusterLifecycleStateCreating,
		ociredis.RedisClusterLifecycleStateUpdating,
	} {
		t.Run(string(state), func(t *testing.T) {
			ociCl := &fakeOciClient{
				listFn: func(_ context.Context, _ ociredis.ListRedisClustersRequest) (ociredis.ListRedisClustersResponse, error) {
					return ociredis.ListRedisClustersResponse{
						RedisClusterCollection: ociredis.RedisClusterCollection{
							Items: []ociredis.RedisClusterSummary{{Id: common.String("ocid1.redis.oc1..pending"), DisplayName: common.String("pending-redis"), LifecycleState: state}},
						},
					}, nil
				},
				getFn: func(_ context.Context, _ ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
					return ociredis.GetRedisClusterResponse{RedisCluster: makePendingRedisCluster("ocid1.redis.oc1..pending", "pending-redis", state)}, nil
				},
			}
			mgr := newMgrWithFakeClient(ociCl, &fakeCredentialClient{})
			cluster := makeRedisSpec("pending-redis")

			resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
			assert.NoError(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
		})
	}
}

func TestPropertyRedisBindByIDUsesSpecIDWhenStatusIsEmpty(t *testing.T) {
	var updatedID string
	ociCl := &fakeOciClient{
		getFn: func(_ context.Context, req ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{RedisCluster: makeActiveRedisCluster(*req.RedisClusterId, "old-bound-redis")}, nil
		},
		updateFn: func(_ context.Context, req ociredis.UpdateRedisClusterRequest) (ociredis.UpdateRedisClusterResponse, error) {
			updatedID = *req.RedisClusterId
			return ociredis.UpdateRedisClusterResponse{}, nil
		},
	}
	mgr := newMgrWithFakeClient(ociCl, &fakeCredentialClient{})
	cluster := makeRedisSpec("new-bound-redis")
	cluster.Spec.RedisClusterId = "ocid1.redis.oc1..bind"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, string(cluster.Spec.RedisClusterId), updatedID)
}

func TestPropertyRedisDeleteWaitsForConfirmedDisappearance(t *testing.T) {
	credCl := &fakeCredentialClient{}
	ociCl := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ociredis.DeleteRedisClusterRequest) (ociredis.DeleteRedisClusterResponse, error) {
			return ociredis.DeleteRedisClusterResponse{}, nil
		},
		getFn: func(_ context.Context, req ociredis.GetRedisClusterRequest) (ociredis.GetRedisClusterResponse, error) {
			return ociredis.GetRedisClusterResponse{RedisCluster: makeActiveRedisCluster(*req.RedisClusterId, "still-there")}, nil
		},
	}
	mgr := newMgrWithFakeClient(ociCl, credCl)
	cluster := makeRedisSpec("still-there")
	cluster.Status.OsokStatus.Ocid = "ocid1.redis.oc1..delete"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.False(t, done)
	assert.False(t, credCl.deleteCalled)
}
