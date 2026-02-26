/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/opensearch"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type OpenSearchClusterServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
	ociClient        OpensearchClusterClientInterface
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

	var clusterInstance *opensearch.OpensearchCluster

	if strings.TrimSpace(string(clusterObj.Spec.OpenSearchClusterId)) == "" {
		// No explicit ID: check if cluster exists by name, or create it
		clusterOcid, err := c.GetOpenSearchClusterOCID(ctx, *clusterObj)
		if err != nil {
			c.Log.ErrorLog(err, "Error while looking up OpenSearch cluster")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Failed to get OpenSearch cluster", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if clusterOcid != nil {
			// Cluster already exists — fetch and possibly update
			clusterInstance, err = c.GetOpenSearchCluster(ctx, *clusterOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error fetching existing OpenSearch cluster")
				c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Error fetching OpenSearch cluster", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			if isValidUpdate(*clusterObj, *clusterInstance) {
				if err = c.UpdateOpenSearchCluster(ctx, clusterObj); err != nil {
					c.Log.ErrorLog(err, "Error while updating OpenSearch cluster")
					c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
						"Error while updating OpenSearch cluster", req.Name, req.Namespace)
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
					ociv1beta1.Updating, v1.ConditionTrue, "", "OpenSearch cluster update success", c.Log)
				c.Log.InfoLog(fmt.Sprintf("OpenSearch cluster %s updated successfully", *clusterInstance.DisplayName))
			}
		} else {
			// Create new cluster
			_, err := c.CreateOpenSearchCluster(ctx, *clusterObj)
			if err != nil {
				clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Error creating OpenSearch cluster")
				c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Error creating OpenSearch cluster", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			// Create returns a work request ID, not a cluster OCID.
			// Set Provisioning state and requeue; next reconcile will find it by name.
			c.Log.InfoLog(fmt.Sprintf("OpenSearch cluster %s create initiated, provisioning", clusterObj.Spec.DisplayName))
			clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "OpenSearch cluster is Provisioning", c.Log)
			return servicemanager.OSOKResponse{IsSuccessful: false}, nil
		}
	} else {
		// Bind to existing cluster by explicit OCID
		clusterInstance, err = c.GetOpenSearchCluster(ctx, clusterObj.Spec.OpenSearchClusterId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error fetching OpenSearch cluster by ID")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Error fetching OpenSearch cluster", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if isValidUpdate(*clusterObj, *clusterInstance) {
			if err = c.UpdateOpenSearchCluster(ctx, clusterObj); err != nil {
				c.Log.ErrorLog(err, "Error while updating OpenSearch cluster")
				c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Error while updating OpenSearch cluster", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
				ociv1beta1.Updating, v1.ConditionTrue, "", "OpenSearch cluster update success", c.Log)
		} else {
			clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
				ociv1beta1.Active, v1.ConditionTrue, "", "OpenSearch cluster bound", c.Log)
			now := metav1.NewTime(time.Now())
			clusterObj.Status.OsokStatus.CreatedAt = &now
		}
	}

	if clusterInstance == nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	clusterObj.Status.OsokStatus.Ocid = ociv1beta1.OCID(*clusterInstance.Id)
	if clusterObj.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		clusterObj.Status.OsokStatus.CreatedAt = &now
	}

	switch clusterInstance.LifecycleState {
	case opensearch.OpensearchClusterLifecycleStateFailed:
		clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("OpenSearch cluster %s creation failed", *clusterInstance.DisplayName), c.Log)
		c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"OpenSearch cluster creation failed", req.Name, req.Namespace)
	case opensearch.OpensearchClusterLifecycleStateActive:
		clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("OpenSearch cluster %s is Active", *clusterInstance.DisplayName), c.Log)
		c.Metrics.AddCRSuccessMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"OpenSearch cluster is Active", req.Name, req.Namespace)
	default:
		clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
			ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("OpenSearch cluster %s lifecycle state: %s", *clusterInstance.DisplayName, clusterInstance.LifecycleState), c.Log)
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func isValidUpdate(clusterObj ociv1beta1.OpenSearchCluster, clusterInstance opensearch.OpensearchCluster) bool {
	definedTagUpdated := false
	if clusterObj.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&clusterObj.Spec.DefinedTags); !reflect.DeepEqual(clusterInstance.DefinedTags, defTag) {
			definedTagUpdated = true
		}
	}

	displayNameUpdated := clusterObj.Spec.DisplayName != "" && clusterInstance.DisplayName != nil &&
		clusterObj.Spec.DisplayName != *clusterInstance.DisplayName

	return displayNameUpdated ||
		clusterObj.Spec.FreeFormTags != nil && !reflect.DeepEqual(clusterObj.Spec.FreeFormTags, clusterInstance.FreeformTags) ||
		definedTagUpdated
}

func (c *OpenSearchClusterServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	clusterObj, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Error converting object")
		return true, nil
	}

	clusterId := clusterObj.Status.OsokStatus.Ocid
	if strings.TrimSpace(string(clusterId)) == "" {
		clusterId = clusterObj.Spec.OpenSearchClusterId
	}
	if strings.TrimSpace(string(clusterId)) == "" {
		c.Log.InfoLog("No cluster OCID found for deletion, skipping")
		return true, nil
	}

	if err = c.DeleteOpenSearchCluster(ctx, clusterId); err != nil {
		c.Log.ErrorLog(err, "Error deleting OpenSearch cluster")
		return true, nil
	}
	c.Log.InfoLog(fmt.Sprintf("OpenSearch cluster %s deletion initiated", clusterId))
	return true, nil
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

func (c *OpenSearchClusterServiceManager) getClusterRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(opensearch.GetOpensearchClusterResponse); ok {
			return resp.LifecycleState == opensearch.OpensearchClusterLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
