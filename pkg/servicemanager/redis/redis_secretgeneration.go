/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import (
	"context"
	"fmt"

	ociRedis "github.com/oracle/oci-go-sdk/v65/redis"
)

func (c *RedisClusterServiceManager) addToSecret(ctx context.Context, namespace string, clusterName string,
	cluster ociRedis.RedisCluster) (bool, error) {

	c.Log.InfoLog("Creating the Credential Map for RedisCluster")
	credMap, err := getRedisCredentialMap(cluster)
	if err != nil {
		c.Log.ErrorLog(err, "Error while creating RedisCluster secret map")
		return false, err
	}

	c.Log.InfoLog(fmt.Sprintf("Creating RedisCluster connection secret - namespace: %s clusterName: %s", namespace, clusterName))
	return c.CredentialClient.CreateSecret(ctx, clusterName, namespace, nil, credMap)
}

func getRedisCredentialMap(cluster ociRedis.RedisCluster) (map[string][]byte, error) {
	credMap := make(map[string][]byte)
	if cluster.PrimaryFqdn != nil {
		credMap["primaryFqdn"] = []byte(*cluster.PrimaryFqdn)
	}
	if cluster.PrimaryEndpointIpAddress != nil {
		credMap["primaryEndpointIpAddress"] = []byte(*cluster.PrimaryEndpointIpAddress)
	}
	if cluster.ReplicasFqdn != nil {
		credMap["replicasFqdn"] = []byte(*cluster.ReplicasFqdn)
	}
	if cluster.ReplicasEndpointIpAddress != nil {
		credMap["replicasEndpointIpAddress"] = []byte(*cluster.ReplicasEndpointIpAddress)
	}
	return credMap, nil
}

func (c *RedisClusterServiceManager) deleteFromSecret(ctx context.Context, namespace string, clusterName string) (bool, error) {
	c.Log.InfoLog(fmt.Sprintf("Deleting RedisCluster connection secret - namespace: %s clusterName: %s", namespace, clusterName))
	return c.CredentialClient.DeleteSecret(ctx, clusterName, namespace)
}
