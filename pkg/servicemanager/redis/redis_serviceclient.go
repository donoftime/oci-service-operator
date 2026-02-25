/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociRedis "github.com/oracle/oci-go-sdk/v65/redis"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

func getRedisClusterClient(provider common.ConfigurationProvider) (ociRedis.RedisClusterClient, error) {
	return ociRedis.NewRedisClusterClientWithConfigurationProvider(provider)
}

func (c *RedisClusterServiceManager) CreateRedisCluster(ctx context.Context, cluster ociv1beta1.RedisCluster) (ociRedis.CreateRedisClusterResponse, error) {
	client, err := getRedisClusterClient(c.Provider)
	if err != nil {
		return ociRedis.CreateRedisClusterResponse{}, err
	}
	c.Log.DebugLog("Creating RedisCluster", "displayName", cluster.Spec.DisplayName)

	softwareVersion, _ := ociRedis.GetMappingRedisClusterSoftwareVersionEnum(cluster.Spec.SoftwareVersion)

	createDetails := ociRedis.CreateRedisClusterDetails{
		DisplayName:     common.String(cluster.Spec.DisplayName),
		CompartmentId:   common.String(string(cluster.Spec.CompartmentId)),
		NodeCount:       common.Int(cluster.Spec.NodeCount),
		NodeMemoryInGBs: common.Float32(cluster.Spec.NodeMemoryInGBs),
		SoftwareVersion: softwareVersion,
		SubnetId:        common.String(string(cluster.Spec.SubnetId)),
	}

	if cluster.Spec.FreeFormTags != nil {
		createDetails.FreeformTags = cluster.Spec.FreeFormTags
	}

	if cluster.Spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&cluster.Spec.DefinedTags)
	}

	req := ociRedis.CreateRedisClusterRequest{
		CreateRedisClusterDetails: createDetails,
	}

	return client.CreateRedisCluster(ctx, req)
}

func (c *RedisClusterServiceManager) GetRedisCluster(ctx context.Context, clusterId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*ociRedis.RedisCluster, error) {
	client, err := getRedisClusterClient(c.Provider)
	if err != nil {
		return nil, err
	}

	req := ociRedis.GetRedisClusterRequest{
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

func (c *RedisClusterServiceManager) UpdateRedisCluster(ctx context.Context, cluster *ociv1beta1.RedisCluster) error {
	client, err := getRedisClusterClient(c.Provider)
	if err != nil {
		return err
	}

	updateDetails := ociRedis.UpdateRedisClusterDetails{}
	updateNeeded := false

	if cluster.Spec.DisplayName != "" {
		updateDetails.DisplayName = common.String(cluster.Spec.DisplayName)
		updateNeeded = true
	}

	if cluster.Spec.NodeCount > 0 {
		updateDetails.NodeCount = common.Int(cluster.Spec.NodeCount)
		updateNeeded = true
	}

	if cluster.Spec.NodeMemoryInGBs > 0 {
		updateDetails.NodeMemoryInGBs = common.Float32(cluster.Spec.NodeMemoryInGBs)
		updateNeeded = true
	}

	if cluster.Spec.FreeFormTags != nil {
		updateDetails.FreeformTags = cluster.Spec.FreeFormTags
		updateNeeded = true
	}

	if cluster.Spec.DefinedTags != nil {
		updateDetails.DefinedTags = *util.ConvertToOciDefinedTags(&cluster.Spec.DefinedTags)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	clusterId := string(cluster.Spec.RedisClusterId)
	if clusterId == "" {
		clusterId = string(cluster.Status.OsokStatus.Ocid)
	}

	req := ociRedis.UpdateRedisClusterRequest{
		RedisClusterId:            common.String(clusterId),
		UpdateRedisClusterDetails: updateDetails,
	}

	_, err = client.UpdateRedisCluster(ctx, req)
	return err
}

func (c *RedisClusterServiceManager) DeleteRedisCluster(ctx context.Context, cluster ociv1beta1.RedisCluster) (ociRedis.DeleteRedisClusterResponse, error) {
	client, err := getRedisClusterClient(c.Provider)
	if err != nil {
		return ociRedis.DeleteRedisClusterResponse{}, err
	}

	clusterId := string(cluster.Spec.RedisClusterId)
	if clusterId == "" {
		clusterId = string(cluster.Status.OsokStatus.Ocid)
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting RedisCluster %s", cluster.Spec.DisplayName))

	req := ociRedis.DeleteRedisClusterRequest{
		RedisClusterId: common.String(clusterId),
	}

	return client.DeleteRedisCluster(ctx, req)
}

func (c *RedisClusterServiceManager) GetRedisClusterOcid(ctx context.Context, cluster ociv1beta1.RedisCluster) (*ociv1beta1.OCID, error) {
	client, err := getRedisClusterClient(c.Provider)
	if err != nil {
		return nil, err
	}

	req := ociRedis.ListRedisClustersRequest{
		CompartmentId: common.String(string(cluster.Spec.CompartmentId)),
		DisplayName:   common.String(cluster.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListRedisClusters(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Items {
		state := item.LifecycleState
		if state == ociRedis.RedisClusterLifecycleStateActive ||
			state == ociRedis.RedisClusterLifecycleStateCreating ||
			state == ociRedis.RedisClusterLifecycleStateUpdating {
			ocid := ociv1beta1.OCID(*item.Id)
			return &ocid, nil
		}
	}

	return nil, nil
}
