/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/streaming"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/util"
	"github.com/pkg/errors"
)

// StreamAdminClientInterface defines the OCI operations used by StreamServiceManager.
type StreamAdminClientInterface interface {
	CreateStream(ctx context.Context, request streaming.CreateStreamRequest) (streaming.CreateStreamResponse, error)
	GetStream(ctx context.Context, request streaming.GetStreamRequest) (streaming.GetStreamResponse, error)
	ListStreams(ctx context.Context, request streaming.ListStreamsRequest) (streaming.ListStreamsResponse, error)
	ChangeStreamCompartment(ctx context.Context, request streaming.ChangeStreamCompartmentRequest) (streaming.ChangeStreamCompartmentResponse, error)
	UpdateStream(ctx context.Context, request streaming.UpdateStreamRequest) (streaming.UpdateStreamResponse, error)
	DeleteStream(ctx context.Context, request streaming.DeleteStreamRequest) (streaming.DeleteStreamResponse, error)
}

func getStreamClient(provider common.ConfigurationProvider) (streaming.StreamAdminClient, error) {
	return streaming.NewStreamAdminClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *StreamServiceManager) getOCIClient() (StreamAdminClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getStreamClient(c.Provider)
}

func (c *StreamServiceManager) CreateStream(ctx context.Context, stream ociv1beta1.Stream) (streaming.CreateStreamResponse, error) {
	streamClient, err := c.getOCIClient()
	if err != nil {
		return streaming.CreateStreamResponse{}, err
	}
	c.Log.DebugLog("Creating Stream ", "name", stream.Spec.Name)

	createStreamDetails := streaming.CreateStreamDetails{
		Name:       common.String(stream.Spec.Name),
		Partitions: common.Int(stream.Spec.Partitions),
	}

	if stream.Spec.StreamPoolId != "" {
		createStreamDetails.StreamPoolId = common.String(string(stream.Spec.StreamPoolId))
	}

	if stream.Spec.CompartmentId != "" {
		createStreamDetails.CompartmentId = common.String(string(stream.Spec.CompartmentId))
	}

	if stream.Spec.RetentionInHours > 0 {
		createStreamDetails.RetentionInHours = common.Int(stream.Spec.RetentionInHours)
	}

	createStreamRequest := streaming.CreateStreamRequest{
		CreateStreamDetails: createStreamDetails,
	}

	return streamClient.CreateStream(ctx, createStreamRequest)
}

func (c *StreamServiceManager) GetStreamOcid(ctx context.Context, stream ociv1beta1.Stream) (*ociv1beta1.OCID, error) {
	streamClient, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}
	listStreamsRequest := streaming.ListStreamsRequest{
		Name: common.String(stream.Spec.Name),
	}

	if string(stream.Spec.StreamPoolId) != "" {
		listStreamsRequest.StreamPoolId = common.String(string(stream.Spec.StreamPoolId))
	}

	if string(stream.Spec.CompartmentId) != "" {
		listStreamsRequest.CompartmentId = common.String(string(stream.Spec.CompartmentId))
	}
	listStreamsResponse, err := streamClient.ListStreams(ctx, listStreamsRequest)
	if err != nil {
		c.Log.ErrorLog(err, "Error while listing Stream")
		return nil, err
	}

	return c.GetCreateOrUpdateStream(listStreamsResponse, stream)
}

func (c *StreamServiceManager) DeleteStream(ctx context.Context, stream ociv1beta1.Stream) (streaming.DeleteStreamResponse, error) {
	streamClient, err := c.getOCIClient()
	if err != nil {
		return streaming.DeleteStreamResponse{}, err
	}
	c.Log.InfoLog("Deleting Stream ", "name", stream.Spec.Name)

	deleteStreamRequest := streaming.DeleteStreamRequest{
		StreamId: common.String(string(stream.Spec.StreamId)),
	}

	return streamClient.DeleteStream(ctx, deleteStreamRequest)
}

