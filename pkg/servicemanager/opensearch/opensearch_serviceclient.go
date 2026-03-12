/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// OpensearchClusterClientInterface defines the OCI operations used by OpenSearchClusterServiceManager.
// It is satisfied by opensearch.OpensearchClusterClient and enables injection of fakes in tests.
type OpensearchClusterClientInterface interface {
	CreateOpensearchCluster(ctx context.Context, request opensearch.CreateOpensearchClusterRequest) (opensearch.CreateOpensearchClusterResponse, error)
	GetOpensearchCluster(ctx context.Context, request opensearch.GetOpensearchClusterRequest) (opensearch.GetOpensearchClusterResponse, error)
	ListOpensearchClusters(ctx context.Context, request opensearch.ListOpensearchClustersRequest) (opensearch.ListOpensearchClustersResponse, error)
	ResizeOpensearchClusterHorizontal(ctx context.Context, request opensearch.ResizeOpensearchClusterHorizontalRequest) (opensearch.ResizeOpensearchClusterHorizontalResponse, error)
	ResizeOpensearchClusterVertical(ctx context.Context, request opensearch.ResizeOpensearchClusterVerticalRequest) (opensearch.ResizeOpensearchClusterVerticalResponse, error)
	UpdateOpensearchCluster(ctx context.Context, request opensearch.UpdateOpensearchClusterRequest) (opensearch.UpdateOpensearchClusterResponse, error)
	DeleteOpensearchCluster(ctx context.Context, request opensearch.DeleteOpensearchClusterRequest) (opensearch.DeleteOpensearchClusterResponse, error)
}

func getOpenSearchClusterClient(provider common.ConfigurationProvider) (OpensearchClusterClientInterface, error) {
	return opensearch.NewOpensearchClusterClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OpenSearchClusterServiceManager) getOCIClient() (OpensearchClusterClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getOpenSearchClusterClient(c.Provider)
}

func (c *OpenSearchClusterServiceManager) CreateOpenSearchCluster(ctx context.Context, cluster ociv1beta1.OpenSearchCluster) (opensearch.CreateOpensearchClusterResponse, error) {
	client, err := c.getOCIClient()
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
	client, err := c.getOCIClient()
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
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	targetID, err := resolveClusterID(cluster.Status.OsokStatus.Ocid, cluster.Spec.OpenSearchClusterId)
	if err != nil {
		return err
	}

	existing, err := c.GetOpenSearchCluster(ctx, targetID, nil)
	if err != nil {
		return err
	}

	if err := validateUnsupportedOpenSearchChanges(cluster, existing); err != nil {
		return err
	}

	if err = applyOpenSearchHorizontalResize(ctx, client, cluster, existing, targetID); err != nil {
		return err
	}

	displayName := cluster.Spec.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = safeString(existing.DisplayName)
	}

	if err = applyOpenSearchVerticalResize(ctx, client, cluster, existing, targetID); err != nil {
		return err
	}

	if err = applyOpenSearchSoftwareUpdate(ctx, client, cluster, existing, targetID, displayName); err != nil {
		return err
	}

	return applyOpenSearchGeneralUpdate(ctx, client, cluster, existing, targetID, displayName)
}

func applyOpenSearchHorizontalResize(ctx context.Context, client OpensearchClusterClientInterface,
	cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster, targetID ociv1beta1.OCID) error {
	horizontalDetails, updateNeeded := buildHorizontalResizeDetails(cluster, existing)
	if !updateNeeded {
		return nil
	}
	_, err := client.ResizeOpensearchClusterHorizontal(ctx, opensearch.ResizeOpensearchClusterHorizontalRequest{
		OpensearchClusterId:                      common.String(string(targetID)),
		ResizeOpensearchClusterHorizontalDetails: horizontalDetails,
	})
	return err
}

func applyOpenSearchVerticalResize(ctx context.Context, client OpensearchClusterClientInterface,
	cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster, targetID ociv1beta1.OCID) error {
	verticalDetails, updateNeeded := buildVerticalResizeDetails(cluster, existing)
	if !updateNeeded {
		return nil
	}
	_, err := client.ResizeOpensearchClusterVertical(ctx, opensearch.ResizeOpensearchClusterVerticalRequest{
		OpensearchClusterId:                    common.String(string(targetID)),
		ResizeOpensearchClusterVerticalDetails: verticalDetails,
	})
	return err
}

