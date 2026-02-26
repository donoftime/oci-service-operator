/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/redis"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/util"
)

// Compile-time check that RedisClusterServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &RedisClusterServiceManager{}

// RedisClusterServiceManager implements OSOKServiceManager for OCI Cache with Redis.
type RedisClusterServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        RedisClusterClientInterface
}

// NewRedisClusterServiceManager creates a new RedisClusterServiceManager.
func NewRedisClusterServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *RedisClusterServiceManager {
	return &RedisClusterServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the RedisCluster resource against OCI.
func (c *RedisClusterServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	cluster, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var clusterInstance *redis.RedisCluster

	if strings.TrimSpace(string(cluster.Spec.RedisClusterId)) == "" {
		// No ID provided — check by display name or create
		clusterOcid, err := c.GetRedisClusterOcid(ctx, *cluster)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if clusterOcid == nil {
			// Create a new Redis cluster
			resp, err := c.CreateRedisCluster(ctx, *cluster)
			if err != nil {
				cluster.Status.OsokStatus = util.UpdateOSOKStatusCondition(cluster.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Create RedisCluster failed")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
				cluster.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
				c.Log.ErrorLog(err, "Create RedisCluster bad request")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}

			c.Log.InfoLog(fmt.Sprintf("RedisCluster %s is Provisioning", cluster.Spec.DisplayName))
			cluster.Status.OsokStatus = util.UpdateOSOKStatusCondition(cluster.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "RedisCluster Provisioning", c.Log)
			cluster.Status.OsokStatus.Ocid = ociv1beta1.OCID(*resp.Id)

			retryPolicy := c.getRetryPolicy(30)
			clusterInstance, err = c.GetRedisCluster(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting RedisCluster after create")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			c.Log.InfoLog(fmt.Sprintf("Getting existing RedisCluster %s", *clusterOcid))
			clusterInstance, err = c.GetRedisCluster(ctx, *clusterOcid, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting RedisCluster by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}

		cluster.Status.OsokStatus.Ocid = ociv1beta1.OCID(*clusterInstance.Id)
		cluster.Status.OsokStatus = util.UpdateOSOKStatusCondition(cluster.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("RedisCluster %s is %s", *clusterInstance.DisplayName, clusterInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("RedisCluster %s is %s", *clusterInstance.DisplayName, clusterInstance.LifecycleState))

	} else {
		// Bind to an existing cluster by ID
		clusterInstance, err = c.GetRedisCluster(ctx, cluster.Spec.RedisClusterId, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing RedisCluster")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateRedisCluster(ctx, cluster); err != nil {
			c.Log.ErrorLog(err, "Error while updating RedisCluster")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		cluster.Status.OsokStatus = util.UpdateOSOKStatusCondition(cluster.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "RedisCluster Bound/Updated", c.Log)
		c.Log.InfoLog(fmt.Sprintf("RedisCluster %s is bound/updated", *clusterInstance.DisplayName))
	}

	cluster.Status.OsokStatus.Ocid = ociv1beta1.OCID(*clusterInstance.Id)
	if cluster.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		cluster.Status.OsokStatus.CreatedAt = &now
	}

	if clusterInstance.LifecycleState == redis.RedisClusterLifecycleStateFailed {
		cluster.Status.OsokStatus = util.UpdateOSOKStatusCondition(cluster.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("RedisCluster %s creation Failed", *clusterInstance.DisplayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("RedisCluster %s creation Failed", *clusterInstance.DisplayName))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	_, err = c.addToSecret(ctx, cluster.Namespace, cluster.Name, *clusterInstance)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		}
		c.Log.InfoLog("Secret creation failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the Redis cluster (called by the finalizer).
func (c *RedisClusterServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	cluster, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if cluster.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("RedisCluster has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting RedisCluster %s", cluster.Status.OsokStatus.Ocid))
	if err := c.DeleteRedisCluster(ctx, cluster.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting RedisCluster")
		return false, err
	}

	if _, err := c.CredentialClient.DeleteSecret(ctx, cluster.Name, cluster.Namespace); err != nil {
		c.Log.ErrorLog(err, "Error while deleting RedisCluster secret")
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
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
