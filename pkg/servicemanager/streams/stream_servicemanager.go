/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type StreamServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
	ociClient        StreamAdminClientInterface
}

func NewStreamServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger, metrics *metrics.Metrics) *StreamServiceManager {
	return &StreamServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
		Metrics:          metrics,
	}
}

func (c *StreamServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	streamObject, err := c.convert(obj)

	// if error happen while object conversion
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var streamInstance *streaming.Stream
	var streamID ociv1beta1.OCID
	var retryPolicy *common.RetryPolicy

	if strings.TrimSpace(string(streamObject.Spec.StreamId)) == "" {
		if streamObject.Spec.Name == "" {
			return servicemanager.OSOKResponse{IsSuccessful: false}, errors.New("Can't able to create the stream")
		}

		// check for whether same name stream exists or not in ACTIVE, UPDATING OR CREATING Phase
		streamOcid, err := c.GetStreamOCID(ctx, *streamObject, "CREATE")
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting Stream using Id")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Failed to get the Stream", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if streamOcid != nil {
			streamID = *streamOcid
			streamInstance, err = c.GetStream(ctx, streamID, nil)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting Stream")
				c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Error while getting Stream", req.Name, req.Namespace)
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			//creating the fresh request for creating the Streams
			resp, err := c.CreateStream(ctx, *streamObject)
			if err != nil {
				streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Invalid Parameter Error")
				c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
					"Invalid Parameter Error", req.Name, req.Namespace)
				_, err := errorutil.OciErrorTypeResponse(err)
				if _, ok := err.(errorutil.BadRequestOciError); !ok {
					c.Log.ErrorLog(err, "Assertion error for BadRequestOciError")
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				} else {
					streamObject.Status.OsokStatus.Message = err.(common.ServiceError).GetCode()
					return servicemanager.OSOKResponse{IsSuccessful: false}, err
				}
			}

			//create the stream then retry to become it active
			c.Log.InfoLog(fmt.Sprintf("Stream %s is getting Provisioned", streamObject.Spec.Name))
			streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "Stream is getting Provisioned", c.Log)
			streamID = ociv1beta1.OCID(*resp.Id)
			retry := c.getStreamRetryPolicy(9)
			retryPolicy = &retry
		}
	} else {
		// stream already exists update the configuration or modify the changes
		// Bind CRD with an existing Stream.
		streamID = streamObject.Spec.StreamId
		streamInstance, err = c.GetStream(ctx, streamID, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting Stream")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Error while getting Stream", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	if streamInstance == nil {
		streamInstance, err = c.GetStream(ctx, streamID, retryPolicy)
		if err != nil {
			streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "Error while getting the stream", err.Error(), c.Log)
			c.Log.ErrorLog(err, "Error while getting Stream")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Error while getting Stream", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	if streamInstance.Id != nil {
		streamObject.Status.OsokStatus.Ocid = ociv1beta1.OCID(*streamInstance.Id)
	} else {
		streamObject.Status.OsokStatus.Ocid = streamID
	}

	if isValidUpdate(*streamObject, *streamInstance) {
		if err = c.UpdateStream(ctx, streamObject); err != nil {
			c.Log.ErrorLog(err, "Error while updating Stream")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Error while updating Stream", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
			ociv1beta1.Updating, v1.ConditionTrue, "", "Stream Update success", c.Log)
		c.Log.InfoLog(fmt.Sprintf("Stream %s is updated successfully", safeStreamString(streamInstance.Name)))

		streamInstance, err = c.GetStream(ctx, streamObject.Status.OsokStatus.Ocid, nil)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting Stream after update")
			c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
				"Error while getting Stream after update", req.Name, req.Namespace)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		if streamInstance.Id != nil {
			streamObject.Status.OsokStatus.Ocid = ociv1beta1.OCID(*streamInstance.Id)
		}
	}

	if streamObject.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		streamObject.Status.OsokStatus.CreatedAt = &now
	}

	if streamInstance.LifecycleState == "FAILED" || streamInstance.LifecycleState == "DELETED" {
		streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("Stream %s is %s", safeStreamString(streamInstance.Name), streamInstance.LifecycleState), c.Log)
		c.Metrics.AddCRFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind, "Failed to Create the Stream",
			req.Name, req.Namespace)
		c.Log.InfoLog(fmt.Sprintf("Stream %s is %s", safeStreamString(streamInstance.Name), streamInstance.LifecycleState))
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	if streamInstance.LifecycleState != "ACTIVE" {
		streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
			ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("Stream %s is %s", safeStreamString(streamInstance.Name), streamInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("Stream %s is %s, requeueing", safeStreamString(streamInstance.Name), streamInstance.LifecycleState))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}, nil
	}

	streamObject.Status.OsokStatus = util.UpdateOSOKStatusCondition(streamObject.Status.OsokStatus,
		ociv1beta1.Active, v1.ConditionTrue, "",
		fmt.Sprintf("Stream %s is Active", safeStreamString(streamInstance.Name)), c.Log)
	c.Log.InfoLog(fmt.Sprintf("Stream %s is Active", safeStreamString(streamInstance.Name)))
	c.Metrics.AddCRSuccessMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind, "Stream in Active state",
		req.Name, req.Namespace)
	_, err = c.addToSecret(ctx, streamObject.Namespace, streamObject.Name, *streamInstance)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		}
		c.Log.InfoLog(fmt.Sprintf("Secret creation got failed"))
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

