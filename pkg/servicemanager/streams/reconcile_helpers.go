/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams

import (
	"context"
	"errors"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func streamBadRequestCode(err error) (string, bool) {
	var badRequest errorutil.BadRequestOciError
	if !errors.As(err, &badRequest) {
		return "", false
	}
	return badRequest.ErrorCode, true
}

func (c *StreamServiceManager) recordStreamFault(ctx context.Context, kind, reason string, req ctrl.Request) {
	c.Metrics.AddCRFaultMetrics(ctx, kind, reason, req.Name, req.Namespace)
}

func (c *StreamServiceManager) loadStreamInstance(ctx context.Context, streamID ociv1beta1.OCID,
	retryPolicy *common.RetryPolicy, kind string, req ctrl.Request) (*streaming.Stream, error) {
	streamInstance, err := c.GetStream(ctx, streamID, retryPolicy)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting Stream")
		c.recordStreamFault(ctx, kind, "Error while getting Stream", req)
		return nil, err
	}
	return streamInstance, nil
}

func (c *StreamServiceManager) handleCreateStreamError(ctx context.Context, streamObject *ociv1beta1.Stream,
	err error, kind string, req ctrl.Request) error {
	streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
		ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
	c.Log.ErrorLog(err, "Invalid Parameter Error")
	c.recordStreamFault(ctx, kind, "Invalid Parameter Error", req)

	_, classifiedErr := errorutil.OciErrorTypeResponse(err)
	if code, ok := streamBadRequestCode(classifiedErr); ok {
		streamObject.Status.OsokStatus.Message = code
		return classifiedErr
	}

	c.Log.ErrorLog(classifiedErr, "Assertion error for BadRequestOciError")
	return classifiedErr
}

func (c *StreamServiceManager) setStreamStatusID(streamObject *ociv1beta1.Stream, fallbackID ociv1beta1.OCID,
	streamInstance *streaming.Stream) {
	if streamInstance != nil && streamInstance.Id != nil {
		streamObject.Status.OsokStatus.Ocid = ociv1beta1.OCID(*streamInstance.Id)
		return
	}
	streamObject.Status.OsokStatus.Ocid = fallbackID
}

func (c *StreamServiceManager) reconcileStreamLifecycle(ctx context.Context, streamObject *ociv1beta1.Stream,
	streamInstance *streaming.Stream, kind string, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	state := string(streamInstance.LifecycleState)
	displayName := safeStreamString(streamInstance.Name)

	switch streamInstance.LifecycleState {
	case streaming.StreamLifecycleStateFailed, streaming.StreamLifecycleStateDeleted:
		streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("Stream %s is %s", displayName, state), c.Log)
		c.recordStreamFault(ctx, kind, "Failed to Create the Stream", req)
		c.Log.InfoLog(fmt.Sprintf("Stream %s is %s", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	case streaming.StreamLifecycleStateActive:
		streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("Stream %s is Active", displayName), c.Log)
		c.Log.InfoLog(fmt.Sprintf("Stream %s is Active", displayName))
		c.Metrics.AddCRSuccessMetrics(ctx, kind, "Stream in Active state", req.Name, req.Namespace)
		if _, err := c.addToSecret(ctx, streamObject.Namespace, streamObject.Name, *streamInstance); err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	default:
		streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
			ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("Stream %s is %s", displayName, state), c.Log)
		c.Log.InfoLog(fmt.Sprintf("Stream %s is %s, requeueing", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}, nil
	}
}
