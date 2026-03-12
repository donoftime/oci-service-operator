/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type OpenSearchClusterServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
	ociClient        OpensearchClusterClientInterface // injectable for testing; nil uses Provider
}

func NewOpenSearchClusterServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger, metricsClient *metrics.Metrics) *OpenSearchClusterServiceManager {
	return &OpenSearchClusterServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
		Metrics:          metricsClient,
	}
}

func (c *OpenSearchClusterServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	clusterObj, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	clusterInstance, response, done, err := c.prepareClusterForReconcile(ctx, clusterObj, kind, req)
	if err != nil || done {
		return response, err
	}

	return c.finishClusterReconcile(ctx, kind, req, clusterObj, clusterInstance), nil
}

func isValidUpdate(clusterObj ociv1beta1.OpenSearchCluster, clusterInstance opensearch.OpensearchCluster) bool {
	_, horizontalUpdateNeeded := buildHorizontalResizeDetails(&clusterObj, &clusterInstance)
	if horizontalUpdateNeeded {
		return true
	}
	_, verticalUpdateNeeded := buildVerticalResizeDetails(&clusterObj, &clusterInstance)
	if verticalUpdateNeeded {
		return true
	}
	displayName := clusterObj.Spec.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = safeString(clusterInstance.DisplayName)
	}
	_, softwareUpdateNeeded := buildSoftwareOnlyUpdateDetails(&clusterObj, &clusterInstance, displayName)
	if softwareUpdateNeeded {
		return true
	}
	_, generalUpdateNeeded := buildGeneralUpdateDetails(&clusterObj, &clusterInstance, displayName)
	return generalUpdateNeeded
}

func (c *OpenSearchClusterServiceManager) prepareClusterForReconcile(ctx context.Context, clusterObj *ociv1beta1.OpenSearchCluster,
	kind string, req ctrl.Request) (*opensearch.OpensearchCluster, servicemanager.OSOKResponse, bool, error) {
	if hasExplicitClusterID(clusterObj) {
		return c.fetchClusterForReconcile(ctx, clusterObj, clusterObj.Spec.OpenSearchClusterId, kind, req)
	}

	return c.prepareDiscoveredCluster(ctx, clusterObj, kind, req)
}

