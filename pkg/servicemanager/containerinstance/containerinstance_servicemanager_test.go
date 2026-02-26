/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocicontainerinstances "github.com/oracle/oci-go-sdk/v65/containerinstances"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/containerinstance"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct {
	createCalled bool
	deleteCalled bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	return true, nil
}

// fakeOciClient implements ContainerInstanceClientInterface for testing.
type fakeOciClient struct {
	createFn      func(ctx context.Context, req ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error)
	getFn         func(ctx context.Context, req ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error)
	listFn        func(ctx context.Context, req ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error)
	updateFn      func(ctx context.Context, req ocicontainerinstances.UpdateContainerInstanceRequest) (ocicontainerinstances.UpdateContainerInstanceResponse, error)
	deleteFn      func(ctx context.Context, req ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error)
	createCalled  bool
	deleteCalled  bool
	createRequest *ocicontainerinstances.CreateContainerInstanceRequest
}

func (f *fakeOciClient) CreateContainerInstance(ctx context.Context, req ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
	f.createCalled = true
	f.createRequest = &req
	if f.createFn != nil {
		return f.createFn(ctx, req)
	}
	id := "ocid1.containerinstance.oc1..new"
	return ocicontainerinstances.CreateContainerInstanceResponse{
		ContainerInstance: ocicontainerinstances.ContainerInstance{
			Id:             common.String(id),
			LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
		},
	}, nil
}

func (f *fakeOciClient) GetContainerInstance(ctx context.Context, req ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	id := *req.ContainerInstanceId
	name := "test-ci"
	return ocicontainerinstances.GetContainerInstanceResponse{
		ContainerInstance: ocicontainerinstances.ContainerInstance{
			Id:             common.String(id),
			DisplayName:    common.String(name),
			LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
		},
	}, nil
}

func (f *fakeOciClient) ListContainerInstances(ctx context.Context, req ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return ocicontainerinstances.ListContainerInstancesResponse{
		ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
			Items: []ocicontainerinstances.ContainerInstanceSummary{},
		},
	}, nil
}

func (f *fakeOciClient) UpdateContainerInstance(ctx context.Context, req ocicontainerinstances.UpdateContainerInstanceRequest) (ocicontainerinstances.UpdateContainerInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return ocicontainerinstances.UpdateContainerInstanceResponse{}, nil
}

func (f *fakeOciClient) DeleteContainerInstance(ctx context.Context, req ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error) {
	f.deleteCalled = true
	if f.deleteFn != nil {
		return f.deleteFn(ctx, req)
	}
	return ocicontainerinstances.DeleteContainerInstanceResponse{}, nil
}

// newTestManager creates a manager with a fake OCI client injected.
func newTestManager(ociClient *fakeOciClient) *ContainerInstanceServiceManager {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewContainerInstanceServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)
	ExportSetClientForTest(mgr, ociClient)
	return mgr
}

// makeContainerInstanceSpec creates a minimal ContainerInstance spec for tests.
func makeContainerInstanceSpec(displayName string) *ociv1beta1.ContainerInstance {
	ci := &ociv1beta1.ContainerInstance{}
	ci.Name = "test-ci"
	ci.Namespace = "default"
	ci.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	ci.Spec.AvailabilityDomain = "AD-1"
	ci.Spec.Shape = "CI.Standard.E4.Flex"
	ci.Spec.ShapeConfig = ociv1beta1.ContainerInstanceShapeConfig{Ocpus: 1, MemoryInGBs: 8}
	ci.Spec.Containers = []ociv1beta1.ContainerDetails{
		{ImageUrl: "busybox:latest"},
	}
	ci.Spec.Vnics = []ociv1beta1.ContainerVnicDetails{
		{SubnetId: "ocid1.subnet.oc1..xxx"},
	}
	if displayName != "" {
		ci.Spec.DisplayName = common.String(displayName)
	}
	return ci
}

