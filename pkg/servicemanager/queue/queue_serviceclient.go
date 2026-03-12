/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"fmt"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociqueue "github.com/oracle/oci-go-sdk/v65/queue"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
)

// QueueAdminClientInterface defines the OCI operations used by OciQueueServiceManager.
type QueueAdminClientInterface interface {
	CreateQueue(ctx context.Context, request ociqueue.CreateQueueRequest) (ociqueue.CreateQueueResponse, error)
	GetQueue(ctx context.Context, request ociqueue.GetQueueRequest) (ociqueue.GetQueueResponse, error)
	ListQueues(ctx context.Context, request ociqueue.ListQueuesRequest) (ociqueue.ListQueuesResponse, error)
	ChangeQueueCompartment(ctx context.Context, request ociqueue.ChangeQueueCompartmentRequest) (ociqueue.ChangeQueueCompartmentResponse, error)
	UpdateQueue(ctx context.Context, request ociqueue.UpdateQueueRequest) (ociqueue.UpdateQueueResponse, error)
	DeleteQueue(ctx context.Context, request ociqueue.DeleteQueueRequest) (ociqueue.DeleteQueueResponse, error)
}

func getQueueAdminClient(provider common.ConfigurationProvider) (ociqueue.QueueAdminClient, error) {
	return ociqueue.NewQueueAdminClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *OciQueueServiceManager) getOCIClient() (QueueAdminClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getQueueAdminClient(c.Provider)
}

// CreateQueue calls the OCI API to create a new Queue and returns the work request ID.
func (c *OciQueueServiceManager) CreateQueue(ctx context.Context, q ociv1beta1.OciQueue) (string, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return "", err
	}

	c.Log.DebugLog("Creating OciQueue", "name", q.Spec.DisplayName)

	details := ociqueue.CreateQueueDetails{
		DisplayName:   common.String(q.Spec.DisplayName),
		CompartmentId: common.String(string(q.Spec.CompartmentId)),
		FreeformTags:  q.Spec.FreeFormTags,
	}

	if q.Spec.RetentionInSeconds > 0 {
		details.RetentionInSeconds = common.Int(q.Spec.RetentionInSeconds)
	}
	if q.Spec.VisibilityInSeconds > 0 {
		details.VisibilityInSeconds = common.Int(q.Spec.VisibilityInSeconds)
	}
	if q.Spec.TimeoutInSeconds > 0 {
		details.TimeoutInSeconds = common.Int(q.Spec.TimeoutInSeconds)
	}
	if q.Spec.DeadLetterQueueDeliveryCount > 0 {
		details.DeadLetterQueueDeliveryCount = common.Int(q.Spec.DeadLetterQueueDeliveryCount)
	}
	if string(q.Spec.CustomEncryptionKeyId) != "" {
		details.CustomEncryptionKeyId = common.String(string(q.Spec.CustomEncryptionKeyId))
	}
	if q.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&q.Spec.DefinedTags)
	}

	req := ociqueue.CreateQueueRequest{
		CreateQueueDetails: details,
	}

	resp, err := client.CreateQueue(ctx, req)
	if err != nil {
		return "", err
	}
	if resp.OpcWorkRequestId == nil {
		return "", fmt.Errorf("CreateQueue returned nil work request ID")
	}
	return *resp.OpcWorkRequestId, nil
}

