/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociqueue "github.com/oracle/oci-go-sdk/v65/queue"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/util"
)

// Compile-time check that OciQueueServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciQueueServiceManager{}

// OciQueueServiceManager implements OSOKServiceManager for OCI Queue.
type OciQueueServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
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

	var queueInstance *ociqueue.Queue

	if strings.TrimSpace(string(q.Spec.QueueId)) == "" {
		// No explicit ID — look up by display name or create.
		queueOcid, err := c.GetQueueOcid(ctx, *q)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if queueOcid == nil {
			// Queue does not exist yet — issue an async create and return Provisioning.
			// The controller will re-reconcile; on the next pass GetQueueOcid will find
			// the queue in CREATING state and we will reach the branch below.
			_, err := c.CreateQueue(ctx, *q)
			if err != nil {
				q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciQueue failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
			c.Log.InfoLog(fmt.Sprintf("OciQueue %s creation submitted, waiting for provisioning", q.Spec.DisplayName))
			q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciQueue Provisioning", c.Log)
			return servicemanager.OSOKResponse{IsSuccessful: false}, nil
		}

		// Queue exists (CREATING or ACTIVE) — fetch full details.
		queueInstance, err = c.GetQueue(ctx, *queueOcid)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting OciQueue by OCID")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if queueInstance.LifecycleState == ociqueue.QueueLifecycleStateCreating {
			// Still provisioning — update status and re-reconcile.
			c.Log.InfoLog(fmt.Sprintf("OciQueue %s is still CREATING", safeString(queueInstance.DisplayName)))
			q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciQueue Provisioning", c.Log)
			q.Status.OsokStatus.Ocid = ociv1beta1.OCID(*queueInstance.Id)
			return servicemanager.OSOKResponse{IsSuccessful: false}, nil
		}

		q.Status.OsokStatus.Ocid = ociv1beta1.OCID(*queueInstance.Id)
		q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("OciQueue %s is %s", safeString(queueInstance.DisplayName), queueInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("OciQueue %s is %s", safeString(queueInstance.DisplayName), queueInstance.LifecycleState))

	} else {
		// Bind to an existing queue by ID.
		queueInstance, err = c.GetQueue(ctx, q.Spec.QueueId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciQueue")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateQueue(ctx, q); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciQueue")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "OciQueue Bound/Updated", c.Log)
		c.Log.InfoLog(fmt.Sprintf("OciQueue %s is bound/updated", safeString(queueInstance.DisplayName)))
	}

	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(*queueInstance.Id)
	if q.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		q.Status.OsokStatus.CreatedAt = &now
	}

	if queueInstance.LifecycleState == ociqueue.QueueLifecycleStateFailed {
		q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("OciQueue %s creation Failed", safeString(queueInstance.DisplayName)), c.Log)
		c.Log.InfoLog(fmt.Sprintf("OciQueue %s creation Failed", safeString(queueInstance.DisplayName)))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	_, err = c.addToSecret(ctx, q.Namespace, q.Name, *queueInstance)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		}
		c.Log.InfoLog("Secret creation failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the Queue (called by the finalizer).
func (c *OciQueueServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	q, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if q.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciQueue has no OCID, nothing to delete")
		return true, nil
	}

	c.Log.InfoLog(fmt.Sprintf("Deleting OciQueue %s", q.Status.OsokStatus.Ocid))
	if err := c.DeleteQueue(ctx, q.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciQueue")
		return false, err
	}

	if _, err := c.CredentialClient.DeleteSecret(ctx, q.Name, q.Namespace); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciQueue secret")
	}

	return true, nil
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

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
