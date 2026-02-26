/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package objectstorage

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociobjectstorage "github.com/oracle/oci-go-sdk/v65/objectstorage"
)

// ObjectStorageClientInterface defines the OCI operations used by ObjectStorageBucketServiceManager.
type ObjectStorageClientInterface interface {
	GetNamespace(ctx context.Context, request ociobjectstorage.GetNamespaceRequest) (ociobjectstorage.GetNamespaceResponse, error)
	CreateBucket(ctx context.Context, request ociobjectstorage.CreateBucketRequest) (ociobjectstorage.CreateBucketResponse, error)
	GetBucket(ctx context.Context, request ociobjectstorage.GetBucketRequest) (ociobjectstorage.GetBucketResponse, error)
	UpdateBucket(ctx context.Context, request ociobjectstorage.UpdateBucketRequest) (ociobjectstorage.UpdateBucketResponse, error)
	DeleteBucket(ctx context.Context, request ociobjectstorage.DeleteBucketRequest) (ociobjectstorage.DeleteBucketResponse, error)
}

func getObjectStorageClient(provider common.ConfigurationProvider) (ociobjectstorage.ObjectStorageClient, error) {
	return ociobjectstorage.NewObjectStorageClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (m *ObjectStorageBucketServiceManager) getOCIClient() (ObjectStorageClientInterface, error) {
	if m.ociClient != nil {
		return m.ociClient, nil
	}
	return getObjectStorageClient(m.Provider)
}