// --- Existing tests preserved ---

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewContainerInstanceServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	ci := &ociv1beta1.ContainerInstance{}
	ci.Name = "test-ci"
	ci.Namespace = "default"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from a ContainerInstance object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewContainerInstanceServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	ci := &ociv1beta1.ContainerInstance{}
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.xxx"

	status, err := mgr.GetCrdStatus(ci)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.containerinstance.xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewContainerInstanceServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-ContainerInstance objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewContainerInstanceServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// --- New mock-based tests ---

// TestCreateOrUpdate_CreatePath verifies that a new container instance is created when
// no ContainerInstanceId is set and no existing instance is found.
func TestCreateOrUpdate_CreatePath(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
			return ocicontainerinstances.CreateContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String("ocid1.containerinstance.oc1..created"),
					DisplayName:    common.String("test-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, ociClient.createCalled)
	assert.Equal(t, ociv1beta1.OCID("ocid1.containerinstance.oc1..created"), ci.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_ListError verifies CreateOrUpdate returns an error when listing fails.
func TestCreateOrUpdate_ListError(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{}, errors.New("list failed")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreateError verifies CreateOrUpdate handles creation errors.
func TestCreateOrUpdate_CreateError(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
			return ocicontainerinstances.CreateContainerInstanceResponse{}, errors.New("create failed")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_ExistingInstanceFound verifies the bind path when an existing
// container instance is found by display name.
func TestCreateOrUpdate_ExistingInstanceFound(t *testing.T) {
	existingOcid := "ocid1.containerinstance.oc1..existing"
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{
						{
							Id:             common.String(existingOcid),
							LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
						},
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, req ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String(existingOcid),
					DisplayName:    common.String("test-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, ociClient.createCalled, "create should not be called when instance already exists")
	assert.Equal(t, ociv1beta1.OCID(existingOcid), ci.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_FailedLifecycleState verifies CreateOrUpdate marks status as failed
// when the container instance reports a FAILED lifecycle state after creation.
func TestCreateOrUpdate_FailedLifecycleState(t *testing.T) {
	failedOcid := "ocid1.containerinstance.oc1..failed"
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ ocicontainerinstances.CreateContainerInstanceRequest) (ocicontainerinstances.CreateContainerInstanceResponse, error) {
			return ocicontainerinstances.CreateContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String(failedOcid),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateFailed,
				},
			}, nil
		},
		// GetContainerInstance is called after create (with retry policy); return FAILED state.
		getFn: func(_ context.Context, _ ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String(failedOcid),
					DisplayName:    common.String("test-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateFailed,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "should be unsuccessful when instance is in FAILED state")
}

// TestCreateOrUpdate_WithContainerInstanceId verifies that when a ContainerInstanceId is
// set, the manager binds to the existing instance and calls update.
func TestCreateOrUpdate_WithContainerInstanceId(t *testing.T) {
	existingOcid := "ocid1.containerinstance.oc1..bound"
	ociClient := &fakeOciClient{
		getFn: func(_ context.Context, req ocicontainerinstances.GetContainerInstanceRequest) (ocicontainerinstances.GetContainerInstanceResponse, error) {
			return ocicontainerinstances.GetContainerInstanceResponse{
				ContainerInstance: ocicontainerinstances.ContainerInstance{
					Id:             common.String(existingOcid),
					DisplayName:    common.String("test-ci"),
					LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")
	ci.Spec.ContainerInstanceId = ociv1beta1.OCID(existingOcid)

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, ociClient.createCalled, "create should not be called when ContainerInstanceId is set")
	assert.Equal(t, ociv1beta1.OCID(existingOcid), ci.Status.OsokStatus.Ocid)
}

// TestDelete_WithOcid verifies that deletion calls the OCI delete API when OCID is set.
func TestDelete_WithOcid(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)

	ci := &ociv1beta1.ContainerInstance{}
	ci.Name = "test-ci"
	ci.Namespace = "default"
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, ociClient.deleteCalled)
}

// TestDelete_Error verifies Delete propagates errors from the OCI API.
func TestDelete_Error(t *testing.T) {
	ociClient := &fakeOciClient{
		deleteFn: func(_ context.Context, _ ocicontainerinstances.DeleteContainerInstanceRequest) (ocicontainerinstances.DeleteContainerInstanceResponse, error) {
			return ocicontainerinstances.DeleteContainerInstanceResponse{}, errors.New("delete failed")
		},
	}
	mgr := newTestManager(ociClient)

	ci := &ociv1beta1.ContainerInstance{}
	ci.Name = "test-ci"
	ci.Status.OsokStatus.Ocid = "ocid1.containerinstance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.Error(t, err)
	assert.False(t, done)
}

// TestGetContainerInstanceOcid_NilDisplayName verifies that a nil display name returns nil.
func TestGetContainerInstanceOcid_NilDisplayName(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)

	ci := makeContainerInstanceSpec("") // no display name
	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}

// TestGetContainerInstanceOcid_NotFound verifies nil is returned when no matching instance exists.
func TestGetContainerInstanceOcid_NotFound(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{},
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("nonexistent")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}

// TestGetContainerInstanceOcid_FoundActive verifies an ACTIVE instance is found and returned.
func TestGetContainerInstanceOcid_FoundActive(t *testing.T) {
	foundOcid := "ocid1.containerinstance.oc1..active"
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{
						{
							Id:             common.String(foundOcid),
							LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
						},
					},
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("my-instance")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID(foundOcid), *ocid)
}

// TestGetContainerInstanceOcid_FoundCreating verifies that a CREATING state instance is
// treated as existing (returned by GetContainerInstanceOcid).
func TestGetContainerInstanceOcid_FoundCreating(t *testing.T) {
	foundOcid := "ocid1.containerinstance.oc1..creating"
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{
				ContainerInstanceCollection: ocicontainerinstances.ContainerInstanceCollection{
					Items: []ocicontainerinstances.ContainerInstanceSummary{
						{
							Id:             common.String(foundOcid),
							LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateCreating,
						},
					},
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("creating-instance")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.NoError(t, err)
	assert.NotNil(t, ocid, "CREATING state instances should be treated as existing")
	assert.Equal(t, ociv1beta1.OCID(foundOcid), *ocid)
}

// TestGetContainerInstanceOcid_ListError verifies errors from listing are propagated.
func TestGetContainerInstanceOcid_ListError(t *testing.T) {
	ociClient := &fakeOciClient{
		listFn: func(_ context.Context, _ ocicontainerinstances.ListContainerInstancesRequest) (ocicontainerinstances.ListContainerInstancesResponse, error) {
			return ocicontainerinstances.ListContainerInstancesResponse{}, errors.New("network error")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeContainerInstanceSpec("test-ci")

	ocid, err := mgr.GetContainerInstanceOcid(context.Background(), *ci)
	assert.Error(t, err)
	assert.Nil(t, ocid)
}

// TestCreateContainerInstance_WithVolumeMounts verifies that volume mount configuration
// in the spec is correctly mapped to the OCI create request.
func TestCreateContainerInstance_WithVolumeMounts(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)

	subPath := "data"
	isReadOnly := true
	ci := makeContainerInstanceSpec("test-ci")
	ci.Spec.Containers[0].VolumeMounts = []ociv1beta1.ContainerVolumeMount{
		{
			MountPath:  "/data",
			VolumeName: "my-volume",
			SubPath:    &subPath,
			IsReadOnly: &isReadOnly,
		},
	}

	_, err := mgr.CreateContainerInstance(context.Background(), *ci)
	assert.NoError(t, err)
	assert.True(t, ociClient.createCalled)

	req := ociClient.createRequest
	assert.NotNil(t, req)
	assert.Len(t, req.CreateContainerInstanceDetails.Containers, 1)
	assert.Len(t, req.CreateContainerInstanceDetails.Containers[0].VolumeMounts, 1)
	vm := req.CreateContainerInstanceDetails.Containers[0].VolumeMounts[0]
	assert.Equal(t, "/data", *vm.MountPath)
	assert.Equal(t, "my-volume", *vm.VolumeName)
	assert.Equal(t, "data", *vm.SubPath)
	assert.Equal(t, true, *vm.IsReadOnly)
}

// TestCreateContainerInstance_WithImagePullSecrets verifies that image pull secret
// configuration in the spec is correctly mapped to the OCI create request.
func TestCreateContainerInstance_WithImagePullSecrets(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)

	ci := makeContainerInstanceSpec("test-ci")
	ci.Spec.ImagePullSecrets = []ociv1beta1.ContainerImagePullSecret{
		{
			RegistryEndpoint: "registry.example.com",
			Username:         "myuser",
			Password:         "mypassword",
		},
	}

	_, err := mgr.CreateContainerInstance(context.Background(), *ci)
	assert.NoError(t, err)
	assert.True(t, ociClient.createCalled)

	req := ociClient.createRequest
	assert.NotNil(t, req)
	assert.Len(t, req.CreateContainerInstanceDetails.ImagePullSecrets, 1)
	secret, ok := req.CreateContainerInstanceDetails.ImagePullSecrets[0].(ocicontainerinstances.CreateBasicImagePullSecretDetails)
	assert.True(t, ok, "secret should be CreateBasicImagePullSecretDetails")
	assert.Equal(t, "registry.example.com", *secret.RegistryEndpoint)
	assert.Equal(t, "myuser", *secret.Username)
	assert.Equal(t, "mypassword", *secret.Password)
}

// TestGetRetryPolicy_CreatingState verifies the retry policy retries when the container
// instance is in CREATING state.
func TestGetRetryPolicy_CreatingState(t *testing.T) {
	mgr := newTestManager(&fakeOciClient{})
	policy := GetRetryPolicyForTest(mgr, 5)

	// Build a fake response with CREATING state
	response := common.OCIOperationResponse{
		Response: ocicontainerinstances.GetContainerInstanceResponse{
			ContainerInstance: ocicontainerinstances.ContainerInstance{
				LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateCreating,
			},
		},
	}

	assert.True(t, policy.ShouldRetryOperation(response), "should retry when state is CREATING")
}

// TestGetRetryPolicy_ActiveState verifies the retry policy does not retry when the
// container instance is in ACTIVE state.
func TestGetRetryPolicy_ActiveState(t *testing.T) {
	mgr := newTestManager(&fakeOciClient{})
	policy := GetRetryPolicyForTest(mgr, 5)

	response := common.OCIOperationResponse{
		Response: ocicontainerinstances.GetContainerInstanceResponse{
			ContainerInstance: ocicontainerinstances.ContainerInstance{
				LifecycleState: ocicontainerinstances.ContainerInstanceLifecycleStateActive,
			},
		},
	}

	assert.False(t, policy.ShouldRetryOperation(response), "should not retry when state is ACTIVE")
}

// TestCreateOrUpdate_NoDisplayNameCreatesWithoutList verifies that when no display name
// is set, the list call is skipped and a new instance is created directly.
func TestCreateOrUpdate_NoDisplayNameCreatesWithoutList(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)

	// No display name set
	ci := makeContainerInstanceSpec("")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, ociClient.createCalled)
}

// TestCreateContainerInstance_ContainerList verifies multiple containers are mapped correctly.
func TestCreateContainerInstance_ContainerList(t *testing.T) {
	ociClient := &fakeOciClient{}
	mgr := newTestManager(ociClient)

	ci := makeContainerInstanceSpec("multi-container")
	ci.Spec.Containers = []ociv1beta1.ContainerDetails{
		{ImageUrl: "nginx:latest", DisplayName: common.String("web")},
		{ImageUrl: "redis:7", DisplayName: common.String("cache")},
	}

	_, err := mgr.CreateContainerInstance(context.Background(), *ci)
	assert.NoError(t, err)

	req := ociClient.createRequest
	assert.NotNil(t, req)
	assert.Len(t, req.CreateContainerInstanceDetails.Containers, 2)
	assert.Equal(t, "nginx:latest", *req.CreateContainerInstanceDetails.Containers[0].ImageUrl)
	assert.Equal(t, "web", *req.CreateContainerInstanceDetails.Containers[0].DisplayName)
	assert.Equal(t, "redis:7", *req.CreateContainerInstanceDetails.Containers[1].ImageUrl)
	assert.Equal(t, "cache", *req.CreateContainerInstanceDetails.Containers[1].DisplayName)
}
