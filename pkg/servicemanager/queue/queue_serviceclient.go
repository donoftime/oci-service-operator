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
	"github.com/oracle/oci-service-operator/pkg/util"
)

func getQueueAdminClient(provider common.ConfigurationProvider) (ociqueue.QueueAdminClient, error) {
	return ociqueue.NewQueueAdminClientWithConfigurationProvider(provider)
}

// CreateQueue calls the OCI API to create a new Queue and returns the work request ID.
func (c *OciQueueServiceManager) CreateQueue(ctx context.Context, q ociv1beta1.OciQueue) (string, error) {
	client, err := getQueueAdminClient(c.Provider)
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
	client, err := getQueueAdminClient(c.Provider)
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
	client, err := getQueueAdminClient(c.Provider)
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
	client, err := getQueueAdminClient(c.Provider)
	if err != nil {
		return err
	}

	existing, err := c.GetQueue(ctx, q.Status.OsokStatus.Ocid)
	if err != nil {
		return err
	}

	updateDetails := ociqueue.UpdateQueueDetails{}
	updateNeeded := false

	if q.Spec.DisplayName != "" && (existing.DisplayName == nil || *existing.DisplayName != q.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(q.Spec.DisplayName)
		updateNeeded = true
	}
	if q.Spec.VisibilityInSeconds > 0 && (existing.VisibilityInSeconds == nil || *existing.VisibilityInSeconds != q.Spec.VisibilityInSeconds) {
		updateDetails.VisibilityInSeconds = common.Int(q.Spec.VisibilityInSeconds)
		updateNeeded = true
	}
	if q.Spec.TimeoutInSeconds > 0 && (existing.TimeoutInSeconds == nil || *existing.TimeoutInSeconds != q.Spec.TimeoutInSeconds) {
		updateDetails.TimeoutInSeconds = common.Int(q.Spec.TimeoutInSeconds)
		updateNeeded = true
	}
	if q.Spec.DeadLetterQueueDeliveryCount > 0 && (existing.DeadLetterQueueDeliveryCount == nil || *existing.DeadLetterQueueDeliveryCount != q.Spec.DeadLetterQueueDeliveryCount) {
		updateDetails.DeadLetterQueueDeliveryCount = common.Int(q.Spec.DeadLetterQueueDeliveryCount)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := ociqueue.UpdateQueueRequest{
		QueueId:            common.String(string(q.Status.OsokStatus.Ocid)),
		UpdateQueueDetails: updateDetails,
	}

	_, err = client.UpdateQueue(ctx, req)
	return err
}

// DeleteQueue deletes the Queue for the given OCID.
func (c *OciQueueServiceManager) DeleteQueue(ctx context.Context, queueId ociv1beta1.OCID) error {
	client, err := getQueueAdminClient(c.Provider)
	if err != nil {
		return err
	}

	req := ociqueue.DeleteQueueRequest{
		QueueId: common.String(string(queueId)),
	}

	_, err = client.DeleteQueue(ctx, req)
	return err
}