func (c *OpenSearchClusterServiceManager) prepareDiscoveredCluster(ctx context.Context, clusterObj *ociv1beta1.OpenSearchCluster,
	kind string, req ctrl.Request) (*opensearch.OpensearchCluster, servicemanager.OSOKResponse, bool, error) {
	clusterOcid, err := c.GetOpenSearchClusterOCID(ctx, *clusterObj)
	if err != nil {
		c.Log.ErrorLog(err, "Error while looking up OpenSearch cluster")
		c.recordFaultMetric(ctx, kind, req, "Failed to get OpenSearch cluster")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	if clusterOcid == nil {
		return c.createCluster(ctx, clusterObj, kind, req)
	}

	return c.fetchClusterForReconcile(ctx, clusterObj, *clusterOcid, kind, req)
}

func (c *OpenSearchClusterServiceManager) createCluster(ctx context.Context, clusterObj *ociv1beta1.OpenSearchCluster,
	kind string, req ctrl.Request) (*opensearch.OpensearchCluster, servicemanager.OSOKResponse, bool, error) {
	if _, err := c.CreateOpenSearchCluster(ctx, *clusterObj); err != nil {
		clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
		c.Log.ErrorLog(err, "Error creating OpenSearch cluster")
		c.recordFaultMetric(ctx, kind, req, "Error creating OpenSearch cluster")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	c.Log.InfoLog(fmt.Sprintf("OpenSearch cluster %s create initiated, provisioning", clusterObj.Spec.DisplayName))
	clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
		ociv1beta1.Provisioning, v1.ConditionTrue, "", "OpenSearch cluster is Provisioning", c.Log)

	return nil, servicemanager.OSOKResponse{
		IsSuccessful:    false,
		ShouldRequeue:   true,
		RequeueDuration: openSearchRequeueDuration,
	}, true, nil
}

func (c *OpenSearchClusterServiceManager) fetchClusterForReconcile(ctx context.Context, clusterObj *ociv1beta1.OpenSearchCluster,
	clusterID ociv1beta1.OCID, kind string, req ctrl.Request) (*opensearch.OpensearchCluster, servicemanager.OSOKResponse, bool, error) {
	clusterInstance, err := c.GetOpenSearchCluster(ctx, clusterID, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error fetching OpenSearch cluster")
		c.recordFaultMetric(ctx, kind, req, "Error fetching OpenSearch cluster")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	clusterObj.Status.OsokStatus.Ocid = clusterID
	if err := c.updateClusterIfNeeded(ctx, clusterObj, clusterInstance, kind, req); err != nil {
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return clusterInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *OpenSearchClusterServiceManager) updateClusterIfNeeded(ctx context.Context, clusterObj *ociv1beta1.OpenSearchCluster,
	clusterInstance *opensearch.OpensearchCluster, kind string, req ctrl.Request) error {
	if !isValidUpdate(*clusterObj, *clusterInstance) {
		return nil
	}

	if err := c.UpdateOpenSearchCluster(ctx, clusterObj); err != nil {
		c.Log.ErrorLog(err, "Error while updating OpenSearch cluster")
		c.recordFaultMetric(ctx, kind, req, "Error while updating OpenSearch cluster")
		return err
	}

	clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
		ociv1beta1.Updating, v1.ConditionTrue, "", "OpenSearch cluster update success", c.Log)
	c.Log.InfoLog(fmt.Sprintf("OpenSearch cluster %s updated successfully", safeString(clusterInstance.DisplayName)))

	return nil
}

func (c *OpenSearchClusterServiceManager) finishClusterReconcile(ctx context.Context, kind string, req ctrl.Request,
	clusterObj *ociv1beta1.OpenSearchCluster, clusterInstance *opensearch.OpensearchCluster) servicemanager.OSOKResponse {
	if clusterInstance == nil {
		return servicemanager.OSOKResponse{
			IsSuccessful:    false,
			ShouldRequeue:   true,
			RequeueDuration: openSearchRequeueDuration,
		}
	}

	response := reconcileLifecycleStatus(&clusterObj.Status.OsokStatus, clusterInstance, c.Log)
	if response.IsSuccessful {
		c.recordSuccessMetric(ctx, kind, req, "OpenSearch cluster is Active")
	} else if !response.ShouldRequeue {
		c.recordFaultMetric(ctx, kind, req, "OpenSearch cluster creation failed")
	}

	return response
}

func (c *OpenSearchClusterServiceManager) recordSuccessMetric(ctx context.Context, kind string, req ctrl.Request, message string) {
	if c.Metrics == nil {
		return
	}

	c.Metrics.AddCRSuccessMetrics(ctx, kind, message, req.Name, req.Namespace)
}

func (c *OpenSearchClusterServiceManager) recordFaultMetric(ctx context.Context, kind string, req ctrl.Request, message string) {
	if c.Metrics == nil {
		return
	}

	c.Metrics.AddCRFaultMetrics(ctx, kind, message, req.Name, req.Namespace)
}

func hasExplicitClusterID(clusterObj *ociv1beta1.OpenSearchCluster) bool {
	return strings.TrimSpace(string(clusterObj.Spec.OpenSearchClusterId)) != ""
}

func (c *OpenSearchClusterServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	clusterObj, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Error converting object")
		return true, nil
	}

	clusterId, err := resolveClusterID(clusterObj.Status.OsokStatus.Ocid, clusterObj.Spec.OpenSearchClusterId)
	if err != nil {
		c.Log.InfoLog("No cluster OCID found for deletion, skipping")
		return true, nil
	}

	if err = c.DeleteOpenSearchCluster(ctx, clusterId); err != nil {
		if isNotFoundServiceError(err) {
			return true, nil
		}
		c.Log.ErrorLog(err, "Error deleting OpenSearch cluster")
		return false, err
	}
	c.Log.InfoLog(fmt.Sprintf("OpenSearch cluster %s deletion initiated", clusterId))

	clusterInstance, err := c.GetOpenSearchCluster(ctx, clusterId, nil)
	if err != nil {
		if isNotFoundServiceError(err) {
			return true, nil
		}
		return false, err
	}
	if clusterInstance.LifecycleState == opensearch.OpensearchClusterLifecycleStateDeleted {
		return true, nil
	}
	return false, nil
}

func (c *OpenSearchClusterServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OpenSearchClusterServiceManager) convert(obj runtime.Object) (*ociv1beta1.OpenSearchCluster, error) {
	cluster, ok := obj.(*ociv1beta1.OpenSearchCluster)
	if !ok {
		return nil, fmt.Errorf("failed to convert type assertion for OpenSearchCluster")
	}
	return cluster, nil
}