func isValidUpdate(streamObject ociv1beta1.Stream, streamInstance streaming.Stream) bool {
	definedTagUpdated := false
	if streamObject.Spec.DefinedTags != nil {
		if defTag := *util.ConvertToOciDefinedTags(&streamObject.Spec.DefinedTags); !reflect.DeepEqual(streamInstance.DefinedTags, defTag) {
			definedTagUpdated = true
		}
	}

	return streamObject.Spec.StreamPoolId != "" && string(streamObject.Spec.StreamPoolId) != *streamInstance.StreamPoolId ||
		streamObject.Spec.FreeFormTags != nil && !reflect.DeepEqual(streamObject.Spec.FreeFormTags, streamInstance.FreeformTags) ||
		definedTagUpdated
}

func (c *StreamServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	streamObject, err := c.convert(obj)

	if err != nil {
		c.Log.ErrorLog(err, "Error while converting the object")
		return false, err
	}

	streamID, err := c.resolveStreamIDForDelete(ctx, streamObject)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting the stream ocid")
		return false, err
	}
	if strings.TrimSpace(string(streamID)) == "" {
		return true, nil
	}

	streamObject.Spec.StreamId = streamID
	_, err = c.DeleteStream(ctx, *streamObject)
	if err != nil {
		if isStreamNotFound(err) {
			return c.completeStreamDeletion(ctx, streamObject)
		}
		c.Log.ErrorLog(err, "Error while Deleting the Stream")
		return false, err
	}

	streamInstance, err := c.GetStream(ctx, streamObject.Spec.StreamId, nil)
	if err != nil {
		if isStreamNotFound(err) {
			return c.completeStreamDeletion(ctx, streamObject)
		}
		c.Log.ErrorLog(err, "Error while Getting the Stream")
		return false, err
	}
	if streamInstance.LifecycleState == "DELETED" {
		return c.completeStreamDeletion(ctx, streamObject)
	}
	if streamInstance.LifecycleState == "DELETING" {
		return false, nil
	}
	return false, nil
}

func (c *StreamServiceManager) resolveStreamIDForDelete(ctx context.Context, streamObject *ociv1beta1.Stream) (ociv1beta1.OCID, error) {
	if strings.TrimSpace(string(streamObject.Spec.StreamId)) != "" {
		return streamObject.Spec.StreamId, nil
	}
	if strings.TrimSpace(string(streamObject.Status.OsokStatus.Ocid)) != "" {
		return streamObject.Status.OsokStatus.Ocid, nil
	}
	streamOcid, err := c.GetStreamOcid(ctx, *streamObject)
	if err != nil {
		return "", err
	}
	if streamOcid != nil {
		return *streamOcid, nil
	}
	streamOcid, err = c.GetStreamOCID(ctx, *streamObject, "DELETE")
	if err != nil {
		return "", err
	}
	if streamOcid != nil {
		return *streamOcid, nil
	}
	return "", nil
}

func (c *StreamServiceManager) completeStreamDeletion(ctx context.Context, streamObject *ociv1beta1.Stream) (bool, error) {
	if _, err := c.deleteFromSecret(ctx, streamObject.Namespace, streamObject.Name); err != nil {
		c.Log.ErrorLog(err, "Secret deletion failed")
		return false, err
	}
	return true, nil
}

func isStreamNotFound(err error) bool {
	if err == nil {
		return false
	}
	serviceErr, ok := common.IsServiceError(err)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func safeStreamString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (c *StreamServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {

	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *StreamServiceManager) convert(obj runtime.Object) (*ociv1beta1.Stream, error) {
	deepcopy, err := obj.(*ociv1beta1.Stream)
	if !err {
		return nil, fmt.Errorf("failed to convert the type assertion for Stream")
	}
	return deepcopy, nil
}

func (c *StreamServiceManager) getStreamRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(streaming.GetStreamResponse); ok {
			return resp.LifecycleState == "CREATING"
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

func (c *StreamServiceManager) deleteStreamRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		if resp, ok := response.Response.(streaming.GetStreamResponse); ok {
			return resp.LifecycleState == "DELETING"
		}
		return true
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