func applyOpenSearchSoftwareUpdate(ctx context.Context, client OpensearchClusterClientInterface,
	cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster, targetID ociv1beta1.OCID, displayName string) error {
	softwareDetails, updateNeeded := buildSoftwareOnlyUpdateDetails(cluster, existing, displayName)
	if !updateNeeded {
		return nil
	}
	_, err := client.UpdateOpensearchCluster(ctx, opensearch.UpdateOpensearchClusterRequest{
		OpensearchClusterId:            common.String(string(targetID)),
		UpdateOpensearchClusterDetails: softwareDetails,
	})
	return err
}

func applyOpenSearchGeneralUpdate(ctx context.Context, client OpensearchClusterClientInterface,
	cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster, targetID ociv1beta1.OCID, displayName string) error {
	generalDetails, updateNeeded := buildGeneralUpdateDetails(cluster, existing, displayName)
	if !updateNeeded {
		return nil
	}
	_, err := client.UpdateOpensearchCluster(ctx, opensearch.UpdateOpensearchClusterRequest{
		OpensearchClusterId:            common.String(string(targetID)),
		UpdateOpensearchClusterDetails: generalDetails,
	})
	return err
}

func buildHorizontalResizeDetails(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) (opensearch.ResizeOpensearchClusterHorizontalDetails, bool) {
	details := opensearch.ResizeOpensearchClusterHorizontalDetails{}
	updateNeeded := false

	if cluster.Spec.MasterNodeCount > 0 && (existing.MasterNodeCount == nil || *existing.MasterNodeCount != cluster.Spec.MasterNodeCount) {
		details.MasterNodeCount = common.Int(cluster.Spec.MasterNodeCount)
		updateNeeded = true
	}
	if cluster.Spec.DataNodeCount > 0 && (existing.DataNodeCount == nil || *existing.DataNodeCount != cluster.Spec.DataNodeCount) {
		details.DataNodeCount = common.Int(cluster.Spec.DataNodeCount)
		updateNeeded = true
	}
	if cluster.Spec.OpendashboardNodeCount > 0 && (existing.OpendashboardNodeCount == nil || *existing.OpendashboardNodeCount != cluster.Spec.OpendashboardNodeCount) {
		details.OpendashboardNodeCount = common.Int(cluster.Spec.OpendashboardNodeCount)
		updateNeeded = true
	}

	return details, updateNeeded
}

func buildVerticalResizeDetails(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) (opensearch.ResizeOpensearchClusterVerticalDetails, bool) {
	details := opensearch.ResizeOpensearchClusterVerticalDetails{}
	updateNeeded := setOpenSearchIntDetail(cluster.Spec.MasterNodeHostOcpuCount, existing.MasterNodeHostOcpuCount, func(value *int) {
		details.MasterNodeHostOcpuCount = value
	})
	if setOpenSearchIntDetail(cluster.Spec.MasterNodeHostMemoryGB, existing.MasterNodeHostMemoryGB, func(value *int) {
		details.MasterNodeHostMemoryGB = value
	}) {
		updateNeeded = true
	}
	if setOpenSearchIntDetail(cluster.Spec.DataNodeHostOcpuCount, existing.DataNodeHostOcpuCount, func(value *int) {
		details.DataNodeHostOcpuCount = value
	}) {
		updateNeeded = true
	}
	if setOpenSearchIntDetail(cluster.Spec.DataNodeHostMemoryGB, existing.DataNodeHostMemoryGB, func(value *int) {
		details.DataNodeHostMemoryGB = value
	}) {
		updateNeeded = true
	}
	if setOpenSearchIntDetail(cluster.Spec.DataNodeStorageGB, existing.DataNodeStorageGB, func(value *int) {
		details.DataNodeStorageGB = value
	}) {
		updateNeeded = true
	}
	if setOpenSearchIntDetail(cluster.Spec.OpendashboardNodeHostOcpuCount, existing.OpendashboardNodeHostOcpuCount, func(value *int) {
		details.OpendashboardNodeHostOcpuCount = value
	}) {
		updateNeeded = true
	}
	if setOpenSearchIntDetail(cluster.Spec.OpendashboardNodeHostMemoryGB, existing.OpendashboardNodeHostMemoryGB, func(value *int) {
		details.OpendashboardNodeHostMemoryGB = value
	}) {
		updateNeeded = true
	}

	return details, updateNeeded
}

