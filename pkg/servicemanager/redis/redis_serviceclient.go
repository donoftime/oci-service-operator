/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/redis"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// RedisClusterClientInterface defines the OCI operations used by RedisClusterServiceManager.
type RedisClusterClientInterface interface {
	CreateRedisCluster(ctx context.Context, request redis.CreateRedisClusterRequest) (redis.CreateRedisClusterResponse, error)
	GetRedisCluster(ctx context.Context, request redis.GetRedisClusterRequest) (redis.GetRedisClusterResponse, error)
	ListRedisClusters(ctx context.Context, request redis.ListRedisClustersRequest) (redis.ListRedisClustersResponse, error)
	UpdateRedisCluster(ctx context.Context, request redis.UpdateRedisClusterRequest) (redis.UpdateRedisClusterResponse, error)
	DeleteRedisCluster(ctx context.Context, request redis.DeleteRedisClusterRequest) (redis.DeleteRedisClusterResponse, error)
}

func getRedisClusterClient(provider common.ConfigurationProvider) (redis.RedisClusterClient, error) {
	return redis.NewRedisClusterClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *RedisClusterServiceManager) getOCIClient() (RedisClusterClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getRedisClusterClient(c.Provider)
}

// CreateRedisCluster calls the OCI API to create a new Redis cluster.
func (c *RedisClusterServiceManager) CreateRedisCluster(ctx context.Context, cluster ociv1beta1.RedisCluster) (redis.CreateRedisClusterResponse, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return redis.CreateRedisClusterResponse{}, err
	}

	c.Log.DebugLog("Creating RedisCluster", "name", cluster.Spec.DisplayName)

	softwareVersion := redis.RedisClusterSoftwareVersionEnum(cluster.Spec.SoftwareVersion)

	details := redis.CreateRedisClusterDetails{
		DisplayName:     common.String(cluster.Spec.DisplayName),
		CompartmentId:   common.String(string(cluster.Spec.CompartmentId)),
		NodeCount:       common.Int(cluster.Spec.NodeCount),
		NodeMemoryInGBs: common.Float32(cluster.Spec.NodeMemoryInGBs),
		SoftwareVersion: softwareVersion,
		SubnetId:        common.String(string(cluster.Spec.SubnetId)),
		FreeformTags:    cluster.Spec.FreeFormTags,
	}

	if cluster.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&cluster.Spec.DefinedTags)
	}

	req := redis.CreateRedisClusterRequest{
		CreateRedisClusterDetails: details,
	}

	return client.CreateRedisCluster(ctx, req)
}

// GetRedisCluster retrieves a Redis cluster by OCID.
func (c *RedisClusterServiceManager) GetRedisCluster(ctx context.Context, clusterId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*redis.RedisCluster, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := redis.GetRedisClusterRequest{
		RedisClusterId: common.String(string(clusterId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetRedisCluster(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.RedisCluster, nil
}

// GetRedisClusterOcid looks up an existing Redis cluster by display name and returns its OCID if found.
func (c *RedisClusterServiceManager) GetRedisClusterOcid(ctx context.Context, cluster ociv1beta1.RedisCluster) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := redis.ListRedisClustersRequest{
		CompartmentId: common.String(string(cluster.Spec.CompartmentId)),
		DisplayName:   common.String(cluster.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListRedisClusters(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Redis clusters")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("RedisCluster %s exists with OCID %s", cluster.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("RedisCluster %s does not exist", cluster.Spec.DisplayName))
	return nil, nil
}

// UpdateRedisCluster updates an existing Redis cluster.
func (c *RedisClusterServiceManager) UpdateRedisCluster(ctx context.Context, cluster *ociv1beta1.RedisCluster) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	updateDetails := redis.UpdateRedisClusterDetails{}
	updateNeeded := false

	existing, err := c.GetRedisCluster(ctx, cluster.Status.OsokStatus.Ocid, nil)
	if err != nil {
		return err
	}

	if cluster.Spec.DisplayName != "" && *existing.DisplayName != cluster.Spec.DisplayName {
		updateDetails.DisplayName = common.String(cluster.Spec.DisplayName)
		updateNeeded = true
	}

	if cluster.Spec.NodeCount > 0 && *existing.NodeCount != cluster.Spec.NodeCount {
		updateDetails.NodeCount = common.Int(cluster.Spec.NodeCount)
		updateNeeded = true
	}

	if cluster.Spec.NodeMemoryInGBs > 0 && *existing.NodeMemoryInGBs != cluster.Spec.NodeMemoryInGBs {
		updateDetails.NodeMemoryInGBs = common.Float32(cluster.Spec.NodeMemoryInGBs)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := redis.UpdateRedisClusterRequest{
		RedisClusterId:            common.String(string(cluster.Status.OsokStatus.Ocid)),
		UpdateRedisClusterDetails: updateDetails,
	}

	_, err = client.UpdateRedisCluster(ctx, req)
	return err
}

// DeleteRedisCluster deletes the Redis cluster for the given OCID.
func (c *RedisClusterServiceManager) DeleteRedisCluster(ctx context.Context, clusterId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := redis.DeleteRedisClusterRequest{
		RedisClusterId: common.String(string(clusterId)),
	}

	_, err = client.DeleteRedisCluster(ctx, req)
	return err
}

// getRetryPolicy returns a retry policy that waits while a cluster is in CREATING state.
func (c *RedisClusterServiceManager) getRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(redis.GetRedisClusterResponse); ok {
			return resp.LifecycleState == redis.RedisClusterLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(1) * time.Minute
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