// GetQueue retrieves a Queue by OCID.
func (c *OciQueueServiceManager) GetQueue(ctx context.Context, queueId ociv1beta1.OCID) (*ociqueue.Queue, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := ociqueue.GetQueueRequest{
		QueueId: common.String(string(queueId)),
	}

	resp, err := client.GetQueue(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Queue, nil
}

// GetQueueOcid looks up an existing Queue by display name and returns its OCID if found.
// Returns nil if no matching queue in CREATING, UPDATING, or ACTIVE state is found.
func (c *OciQueueServiceManager) GetQueueOcid(ctx context.Context, q ociv1beta1.OciQueue) (*ociv1beta1.OCID, error) {
	client, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	req := ociqueue.ListQueuesRequest{
		CompartmentId: common.String(string(q.Spec.CompartmentId)),
		DisplayName:   common.String(q.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	resp, err := client.ListQueues(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, "Error listing Queues")
		return nil, err
	}

	for _, item := range resp.Items {
		state := string(item.LifecycleState)
		if state == "ACTIVE" || state == "CREATING" || state == "UPDATING" {
			c.Log.DebugLog(fmt.Sprintf("OciQueue %s exists with OCID %s", q.Spec.DisplayName, *item.Id))
			return (*ociv1beta1.OCID)(item.Id), nil
		}
	}

	c.Log.DebugLog(fmt.Sprintf("OciQueue %s does not exist", q.Spec.DisplayName))
	return nil, nil
}

// UpdateQueue updates an existing Queue.
func (c *OciQueueServiceManager) UpdateQueue(ctx context.Context, q *ociv1beta1.OciQueue) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	targetID, err := servicemanager.ResolveResourceID(q.Status.OsokStatus.Ocid, q.Spec.QueueId)
	if err != nil {
		return err
	}

	existing, err := c.GetQueue(ctx, targetID)
	if err != nil {
		return err
	}
	if err := validateImmutableQueueUpdate(q, existing); err != nil {
		return err
	}

	if err := c.changeQueueCompartmentIfNeeded(ctx, client, targetID, q, existing); err != nil {
		return err
	}

	req, updateNeeded := buildQueueUpdateRequest(targetID, q, existing)
	if !updateNeeded {
		return nil
	}

	_, err = client.UpdateQueue(ctx, req)
	return err
}

func (c *OciQueueServiceManager) changeQueueCompartmentIfNeeded(ctx context.Context,
	client QueueAdminClientInterface, targetID ociv1beta1.OCID, q *ociv1beta1.OciQueue, existing *ociqueue.Queue) error {
	if q.Spec.CompartmentId == "" {
		return nil
	}
	if existing.CompartmentId != nil && *existing.CompartmentId == string(q.Spec.CompartmentId) {
		return nil
	}

	_, err := client.ChangeQueueCompartment(ctx, ociqueue.ChangeQueueCompartmentRequest{
		QueueId: common.String(string(targetID)),
		ChangeQueueCompartmentDetails: ociqueue.ChangeQueueCompartmentDetails{
			CompartmentId: common.String(string(q.Spec.CompartmentId)),
		},
	})
	return err
}

func validateImmutableQueueUpdate(q *ociv1beta1.OciQueue, existing *ociqueue.Queue) error {
	if q.Spec.RetentionInSeconds <= 0 {
		return nil
	}
	if existing.RetentionInSeconds != nil && *existing.RetentionInSeconds == q.Spec.RetentionInSeconds {
		return nil
	}

	currentRetention := "unset"
	if existing.RetentionInSeconds != nil {
		currentRetention = fmt.Sprintf("%d", *existing.RetentionInSeconds)
	}

	return fmt.Errorf("retentionInSeconds cannot be updated in place (desired=%d, current=%s)", q.Spec.RetentionInSeconds, currentRetention)
}

func buildQueueUpdateRequest(targetID ociv1beta1.OCID, q *ociv1beta1.OciQueue,
	existing *ociqueue.Queue) (ociqueue.UpdateQueueRequest, bool) {
	updateDetails := ociqueue.UpdateQueueDetails{}
	updateNeeded := applyQueueDisplayNameUpdate(&updateDetails, q, existing)
	updateNeeded = applyQueueVisibilityUpdate(&updateDetails, q, existing) || updateNeeded
	updateNeeded = applyQueueTimeoutUpdate(&updateDetails, q, existing) || updateNeeded
	updateNeeded = applyQueueDeadLetterCountUpdate(&updateDetails, q, existing) || updateNeeded
	updateNeeded = applyQueueCustomEncryptionKeyUpdate(&updateDetails, q, existing) || updateNeeded
	updateNeeded = applyQueueFreeformTagsUpdate(&updateDetails, q, existing) || updateNeeded
	updateNeeded = applyQueueDefinedTagsUpdate(&updateDetails, q, existing) || updateNeeded

	return ociqueue.UpdateQueueRequest{
		QueueId:            common.String(string(targetID)),
		UpdateQueueDetails: updateDetails,
	}, updateNeeded
}

func applyQueueDisplayNameUpdate(updateDetails *ociqueue.UpdateQueueDetails, q *ociv1beta1.OciQueue, existing *ociqueue.Queue) bool {
	if q.Spec.DisplayName == "" || (existing.DisplayName != nil && *existing.DisplayName == q.Spec.DisplayName) {
		return false
	}

	updateDetails.DisplayName = common.String(q.Spec.DisplayName)
	return true
}

func applyQueueVisibilityUpdate(updateDetails *ociqueue.UpdateQueueDetails, q *ociv1beta1.OciQueue, existing *ociqueue.Queue) bool {
	if q.Spec.VisibilityInSeconds <= 0 || (existing.VisibilityInSeconds != nil && *existing.VisibilityInSeconds == q.Spec.VisibilityInSeconds) {
		return false
	}

	updateDetails.VisibilityInSeconds = common.Int(q.Spec.VisibilityInSeconds)
	return true
}

func applyQueueTimeoutUpdate(updateDetails *ociqueue.UpdateQueueDetails, q *ociv1beta1.OciQueue, existing *ociqueue.Queue) bool {
	if q.Spec.TimeoutInSeconds <= 0 || (existing.TimeoutInSeconds != nil && *existing.TimeoutInSeconds == q.Spec.TimeoutInSeconds) {
		return false
	}

	updateDetails.TimeoutInSeconds = common.Int(q.Spec.TimeoutInSeconds)
	return true
}

func applyQueueDeadLetterCountUpdate(updateDetails *ociqueue.UpdateQueueDetails, q *ociv1beta1.OciQueue, existing *ociqueue.Queue) bool {
	if q.Spec.DeadLetterQueueDeliveryCount <= 0 ||
		(existing.DeadLetterQueueDeliveryCount != nil && *existing.DeadLetterQueueDeliveryCount == q.Spec.DeadLetterQueueDeliveryCount) {
		return false
	}

	updateDetails.DeadLetterQueueDeliveryCount = common.Int(q.Spec.DeadLetterQueueDeliveryCount)
	return true
}

func applyQueueCustomEncryptionKeyUpdate(updateDetails *ociqueue.UpdateQueueDetails, q *ociv1beta1.OciQueue, existing *ociqueue.Queue) bool {
	desiredKey := string(q.Spec.CustomEncryptionKeyId)
	if desiredKey == "" {
		return false
	}
	if existing.CustomEncryptionKeyId != nil && *existing.CustomEncryptionKeyId == desiredKey {
		return false
	}

	updateDetails.CustomEncryptionKeyId = common.String(desiredKey)
	return true
}

func applyQueueFreeformTagsUpdate(updateDetails *ociqueue.UpdateQueueDetails, q *ociv1beta1.OciQueue, existing *ociqueue.Queue) bool {
	if q.Spec.FreeFormTags == nil || reflect.DeepEqual(existing.FreeformTags, q.Spec.FreeFormTags) {
		return false
	}

	updateDetails.FreeformTags = q.Spec.FreeFormTags
	return true
}

func applyQueueDefinedTagsUpdate(updateDetails *ociqueue.UpdateQueueDetails, q *ociv1beta1.OciQueue, existing *ociqueue.Queue) bool {
	if q.Spec.DefinedTags == nil {
		return false
	}

	desiredDefinedTags := *util.ConvertToOciDefinedTags(&q.Spec.DefinedTags)
	if reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
		return false
	}

	updateDetails.DefinedTags = desiredDefinedTags
	return true
}

// DeleteQueue deletes the Queue for the given OCID.
func (c *OciQueueServiceManager) DeleteQueue(ctx context.Context, queueId ociv1beta1.OCID) error {
	client, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := ociqueue.DeleteQueueRequest{
		QueueId: common.String(string(queueId)),
	}

	_, err = client.DeleteQueue(ctx, req)
	return err
}