func setOpenSearchIntDetail(desired int, existing *int, assign func(*int)) bool {
	if desired <= 0 || (existing != nil && *existing == desired) {
		return false
	}
	assign(common.Int(desired))
	return true
}

func buildSoftwareOnlyUpdateDetails(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster, displayName string) (opensearch.UpdateOpensearchClusterDetails, bool) {
	details := opensearch.UpdateOpensearchClusterDetails{DisplayName: common.String(displayName)}
	updateNeeded := false

	if strings.TrimSpace(cluster.Spec.SoftwareVersion) != "" && safeString(existing.SoftwareVersion) != cluster.Spec.SoftwareVersion {
		details.SoftwareVersion = common.String(cluster.Spec.SoftwareVersion)
		updateNeeded = true
	}

	return details, updateNeeded
}

func buildGeneralUpdateDetails(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster, displayName string) (opensearch.UpdateOpensearchClusterDetails, bool) {
	details := opensearch.UpdateOpensearchClusterDetails{DisplayName: common.String(displayName)}
	updateNeeded := strings.TrimSpace(cluster.Spec.DisplayName) != "" && safeString(existing.DisplayName) != cluster.Spec.DisplayName

	if applyOpenSearchSecurityModeUpdate(&details, cluster, existing) {
		updateNeeded = true
	}
	if applyOpenSearchSecurityUserUpdate(&details, cluster, existing) {
		updateNeeded = true
	}
	if applyOpenSearchSecurityPasswordUpdate(&details, cluster, existing) {
		updateNeeded = true
	}
	if applyOpenSearchFreeformTagUpdate(&details, cluster, existing) {
		updateNeeded = true
	}
	if applyOpenSearchDefinedTagUpdate(&details, cluster, existing) {
		updateNeeded = true
	}

	return details, updateNeeded
}

func applyOpenSearchSecurityModeUpdate(details *opensearch.UpdateOpensearchClusterDetails, cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) bool {
	if strings.TrimSpace(cluster.Spec.SecurityMode) == "" || string(existing.SecurityMode) == cluster.Spec.SecurityMode {
		return false
	}
	details.SecurityMode = opensearch.SecurityModeEnum(cluster.Spec.SecurityMode)
	return true
}

func applyOpenSearchSecurityUserUpdate(details *opensearch.UpdateOpensearchClusterDetails, cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) bool {
	if strings.TrimSpace(cluster.Spec.SecurityMasterUserName) == "" || safeString(existing.SecurityMasterUserName) == cluster.Spec.SecurityMasterUserName {
		return false
	}
	details.SecurityMasterUserName = common.String(cluster.Spec.SecurityMasterUserName)
	return true
}

func applyOpenSearchSecurityPasswordUpdate(details *opensearch.UpdateOpensearchClusterDetails, cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) bool {
	if strings.TrimSpace(cluster.Spec.SecurityMasterUserPasswordHash) == "" || safeString(existing.SecurityMasterUserPasswordHash) == cluster.Spec.SecurityMasterUserPasswordHash {
		return false
	}
	details.SecurityMasterUserPasswordHash = common.String(cluster.Spec.SecurityMasterUserPasswordHash)
	return true
}

func applyOpenSearchFreeformTagUpdate(details *opensearch.UpdateOpensearchClusterDetails, cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) bool {
	if cluster.Spec.FreeFormTags == nil || reflect.DeepEqual(existing.FreeformTags, cluster.Spec.FreeFormTags) {
		return false
	}
	details.FreeformTags = cluster.Spec.FreeFormTags
	return true
}

func applyOpenSearchDefinedTagUpdate(details *opensearch.UpdateOpensearchClusterDetails, cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) bool {
	if cluster.Spec.DefinedTags == nil {
		return false
	}
	desiredDefinedTags := *util.ConvertToOciDefinedTags(&cluster.Spec.DefinedTags)
	if reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
		return false
	}
	details.DefinedTags = desiredDefinedTags
	return true
}