func (c *StreamServiceManager) GetStream(ctx context.Context, streamId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*streaming.Stream, error) {
	streamClient, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	getStreamRequest := streaming.GetStreamRequest{
		StreamId: common.String(string(streamId)),
	}

	if retryPolicy != nil {
		getStreamRequest.RequestMetadata.RetryPolicy = retryPolicy
	}

	response, err := streamClient.GetStream(ctx, getStreamRequest)
	if err != nil {
		return nil, err
	}

	return &response.Stream, nil
}

func (c *StreamServiceManager) UpdateStream(ctx context.Context, stream *ociv1beta1.Stream) error {
	streamClient, err := c.getOCIClient()
	if err != nil {
		return err
	}
	streamID, err := resolveStreamUpdateID(stream)
	if err != nil {
		return err
	}
	existingStream, err := c.GetStream(ctx, streamID, nil)
	if err != nil {
		return err
	}
	if err := validateImmutableStreamUpdate(stream, existingStream); err != nil {
		return err
	}
	if stream.Spec.CompartmentId != "" &&
		(existingStream.CompartmentId == nil || *existingStream.CompartmentId != string(stream.Spec.CompartmentId)) {
		_, err = streamClient.ChangeStreamCompartment(ctx, streaming.ChangeStreamCompartmentRequest{
			StreamId: common.String(string(streamID)),
			ChangeStreamCompartmentDetails: streaming.ChangeStreamCompartmentDetails{
				CompartmentId: common.String(string(stream.Spec.CompartmentId)),
			},
		})
		if err != nil {
			return err
		}
	}
	updateStreamDetails, updateNeeded := buildStreamUpdateDetails(stream, existingStream)
	if !updateNeeded {
		return nil
	}
	updateRequest := streaming.UpdateStreamRequest{
		StreamId:            common.String(string(streamID)),
		UpdateStreamDetails: updateStreamDetails,
	}
	_, err = streamClient.UpdateStream(ctx, updateRequest)
	return err
}

func (c *StreamServiceManager) GetStreamOCID(ctx context.Context, stream ociv1beta1.Stream, status string) (*ociv1beta1.OCID, error) {

	if status == "CREATE" {
		listResponse, err := c.GetListOfStreams(ctx, stream)

		if err != nil {
			return nil, err
		}

		return c.GetCreateOrUpdateStream(listResponse, stream)
	} else {

		listResponse, err := c.GetListOfStreams(ctx, stream)

		if err != nil {
			return nil, err
		}

		return c.GetFailedOrDeleteStream(listResponse, stream)
	}
}

func (c *StreamServiceManager) GetListOfStreams(ctx context.Context, stream ociv1beta1.Stream) (streaming.ListStreamsResponse, error) {
	streamClient, err := c.getOCIClient()
	if err != nil {
		return streaming.ListStreamsResponse{}, err
	}
	listStreamsRequest := streaming.ListStreamsRequest{
		Name:  common.String(stream.Spec.Name),
		Limit: common.Int(1),
	}

	if string(stream.Spec.StreamPoolId) != "" {
		listStreamsRequest.StreamPoolId = common.String(string(stream.Spec.StreamPoolId))
	}

	if string(stream.Spec.CompartmentId) != "" {
		listStreamsRequest.CompartmentId = common.String(string(stream.Spec.CompartmentId))
	}
	listStreamsResponse, err := streamClient.ListStreams(ctx, listStreamsRequest)

	if err != nil {
		c.Log.ErrorLog(err, "Error while listing Stream")
		return listStreamsResponse, err
	}

	return listStreamsResponse, nil
}

func (c *StreamServiceManager) GetFailedOrDeleteStream(listStreamsResponse streaming.ListStreamsResponse, stream ociv1beta1.Stream) (*ociv1beta1.OCID, error) {

	if len(listStreamsResponse.Items) > 0 {
		status := listStreamsResponse.Items[0].LifecycleState
		if status == "DELETED" || status == "DELETING" || status == "FAILED" {

			c.Log.DebugLog(fmt.Sprintf("Stream %s exists in GetFailedOrDeletingStream", stream.Spec.Name))

			return (*ociv1beta1.OCID)(listStreamsResponse.Items[0].Id), nil
		}
	}
	c.Log.DebugLog(fmt.Sprintf("Stream %s does not exist.", stream.Spec.Name))
	return nil, nil
}

