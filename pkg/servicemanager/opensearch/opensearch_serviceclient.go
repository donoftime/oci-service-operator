/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch

import (
	"context"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

func getOpenSearchClusterClient(provider common.ConfigurationProvider) (opensearch.OpensearchClusterClient, error) {
	return opensearch.NewOpensearchClusterClientWithConfigurationProvider(provider)
}

func (c *OpenSearchClusterServiceManager) CreateOpenSearchCluster(ctx context.Context, cluster ociv1beta1.OpenSearchCluster) (opensearch.CreateOpensearchClusterResponse, error) {
	client, err := getOpenSearchClusterClient(c.Provider)
	if err != nil {
		return opensearch.CreateOpensearchClusterResponse{}, err
	}
	c.Log.DebugLog("Creating OpenSearch cluster", "displayName", cluster.Spec.DisplayName)

	details := opensearch.CreateOpensearchClusterDetails{
		DisplayName:                    common.String(cluster.Spec.DisplayName),
		CompartmentId:                  common.String(string(cluster.Spec.CompartmentId)),
		SoftwareVersion:                common.String(cluster.Spec.SoftwareVersion),
		MasterNodeCount:                common.Int(cluster.Spec.MasterNodeCount),
		MasterNodeHostType:             opensearch.MasterNodeHostTypeEnum(cluster.Spec.MasterNodeHostType),
		MasterNodeHostOcpuCount:        common.Int(cluster.Spec.MasterNodeHostOcpuCount),
		MasterNodeHostMemoryGB:         common.Int(cluster.Spec.MasterNodeHostMemoryGB),
		DataNodeCount:                  common.Int(cluster.Spec.DataNodeCount),
		DataNodeHostType:               opensearch.DataNodeHostTypeEnum(cluster.Spec.DataNodeHostType),
		DataNodeHostOcpuCount:          common.Int(cluster.Spec.DataNodeHostOcpuCount),
		DataNodeHostMemoryGB:           common.Int(cluster.Spec.DataNodeHostMemoryGB),
		DataNodeStorageGB:              common.Int(cluster.Spec.DataNodeStorageGB),
		OpendashboardNodeCount:         common.Int(cluster.Spec.OpendashboardNodeCount),
		OpendashboardNodeHostOcpuCount: common.Int(cluster.Spec.OpendashboardNodeHostOcpuCount),
		OpendashboardNodeHostMemoryGB:  common.Int(cluster.Spec.OpendashboardNodeHostMemoryGB),
		VcnId:                          common.String(string(cluster.Spec.VcnId)),
		SubnetId:                       common.String(string(cluster.Spec.SubnetId)),
		VcnCompartmentId:               common.String(string(cluster.Spec.VcnCompartmentId)),
		SubnetCompartmentId:            common.String(string(cluster.Spec.SubnetCompartmentId)),
	}

	if strings.TrimSpace(cluster.Spec.MasterNodeHostBareMetalShape) != "" {
		details.MasterNodeHostBareMetalShape = common.String(cluster.Spec.MasterNodeHostBareMetalShape)
	}
	if strings.TrimSpace(cluster.Spec.DataNodeHostBareMetalShape) != "" {
		details.DataNodeHostBareMetalShape = common.String(cluster.Spec.DataNodeHostBareMetalShape)
	}
	if strings.TrimSpace(cluster.Spec.SecurityMode) != "" {
		details.SecurityMode = opensearch.SecurityModeEnum(cluster.Spec.SecurityMode)
	}
	if strings.TrimSpace(cluster.Spec.SecurityMasterUserName) != "" {
		details.SecurityMasterUserName = common.String(cluster.Spec.SecurityMasterUserName)
	}
	if strings.TrimSpace(cluster.Spec.SecurityMasterUserPasswordHash) != "" {
		details.SecurityMasterUserPasswordHash = common.String(cluster.Spec.SecurityMasterUserPasswordHash)
	}
	if cluster.Spec.FreeFormTags != nil {
		details.FreeformTags = cluster.Spec.FreeFormTags
	}
	if cluster.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&cluster.Spec.DefinedTags)
	}

	return client.CreateOpensearchCluster(ctx, opensearch.CreateOpensearchClusterRequest{
		CreateOpensearchClusterDetails: details,
	})
}

func (c *OpenSearchClusterServiceManager) GetOpenSearchCluster(ctx context.Context, clusterId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*opensearch.OpensearchCluster, error) {
	client, err := getOpenSearchClusterClient(c.Provider)
	if err != nil {
		return nil, err
	}

	req := opensearch.GetOpensearchClusterRequest{
		OpensearchClusterId: common.String(string(clusterId)),
	}
	if retryPolicy != nil {
		req.RequestMetadata.RetryPolicy = retryPolicy
	}

	resp, err := client.GetOpensearchCluster(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.OpensearchCluster, nil
}

func (c *OpenSearchClusterServiceManager) UpdateOpenSearchCluster(ctx context.Context, cluster *ociv1beta1.OpenSearchCluster) error {
	client, err := getOpenSearchClusterClient(c.Provider)
	if err != nil {
		return err
	}

	details := opensearch.UpdateOpensearchClusterDetails{
		DisplayName: common.String(cluster.Spec.DisplayName),
	}
	if strings.TrimSpace(cluster.Spec.SoftwareVersion) != "" {
		details.SoftwareVersion = common.String(cluster.Spec.SoftwareVersion)
	}
	if cluster.Spec.FreeFormTags != nil {
		details.FreeformTags = cluster.Spec.FreeFormTags
	}
	if cluster.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&cluster.Spec.DefinedTags)
	}

	_, err = client.UpdateOpensearchCluster(ctx, opensearch.UpdateOpensearchClusterRequest{
		OpensearchClusterId:            common.String(string(cluster.Status.OsokStatus.Ocid)),
		UpdateOpensearchClusterDetails: details,
	})
	return err
}

func (c *OpenSearchClusterServiceManager) DeleteOpenSearchCluster(ctx context.Context, clusterId ociv1beta1.OCID) error {
	client, err := getOpenSearchClusterClient(c.Provider)
	if err != nil {
		return err
	}

	_, err = client.DeleteOpensearchCluster(ctx, opensearch.DeleteOpensearchClusterRequest{
		OpensearchClusterId: common.String(string(clusterId)),
	})
	return err
}

func (c *OpenSearchClusterServiceManager) GetOpenSearchClusterOCID(ctx context.Context, cluster ociv1beta1.OpenSearchCluster) (*ociv1beta1.OCID, error) {
	client, err := getOpenSearchClusterClient(c.Provider)
	if err != nil {
		return nil, err
	}

	req := opensearch.ListOpensearchClustersRequest{
		CompartmentId: common.String(string(cluster.Spec.CompartmentId)),
		DisplayName:   common.String(cluster.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListOpensearchClusters(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Items {
		state := item.LifecycleState
		if state == opensearch.OpensearchClusterLifecycleStateActive ||
			state == opensearch.OpensearchClusterLifecycleStateCreating ||
			state == opensearch.OpensearchClusterLifecycleStateUpdating {
			ocid := ociv1beta1.OCID(*item.Id)
			return &ocid, nil
		}
	}
	return nil, nil
}
