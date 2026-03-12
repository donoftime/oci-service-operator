/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import (
	"context"
	goerrors "errors"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/redis"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	clusterInstance, response, done, err := c.resolveClusterForReconcile(ctx, cluster)
	if err != nil || done {
		return response, err
	}

	reconcileResponse := reconcileLifecycleStatus(&cluster.Status.OsokStatus, clusterInstance, c.Log)
	if !reconcileResponse.IsSuccessful {
		return reconcileResponse, nil
	}

	_, err = c.addToSecret(ctx, cluster.Namespace, cluster.Name, *clusterInstance)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		}
		c.Log.InfoLog("Secret creation failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return reconcileResponse, nil
}

func (c *RedisClusterServiceManager) resolveClusterForReconcile(ctx context.Context,
	cluster *ociv1beta1.RedisCluster) (*redis.RedisCluster, servicemanager.OSOKResponse, bool, error) {
	if strings.TrimSpace(string(cluster.Spec.RedisClusterId)) == "" {
		return c.resolveManagedCluster(ctx, cluster)
	}

	return c.resolveBoundCluster(ctx, cluster)
}

func (c *RedisClusterServiceManager) resolveManagedCluster(ctx context.Context,
	cluster *ociv1beta1.RedisCluster) (*redis.RedisCluster, servicemanager.OSOKResponse, bool, error) {
	clusterOcid, err := c.GetRedisClusterOcid(ctx, *cluster)
	if err != nil {
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if clusterOcid == nil {
		return c.createManagedCluster(ctx, cluster)
	}

	c.Log.InfoLog(fmt.Sprintf("Getting existing RedisCluster %s", *clusterOcid))
	clusterInstance, err := c.GetRedisCluster(ctx, *clusterOcid, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting RedisCluster by OCID")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return clusterInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *RedisClusterServiceManager) createManagedCluster(ctx context.Context,
	cluster *ociv1beta1.RedisCluster) (*redis.RedisCluster, servicemanager.OSOKResponse, bool, error) {
	resp, err := c.CreateRedisCluster(ctx, *cluster)
	if err != nil {
		return c.handleCreateRedisClusterError(cluster, err)
	}

	c.markRedisClusterProvisioning(cluster, *resp.Id)
	retryPolicy := c.getRetryPolicy(30)
	clusterInstance, err := c.GetRedisCluster(ctx, ociv1beta1.OCID(*resp.Id), &retryPolicy)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting RedisCluster after create")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return clusterInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *RedisClusterServiceManager) resolveBoundCluster(ctx context.Context,
	cluster *ociv1beta1.RedisCluster) (*redis.RedisCluster, servicemanager.OSOKResponse, bool, error) {
	clusterInstance, err := c.GetRedisCluster(ctx, cluster.Spec.RedisClusterId, nil)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting existing RedisCluster")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	if err = c.UpdateRedisCluster(ctx, cluster); err != nil {
		c.Log.ErrorLog(err, "Error while updating RedisCluster")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}

	return clusterInstance, servicemanager.OSOKResponse{}, false, nil
}

func (c *RedisClusterServiceManager) handleCreateRedisClusterError(cluster *ociv1beta1.RedisCluster,
	err error) (*redis.RedisCluster, servicemanager.OSOKResponse, bool, error) {
	cluster.Status.OsokStatus = util.UpdateOSOKStatusCondition(cluster.Status.OsokStatus,
		ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
	var badRequestErr errorutil.BadRequestOciError
	if !goerrors.As(err, &badRequestErr) {
		c.Log.ErrorLog(err, "Create RedisCluster failed")
		return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
	}
	if serviceErr, ok := common.IsServiceError(err); ok {
		cluster.Status.OsokStatus.Message = serviceErr.GetCode()
	}
	c.Log.ErrorLog(err, "Create RedisCluster bad request")
	return nil, servicemanager.OSOKResponse{IsSuccessful: false}, true, err
}

func (c *RedisClusterServiceManager) markRedisClusterProvisioning(cluster *ociv1beta1.RedisCluster, clusterID string) {
	c.Log.InfoLog(fmt.Sprintf("RedisCluster %s is Provisioning", cluster.Spec.DisplayName))
	cluster.Status.OsokStatus = util.UpdateOSOKStatusCondition(cluster.Status.OsokStatus,
		ociv1beta1.Provisioning, v1.ConditionTrue, "", "RedisCluster Provisioning", c.Log)
	cluster.Status.OsokStatus.Ocid = ociv1beta1.OCID(clusterID)
}

// Delete handles deletion of the Redis cluster (called by the finalizer).
func (c *RedisClusterServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	cluster, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := resolveClusterID(cluster.Status.OsokStatus.Ocid, cluster.Spec.RedisClusterId)
	if err != nil {
		c.Log.InfoLog("RedisCluster has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting RedisCluster %s", targetID))
	if err := c.DeleteRedisCluster(ctx, targetID); err != nil {
		if isNotFoundServiceError(err) {
			return true, nil
		}
		c.Log.ErrorLog(err, "Error while deleting RedisCluster")
		return false, err
	}

	clusterInstance, err := c.GetRedisCluster(ctx, targetID, nil)
	if err != nil {
		if isNotFoundServiceError(err) {
			if _, err := servicemanager.DeleteOwnedSecretIfPresent(ctx, c.CredentialClient, cluster.Name, cluster.Namespace, "RedisCluster", cluster.Name); err != nil {
				c.Log.ErrorLog(err, "Error while deleting RedisCluster secret")
				return false, err
			}
			return true, nil
		}
		return false, err
	}
	if clusterInstance.LifecycleState == redis.RedisClusterLifecycleStateDeleted {
		if _, err := servicemanager.DeleteOwnedSecretIfPresent(ctx, c.CredentialClient, cluster.Name, cluster.Namespace, "RedisCluster", cluster.Name); err != nil {
			c.Log.ErrorLog(err, "Error while deleting RedisCluster secret")
			return false, err
		}
		return true, nil
	}

	return false, nil
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