func resolveStreamUpdateID(stream *ociv1beta1.Stream) (ociv1beta1.OCID, error) {
	streamID := stream.Spec.StreamId
	if strings.TrimSpace(string(streamID)) == "" {
		streamID = stream.Status.OsokStatus.Ocid
	}
	if strings.TrimSpace(string(streamID)) == "" {
		return "", errors.New("stream id is required for update")
	}
	return streamID, nil
}

func validateImmutableStreamUpdate(stream *ociv1beta1.Stream, existingStream *streaming.Stream) error {
	if stream.Spec.Name != "" && existingStream.Name != nil && *existingStream.Name != stream.Spec.Name {
		return errors.New("name can't be updated")
	}
	if stream.Spec.Partitions > 0 && existingStream.Partitions != nil && stream.Spec.Partitions != *existingStream.Partitions {
		return errors.New("Partitions can't be updated")
	}
	if stream.Spec.RetentionInHours > 0 && existingStream.RetentionInHours != nil &&
		stream.Spec.RetentionInHours != *existingStream.RetentionInHours {
		return errors.New("RetentionsHours can't be updated")
	}
	return nil
}

func buildStreamUpdateDetails(stream *ociv1beta1.Stream, existingStream *streaming.Stream) (streaming.UpdateStreamDetails, bool) {
	updateStreamDetails := streaming.UpdateStreamDetails{}
	updateNeeded := false

	if stream.Spec.StreamPoolId != "" && string(stream.Spec.StreamPoolId) != *existingStream.StreamPoolId {
		updateStreamDetails.StreamPoolId = common.String(strings.TrimSpace(string(stream.Spec.StreamPoolId)))
		updateNeeded = true
	}
	if stream.Spec.FreeFormTags != nil && !reflect.DeepEqual(existingStream.FreeformTags, stream.Spec.FreeFormTags) {
		updateStreamDetails.FreeformTags = stream.Spec.FreeFormTags
		updateNeeded = true
	}
	if definedTags, ok := changedStreamDefinedTags(stream, existingStream); ok {
		updateStreamDetails.DefinedTags = definedTags
		updateNeeded = true
	}

	return updateStreamDetails, updateNeeded
}

func changedStreamDefinedTags(stream *ociv1beta1.Stream, existingStream *streaming.Stream) (map[string]map[string]interface{}, bool) {
	if stream.Spec.DefinedTags == nil {
		return nil, false
	}
	defTag := *util.ConvertToOciDefinedTags(&stream.Spec.DefinedTags)
	if reflect.DeepEqual(existingStream.DefinedTags, defTag) {
		return nil, false
	}
	return defTag, true
}

func (c *StreamServiceManager) GetCreateOrUpdateStream(listStreamsResponse streaming.ListStreamsResponse, stream ociv1beta1.Stream) (*ociv1beta1.OCID, error) {

	if len(listStreamsResponse.Items) > 0 {
		c.Log.DebugLog(fmt.Sprintf(
			"Number of streams with same name %d ",
			len(listStreamsResponse.Items),
		))
		for entry := 0; entry < len(listStreamsResponse.Items); entry++ {
			status := listStreamsResponse.Items[entry].LifecycleState
			if status == "ACTIVE" || status == "CREATING" || status == "UPDATING" {

				c.Log.DebugLog(fmt.Sprintf("Stream %s exists.", stream.Spec.Name))

				return (*ociv1beta1.OCID)(listStreamsResponse.Items[entry].Id), nil
			}
		}

	}
	c.Log.DebugLog(fmt.Sprintf("Stream %s does not exist.", stream.Spec.Name))
	return nil, nil
}
