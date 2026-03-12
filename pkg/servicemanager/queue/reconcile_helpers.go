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

	ociqueue "github.com/oracle/oci-go-sdk/v65/queue"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/oracle/oci-service-operator/pkg/util"
)

const queueRequeueDuration = 30 * time.Second

func (c *OciQueueServiceManager) resolveQueueForReconcile(ctx context.Context, q *ociv1beta1.OciQueue) (*ociqueue.Queue, *servicemanager.OSOKResponse, error) {
	if strings.TrimSpace(string(q.Spec.QueueId)) != "" {
		return c.bindQueueByID(ctx, q)
	}

	if strings.TrimSpace(string(q.Status.OsokStatus.Ocid)) != "" {
		queueInstance, err := c.GetQueue(ctx, q.Status.OsokStatus.Ocid)
		if err != nil {
			if !isQueueNotFound(err) {
				return nil, nil, err
			}
			q.Status.OsokStatus.Ocid = ""
		} else {
			if queueInstance.LifecycleState == ociqueue.QueueLifecycleStateActive {
				if err := c.UpdateQueue(ctx, q); err != nil {
					return nil, nil, err
				}
			}
			return queueInstance, nil, nil
		}
	}

	return c.createOrLookupQueue(ctx, q)
}

func (c *OciQueueServiceManager) createOrLookupQueue(ctx context.Context, q *ociv1beta1.OciQueue) (*ociqueue.Queue, *servicemanager.OSOKResponse, error) {
	queueOcid, err := c.GetQueueOcid(ctx, *q)
	if err != nil {
		return nil, nil, err
	}
	if queueOcid == nil {
		if _, err := c.CreateQueue(ctx, *q); err != nil {
			q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
			c.Log.ErrorLog(err, "Create OciQueue failed")
			return nil, nil, err
		}
		c.Log.InfoLog(fmt.Sprintf("OciQueue %s creation submitted, waiting for provisioning", q.Spec.DisplayName))
		q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
			ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciQueue Provisioning", c.Log)
		response := servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true, RequeueDuration: queueRequeueDuration}
		return nil, &response, nil
	}

	queueInstance, err := c.GetQueue(ctx, *queueOcid)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting OciQueue by OCID")
		return nil, nil, err
	}
	if queueInstance.LifecycleState == ociqueue.QueueLifecycleStateCreating {
		c.Log.InfoLog(fmt.Sprintf("OciQueue %s is still CREATING", safeString(queueInstance.DisplayName)))
		q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
			ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciQueue Provisioning", c.Log)
		q.Status.OsokStatus.Ocid = ociv1beta1.OCID(safeString(queueInstance.Id))
		response := servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true, RequeueDuration: queueRequeueDuration}
		return nil, &response, nil
	}

	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(safeString(queueInstance.Id))
	c.Log.InfoLog(fmt.Sprintf("OciQueue %s is %s", safeString(queueInstance.DisplayName), queueInstance.LifecycleState))
	return queueInstance, nil, nil
}

func (c *OciQueueServiceManager) bindQueueByID(ctx context.Context, q *ociv1beta1.OciQueue) (*ociqueue.Queue, *servicemanager.OSOKResponse, error) {
	queueInstance, err := c.GetQueue(ctx, q.Spec.QueueId)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting existing OciQueue")
		return nil, nil, err
	}

	q.Status.OsokStatus.Ocid = q.Spec.QueueId
	if err := c.UpdateQueue(ctx, q); err != nil {
		c.Log.ErrorLog(err, "Error while updating OciQueue")
		return nil, nil, err
	}

	c.Log.InfoLog(fmt.Sprintf("OciQueue %s is bound/updated", safeString(queueInstance.DisplayName)))
	return queueInstance, nil, nil
}

func (c *OciQueueServiceManager) finalizeQueueReconcile(ctx context.Context, q *ociv1beta1.OciQueue, queueInstance *ociqueue.Queue) (servicemanager.OSOKResponse, error) {
	q.Status.OsokStatus.Ocid = ociv1beta1.OCID(safeString(queueInstance.Id))
	if q.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		q.Status.OsokStatus.CreatedAt = &now
	}

	switch queueInstance.LifecycleState {
	case ociqueue.QueueLifecycleStateFailed, ociqueue.QueueLifecycleStateDeleted:
		q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("OciQueue %s is %s", safeString(queueInstance.DisplayName), queueInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("OciQueue %s is %s", safeString(queueInstance.DisplayName), queueInstance.LifecycleState))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	case ociqueue.QueueLifecycleStateActive:
		q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("OciQueue %s is %s", safeString(queueInstance.DisplayName), queueInstance.LifecycleState), c.Log)
		_, err := c.addToSecret(ctx, q.Namespace, q.Name, *queueInstance)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				return servicemanager.OSOKResponse{IsSuccessful: true}, nil
			}
			c.Log.InfoLog("Secret creation failed")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	default:
		q.Status.OsokStatus = util.UpdateOSOKStatusCondition(q.Status.OsokStatus,
			ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("OciQueue %s is %s", safeString(queueInstance.DisplayName), queueInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("OciQueue %s is %s, requeueing", safeString(queueInstance.DisplayName), queueInstance.LifecycleState))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true, RequeueDuration: queueRequeueDuration}, nil
	}
}