func validateUnsupportedOpenSearchChanges(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if err := validateOpenSearchCompartment(cluster, existing); err != nil {
		return err
	}
	if err := validateOpenSearchDataNodeHostShape(cluster, existing); err != nil {
		return err
	}
	if err := validateOpenSearchDataNodeHostType(cluster, existing); err != nil {
		return err
	}
	if err := validateOpenSearchMasterNodeHostShape(cluster, existing); err != nil {
		return err
	}
	if err := validateOpenSearchMasterNodeHostType(cluster, existing); err != nil {
		return err
	}
	if err := validateOpenSearchSubnetCompartment(cluster, existing); err != nil {
		return err
	}
	if err := validateOpenSearchSubnet(cluster, existing); err != nil {
		return err
	}
	if err := validateOpenSearchVcnCompartment(cluster, existing); err != nil {
		return err
	}
	return validateOpenSearchVcn(cluster, existing)
}

func validateOpenSearchCompartment(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.CompartmentId != "" && safeString(existing.CompartmentId) != string(cluster.Spec.CompartmentId) {
		return fmt.Errorf("compartmentId cannot be updated in place")
	}
	return nil
}

func validateOpenSearchDataNodeHostShape(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.DataNodeHostBareMetalShape != "" && safeString(existing.DataNodeHostBareMetalShape) != cluster.Spec.DataNodeHostBareMetalShape {
		return fmt.Errorf("dataNodeHostBareMetalShape cannot be updated in place")
	}
	return nil
}

func validateOpenSearchDataNodeHostType(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.DataNodeHostType != "" && string(existing.DataNodeHostType) != cluster.Spec.DataNodeHostType {
		return fmt.Errorf("dataNodeHostType cannot be updated in place")
	}
	return nil
}

func validateOpenSearchMasterNodeHostShape(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.MasterNodeHostBareMetalShape != "" && safeString(existing.MasterNodeHostBareMetalShape) != cluster.Spec.MasterNodeHostBareMetalShape {
		return fmt.Errorf("masterNodeHostBareMetalShape cannot be updated in place")
	}
	return nil
}

func validateOpenSearchMasterNodeHostType(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.MasterNodeHostType != "" && string(existing.MasterNodeHostType) != cluster.Spec.MasterNodeHostType {
		return fmt.Errorf("masterNodeHostType cannot be updated in place")
	}
	return nil
}

func validateOpenSearchSubnetCompartment(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.SubnetCompartmentId != "" && safeString(existing.SubnetCompartmentId) != string(cluster.Spec.SubnetCompartmentId) {
		return fmt.Errorf("subnetCompartmentId cannot be updated in place")
	}
	return nil
}

func validateOpenSearchSubnet(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.SubnetId != "" && safeString(existing.SubnetId) != string(cluster.Spec.SubnetId) {
		return fmt.Errorf("subnetId cannot be updated in place")
	}
	return nil
}

func validateOpenSearchVcnCompartment(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.VcnCompartmentId != "" && safeString(existing.VcnCompartmentId) != string(cluster.Spec.VcnCompartmentId) {
		return fmt.Errorf("vcnCompartmentId cannot be updated in place")
	}
	return nil
}

func validateOpenSearchVcn(cluster *ociv1beta1.OpenSearchCluster, existing *opensearch.OpensearchCluster) error {
	if cluster.Spec.VcnId != "" && safeString(existing.VcnId) != string(cluster.Spec.VcnId) {
		return fmt.Errorf("vcnId cannot be updated in place")
	}
	return nil
}

func (c *OpenSearchClusterServiceManager) DeleteOpenSearchCluster(ctx context.Context, clusterId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	_, err = client.DeleteOpensearchCluster(ctx, opensearch.DeleteOpensearchClusterRequest{
		OpensearchClusterId: common.String(string(clusterId)),
	})
	return err
}

func (c *OpenSearchClusterServiceManager) GetOpenSearchClusterOCID(ctx context.Context, cluster ociv1beta1.OpenSearchCluster) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
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
