/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociRedis "github.com/oracle/oci-go-sdk/v65/redis"
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

type RedisClusterServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
}

func NewRedisClusterServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger, metrics *metrics.Metrics) *RedisClusterServiceManager {
	return &RedisClusterServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
		Metrics:          metrics,
	}
}

func (c *RedisClusterServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	clusterObj, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var clusterInstance *ociRedis.RedisCluster

	if strings.TrimSpace(string(clusterObj.Spec.RedisClusterId)) == "" {
		if clusterObj.Spec.DisplayName == "" {
			return servicemanager.OSOKResponse{IsSuccessful: false}, fmt.Errorf("displayName is required to create a RedisCluster")
		}

		// Check if a cluster with this display name already exists
		existingOcid, err := c.GetRedisClusterOcid(ctx, *clusterObj)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting RedisCluster by display name")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Failed to get the RedisCluster", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if existingOcid != nil {
			// Cluster exists — bind to it and update if needed
			clusterInstance, err = c.GetRedisCluster(ctx, *existingOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting RedisCluster")
				c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Error while getting RedisCluster", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			clusterObj.Spec.RedisClusterId = *existingOcid
			clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
				ociv1beta1.Updating, v1.ConditionTrue, "", "RedisCluster Bound success", c.Log)
			c.Log.InfoLog(fmt.Sprintf("RedisCluster %s already exists, bound successfully", *clusterInstance.DisplayName))
		} else {
			// Create a new cluster
			resp, err := c.CreateRedisCluster(ctx, *clusterObj)
			if err != nil {
				clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Error creating RedisCluster")
				c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Error creating RedisCluster", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			c.Log.InfoLog(fmt.Sprintf("RedisCluster %s is getting Provisioned", clusterObj.Spec.DisplayName))
			clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "RedisCluster is getting Provisioned", c.Log)

			retryPolicy := c.getRedisClusterCreateRetryPolicy(20)
			clusterInstance, err = c.GetRedisCluster(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "Error while getting the RedisCluster", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Error while getting RedisCluster after creation")
				c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Error while getting RedisCluster", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}
	} else {
		// Bind to an existing cluster by OCID
		clusterInstance, err = c.GetRedisCluster(ctx, clusterObj.Spec.RedisClusterId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting RedisCluster")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Error while getting RedisCluster", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "RedisCluster Bound success", c.Log)
		clusterObj.Status.OsokStatus.Ocid = clusterObj.Spec.RedisClusterId
		now := metav1.NewTime(time.Now())
		clusterObj.Status.OsokStatus.CreatedAt = &now
		c.Log.InfoLog(fmt.Sprintf("RedisCluster %s bounded successfully", *clusterInstance.DisplayName))
	}

	clusterObj.Status.OsokStatus.Ocid = ociv1beta1.OCID(*clusterInstance.Id)
	if clusterObj.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		clusterObj.Status.OsokStatus.CreatedAt = &now
	}

	if clusterInstance.LifecycleState == ociRedis.RedisClusterLifecycleStateFailed {
		clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("RedisCluster %s creation Failed", *clusterInstance.DisplayName), c.Log)
		c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Failed to Create the RedisCluster", req.Name, req.Namespace)
		c.Log.InfoLog(fmt.Sprintf("RedisCluster %s creation Failed", *clusterInstance.DisplayName))
	} else {
		clusterObj.Status.OsokStatus = util.UpdateOSOKStatusCondition(clusterObj.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("RedisCluster %s is Active", *clusterInstance.DisplayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("RedisCluster %s is Active", *clusterInstance.DisplayName))
		c.Metrics.AddCRSuccessMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"RedisCluster in Active state", req.Name, req.Namespace)

		if _, err := c.addToSecret(ctx, clusterObj.Namespace, clusterObj.Name, *clusterInstance); err != nil {
			c.Log.InfoLog("Secret creation failed")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func (c *RedisClusterServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	clusterObj, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Error while converting the object")
		return true, nil
	}

	clusterId := clusterObj.Spec.RedisClusterId
	if clusterId == "" {
		clusterId = clusterObj.Status.OsokStatus.Ocid
	}

	if clusterId == "" {
		c.Log.InfoLog("RedisCluster OCID not found, skipping delete")
		return true, nil
	}

	_, err = c.DeleteRedisCluster(ctx, *clusterObj)
	if err != nil {
		c.Log.ErrorLog(err, "Error while deleting the RedisCluster")
		return true, nil
	}

	retryPolicy := c.getRedisClusterDeleteRetryPolicy(20)
	clusterInstance, err := c.GetRedisCluster(ctx, clusterId, &retryPolicy)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting RedisCluster after delete")
		return true, nil
	}

	if clusterInstance.LifecycleState == ociRedis.RedisClusterLifecycleStateDeleted ||
		clusterInstance.LifecycleState == ociRedis.RedisClusterLifecycleStateDeleting {
		if _, err := c.deleteFromSecret(ctx, clusterObj.Namespace, clusterObj.Name); err != nil {
			c.Log.ErrorLog(err, "Secret deletion failed")
			return true, err
		}
		return true, nil
	}

	return true, nil
}

func (c *RedisClusterServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *RedisClusterServiceManager) convert(obj runtime.Object) (*ociv1beta1.RedisCluster, error) {
	cluster, ok := obj.(*ociv1beta1.RedisCluster)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for RedisCluster")
	}
	return cluster, nil
}

func (c *RedisClusterServiceManager) getRedisClusterCreateRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(ociRedis.GetRedisClusterResponse); ok {
			return resp.LifecycleState == ociRedis.RedisClusterLifecycleStateCreating
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

func (c *RedisClusterServiceManager) getRedisClusterDeleteRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(ociRedis.GetRedisClusterResponse); ok {
			return resp.LifecycleState == ociRedis.RedisClusterLifecycleStateDeleting
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
