/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociqueue "github.com/oracle/oci-go-sdk/v65/queue"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that OciQueueServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciQueueServiceManager{}

// OciQueueServiceManager implements OSOKServiceManager for OCI Queue.
type OciQueueServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        QueueAdminClientInterface
}

// NewOciQueueServiceManager creates a new OciQueueServiceManager.
func NewOciQueueServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciQueueServiceManager {
	return &OciQueueServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciQueue resource against OCI.
//
// Queue creation is asynchronous: the OCI API returns a work request ID rather than a
// queue ID immediately. We therefore use a two-phase approach:
//
//  1. If the queue does not yet exist in OCI, issue CreateQueue and return
//     Provisioning (IsSuccessful=false). The controller will re-reconcile.
//  2. On subsequent reconciles, GetQueueOcid finds the queue (CREATING or ACTIVE)
//     and we proceed normally.
func (c *OciQueueServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	q, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	queueInstance, response, err := c.resolveQueueForReconcile(ctx, q)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	if response != nil {
		return *response, nil
	}

	return c.finalizeQueueReconcile(ctx, q, queueInstance)
}

// Delete handles deletion of the Queue (called by the finalizer).
func (c *OciQueueServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	q, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	targetID, err := servicemanager.ResolveResourceID(q.Status.OsokStatus.Ocid, q.Spec.QueueId)
	if err != nil {
		c.Log.InfoLog("OciQueue has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciQueue %s", targetID))
	if err := c.DeleteQueue(ctx, targetID); err != nil {
		if isQueueNotFound(err) {
			return c.deleteQueueSecret(ctx, q)
		}
		c.Log.ErrorLog(err, "Error while deleting OciQueue")
		return false, err
	}

	queueInstance, err := c.GetQueue(ctx, targetID)
	if err != nil {
		if isQueueNotFound(err) {
			return c.deleteQueueSecret(ctx, q)
		}
		c.Log.ErrorLog(err, "Error while checking OciQueue deletion")
		return false, err
	}

	if queueInstance.LifecycleState == ociqueue.QueueLifecycleStateDeleted {
		return c.deleteQueueSecret(ctx, q)
	}

	return false, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciQueueServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciQueueServiceManager) convert(obj runtime.Object) (*ociv1beta1.OciQueue, error) {
	q, ok := obj.(*ociv1beta1.OciQueue)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciQueue")
	}
	return q, nil
}

func (c *OciQueueServiceManager) deleteQueueSecret(ctx context.Context, q *ociv1beta1.OciQueue) (bool, error) {
	return servicemanager.DeleteOwnedSecretIfPresent(ctx, c.CredentialClient, q.Name, q.Namespace, "OciQueue", q.Name)
}

func isQueueNotFound(err error) bool {
	if err == nil {
		return false
	}
	serviceErr, ok := common.IsServiceError(err)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
