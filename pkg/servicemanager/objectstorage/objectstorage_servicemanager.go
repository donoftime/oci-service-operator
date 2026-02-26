/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package objectstorage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociobjectstorage "github.com/oracle/oci-go-sdk/v65/objectstorage"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Compile-time check that ObjectStorageBucketServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &ObjectStorageBucketServiceManager{}

// ObjectStorageBucketServiceManager implements OSOKServiceManager for OCI Object Storage.
type ObjectStorageBucketServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	ociClient        ObjectStorageClientInterface
}

// NewObjectStorageBucketServiceManager creates a new ObjectStorageBucketServiceManager.
func NewObjectStorageBucketServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *ObjectStorageBucketServiceManager {
	return &ObjectStorageBucketServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the ObjectStorageBucket resource against OCI.
//
// Unlike most OCI resources, Object Storage buckets are NOT identified by OCIDs.
// They are identified by namespace + bucketName. The status.ocid field stores
// "namespace/bucketName" as a composite identifier.
//
// Creation logic:
//  1. If spec.id is set, it contains "namespace/bucketName" — bind to existing bucket.
//  2. If status.ocid is set, the bucket was previously created — verify and optionally update.
//  3. Otherwise, resolve namespace and create the bucket.
func (m *ObjectStorageBucketServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	resource, err := m.convert(obj)
	if err != nil {
		m.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	// Determine namespace and bucketName to use.
	var ns, bucketName string

	if strings.TrimSpace(string(resource.Spec.BucketId)) != "" {
		// Bind to existing bucket: spec.id = "namespace/bucketName"
		parts := strings.SplitN(string(resource.Spec.BucketId), "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			err = fmt.Errorf("spec.id must be in format 'namespace/bucketName', got: %s", resource.Spec.BucketId)
			m.Log.ErrorLog(err, "Invalid spec.id for ObjectStorageBucket")
			resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), m.Log)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		ns, bucketName = parts[0], parts[1]

		// Verify the bucket exists.
		if err = m.getBucket(ctx, ns, bucketName); err != nil {
			m.Log.ErrorLog(err, "Error getting existing ObjectStorageBucket")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		compositeId := ns + "/" + bucketName
		resource.Status.OsokStatus.Ocid = ociv1beta1.OCID(compositeId)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "ObjectStorageBucket Bound", m.Log)
		m.Log.InfoLog(fmt.Sprintf("ObjectStorageBucket %s bound", compositeId))

	} else if strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" {
		// Bucket was previously created — verify it still exists and apply updates.
		compositeId := string(resource.Status.OsokStatus.Ocid)
		parts := strings.SplitN(compositeId, "/", 2)
		if len(parts) != 2 {
			err = fmt.Errorf("status.ocid is malformed: %s", compositeId)
			m.Log.ErrorLog(err, "Malformed status.ocid")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		ns, bucketName = parts[0], parts[1]

		if err = m.getBucket(ctx, ns, bucketName); err != nil {
			m.Log.ErrorLog(err, "Error verifying existing ObjectStorageBucket")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		// Apply updates (accessType, versioning) if spec has changed.
		if err = m.updateBucket(ctx, ns, bucketName, resource); err != nil {
			m.Log.ErrorLog(err, "Error updating ObjectStorageBucket")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "ObjectStorageBucket Active", m.Log)
		m.Log.InfoLog(fmt.Sprintf("ObjectStorageBucket %s is active", compositeId))

	} else {
		// Create a new bucket.
		ns, err = m.resolveNamespace(ctx, resource)
		if err != nil {
			m.Log.ErrorLog(err, "Error resolving Object Storage namespace")
			resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), m.Log)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
		bucketName = resource.Spec.Name

		if err = m.createBucket(ctx, ns, resource); err != nil {
			m.Log.ErrorLog(err, "Create ObjectStorageBucket failed")
			resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus,
				ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), m.Log)
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		compositeId := ns + "/" + bucketName
		resource.Status.OsokStatus.Ocid = ociv1beta1.OCID(compositeId)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "ObjectStorageBucket Created", m.Log)
		m.Log.InfoLog(fmt.Sprintf("ObjectStorageBucket %s created", compositeId))
	}

	if resource.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		resource.Status.OsokStatus.CreatedAt = &now
	}

	_, err = m.addToSecret(ctx, resource.Namespace, resource.Name, ns, bucketName)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		}
		m.Log.InfoLog("Secret creation failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// Delete handles deletion of the ObjectStorageBucket (called by the finalizer).
func (m *ObjectStorageBucketServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	resource, err := m.convert(obj)
	if err != nil {
		return false, err
	}

	if strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) == "" {
		m.Log.InfoLog("ObjectStorageBucket has no composite ID, nothing to delete")
		return true, nil
	}

	compositeId := string(resource.Status.OsokStatus.Ocid)
	parts := strings.SplitN(compositeId, "/", 2)
	if len(parts) != 2 {
		m.Log.InfoLog(fmt.Sprintf("ObjectStorageBucket status.ocid malformed: %s, skipping OCI delete", compositeId))
		return true, nil
	}
	ns, bucketName := parts[0], parts[1]

	m.Log.InfoLog(fmt.Sprintf("Deleting ObjectStorageBucket %s/%s", ns, bucketName))

	client, err := m.getOCIClient()
	if err != nil {
		return false, err
	}

	req := ociobjectstorage.DeleteBucketRequest{
		NamespaceName: common.String(ns),
		BucketName:    common.String(bucketName),
	}
	_, err = client.DeleteBucket(ctx, req)
	if err != nil {
		// 404 means already deleted — treat as success.
		if isNotFound(err) {
			m.Log.InfoLog(fmt.Sprintf("ObjectStorageBucket %s/%s already deleted", ns, bucketName))
		} else {
			m.Log.ErrorLog(err, "Error while deleting ObjectStorageBucket")
			return false, err
		}
	}

	if _, err := m.CredentialClient.DeleteSecret(ctx, resource.Name, resource.Namespace); err != nil {
		m.Log.ErrorLog(err, "Error while deleting ObjectStorageBucket secret")
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (m *ObjectStorageBucketServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := m.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

// resolveNamespace returns spec.namespace if set, otherwise calls GetNamespace to retrieve
// the tenancy Object Storage namespace and caches it in spec.
func (m *ObjectStorageBucketServiceManager) resolveNamespace(ctx context.Context, resource *ociv1beta1.ObjectStorageBucket) (string, error) {
	if resource.Spec.Namespace != "" {
		return resource.Spec.Namespace, nil
	}

	client, err := m.getOCIClient()
	if err != nil {
		return "", err
	}

	req := ociobjectstorage.GetNamespaceRequest{
		CompartmentId: common.String(string(resource.Spec.CompartmentId)),
	}
	resp, err := client.GetNamespace(ctx, req)
	if err != nil {
		return "", fmt.Errorf("GetNamespace failed: %w", err)
	}
	if resp.Value == nil {
		return "", fmt.Errorf("GetNamespace returned nil namespace")
	}

	// Cache in spec so subsequent reconciles don't need to call GetNamespace again.
	resource.Spec.Namespace = *resp.Value
	return *resp.Value, nil
}

// createBucket calls the OCI API to create a new bucket.
func (m *ObjectStorageBucketServiceManager) createBucket(ctx context.Context, ns string, resource *ociv1beta1.ObjectStorageBucket) error {
	client, err := m.getOCIClient()
	if err != nil {
		return err
	}

	details := ociobjectstorage.CreateBucketDetails{
		Name:          common.String(resource.Spec.Name),
		CompartmentId: common.String(string(resource.Spec.CompartmentId)),
	}

	if resource.Spec.AccessType != "" {
		details.PublicAccessType = ociobjectstorage.CreateBucketDetailsPublicAccessTypeEnum(resource.Spec.AccessType)
	}
	if resource.Spec.StorageType != "" {
		details.StorageTier = ociobjectstorage.CreateBucketDetailsStorageTierEnum(resource.Spec.StorageType)
	}
	if resource.Spec.Versioning != "" {
		details.Versioning = ociobjectstorage.CreateBucketDetailsVersioningEnum(resource.Spec.Versioning)
	}
	if resource.Spec.FreeFormTags != nil {
		details.FreeformTags = resource.Spec.FreeFormTags
	}
	if resource.Spec.DefinedTags != nil {
		details.DefinedTags = *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
	}

	req := ociobjectstorage.CreateBucketRequest{
		NamespaceName:       common.String(ns),
		CreateBucketDetails: details,
	}

	_, err = client.CreateBucket(ctx, req)
	return err
}

// getBucket verifies a bucket exists by calling GetBucket.
func (m *ObjectStorageBucketServiceManager) getBucket(ctx context.Context, ns, bucketName string) error {
	client, err := m.getOCIClient()
	if err != nil {
		return err
	}

	req := ociobjectstorage.GetBucketRequest{
		NamespaceName: common.String(ns),
		BucketName:    common.String(bucketName),
	}
	_, err = client.GetBucket(ctx, req)
	return err
}

// updateBucket applies spec changes (accessType, versioning) to an existing bucket.
func (m *ObjectStorageBucketServiceManager) updateBucket(ctx context.Context, ns, bucketName string, resource *ociv1beta1.ObjectStorageBucket) error {
	client, err := m.getOCIClient()
	if err != nil {
		return err
	}

	updateDetails := ociobjectstorage.UpdateBucketDetails{}
	updateNeeded := false

	if resource.Spec.AccessType != "" {
		updateDetails.PublicAccessType = ociobjectstorage.UpdateBucketDetailsPublicAccessTypeEnum(resource.Spec.AccessType)
		updateNeeded = true
	}
	if resource.Spec.Versioning != "" {
		updateDetails.Versioning = ociobjectstorage.UpdateBucketDetailsVersioningEnum(resource.Spec.Versioning)
		updateNeeded = true
	}

	if !updateNeeded {
		return nil
	}

	req := ociobjectstorage.UpdateBucketRequest{
		NamespaceName:       common.String(ns),
		BucketName:          common.String(bucketName),
		UpdateBucketDetails: updateDetails,
	}
	_, err = client.UpdateBucket(ctx, req)
	return err
}

func (m *ObjectStorageBucketServiceManager) convert(obj runtime.Object) (*ociv1beta1.ObjectStorageBucket, error) {
	resource, ok := obj.(*ociv1beta1.ObjectStorageBucket)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for ObjectStorageBucket")
	}
	return resource, nil
}

// isNotFound checks whether an OCI error is a 404 Not Found.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	serviceErr, ok := common.IsServiceError(err)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}
