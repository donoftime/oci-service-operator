/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compute_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/compute"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeComputeClient implements ComputeInstanceClientInterface for testing.
type fakeComputeClient struct {
	launchFn        func(ctx context.Context, req core.LaunchInstanceRequest) (core.LaunchInstanceResponse, error)
	getFn           func(ctx context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error)
	listFn          func(ctx context.Context, req core.ListInstancesRequest) (core.ListInstancesResponse, error)
	updateFn        func(ctx context.Context, req core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error)
	terminateFn     func(ctx context.Context, req core.TerminateInstanceRequest) (core.TerminateInstanceResponse, error)
	launchCalled    bool
	terminateCalled bool
	terminateOcid   string
}

func (f *fakeComputeClient) LaunchInstance(ctx context.Context, req core.LaunchInstanceRequest) (core.LaunchInstanceResponse, error) {
	f.launchCalled = true
	if f.launchFn != nil {
		return f.launchFn(ctx, req)
	}
	id := "ocid1.instance.oc1..launched"
	return core.LaunchInstanceResponse{
		Instance: core.Instance{
			Id:             common.String(id),
			DisplayName:    common.String("test-instance"),
			LifecycleState: core.InstanceLifecycleStateRunning,
		},
	}, nil
}

func (f *fakeComputeClient) GetInstance(ctx context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
	if f.getFn != nil {
		return f.getFn(ctx, req)
	}
	id := "ocid1.instance.oc1..launched"
	if req.InstanceId != nil && *req.InstanceId != "" {
		id = *req.InstanceId
	}
	return core.GetInstanceResponse{
		Instance: core.Instance{
			Id:             common.String(id),
			DisplayName:    common.String("test-instance"),
			LifecycleState: core.InstanceLifecycleStateRunning,
		},
	}, nil
}

func (f *fakeComputeClient) ListInstances(ctx context.Context, req core.ListInstancesRequest) (core.ListInstancesResponse, error) {
	if f.listFn != nil {
		return f.listFn(ctx, req)
	}
	return core.ListInstancesResponse{Items: []core.Instance{}}, nil
}

func (f *fakeComputeClient) UpdateInstance(ctx context.Context, req core.UpdateInstanceRequest) (core.UpdateInstanceResponse, error) {
	if f.updateFn != nil {
		return f.updateFn(ctx, req)
	}
	return core.UpdateInstanceResponse{}, nil
}

func (f *fakeComputeClient) TerminateInstance(ctx context.Context, req core.TerminateInstanceRequest) (core.TerminateInstanceResponse, error) {
	f.terminateCalled = true
	if req.InstanceId != nil {
		f.terminateOcid = *req.InstanceId
	}
	if f.terminateFn != nil {
		return f.terminateFn(ctx, req)
	}
	return core.TerminateInstanceResponse{}, nil
}

// newTestManager creates a ComputeInstanceServiceManager with a fake OCI client injected.
func newTestManager(ociClient *fakeComputeClient) *ComputeInstanceServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewComputeInstanceServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		nil, nil, log)
	ExportSetClientForTest(mgr, ociClient)
	return mgr
}

// makeComputeInstanceSpec creates a minimal ComputeInstance spec for tests.
func makeComputeInstanceSpec(displayName string) *ociv1beta1.ComputeInstance {
	ci := &ociv1beta1.ComputeInstance{}
	ci.Name = "test-instance"
	ci.Namespace = "default"
	ci.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	ci.Spec.AvailabilityDomain = "AD-1"
	ci.Spec.Shape = "VM.Standard.E4.Flex"
	ci.Spec.ImageId = "ocid1.image.oc1..xxx"
	ci.Spec.SubnetId = "ocid1.subnet.oc1..xxx"
	if displayName != "" {
		ci.Spec.DisplayName = common.String(displayName)
	}
	return ci
}

// TestCreateOrUpdate_CreatesNewInstance verifies that when GetInstanceOcid returns nil
// (no existing instance), a new instance is launched and status is set correctly.
func TestCreateOrUpdate_CreatesNewInstance(t *testing.T) {
	launchedId := "ocid1.instance.oc1..new"
	ociClient := &fakeComputeClient{
		listFn: func(_ context.Context, _ core.ListInstancesRequest) (core.ListInstancesResponse, error) {
			return core.ListInstancesResponse{Items: []core.Instance{}}, nil
		},
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             common.String(launchedId),
					DisplayName:    common.String("test-instance"),
					LifecycleState: core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("test-instance")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, ociClient.launchCalled, "LaunchInstance should be called when no existing instance found")
	assert.Equal(t, ociv1beta1.OCID(launchedId), ci.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_InstanceRunning verifies that an existing RUNNING instance is found
// by display name and status is set to Active with IsSuccessful=true.
func TestCreateOrUpdate_InstanceRunning(t *testing.T) {
	existingOcid := "ocid1.instance.oc1..running"
	ociClient := &fakeComputeClient{
		listFn: func(_ context.Context, _ core.ListInstancesRequest) (core.ListInstancesResponse, error) {
			return core.ListInstancesResponse{
				Items: []core.Instance{
					{
						Id:             common.String(existingOcid),
						DisplayName:    common.String("test-instance"),
						LifecycleState: core.InstanceLifecycleStateRunning,
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             common.String(existingOcid),
					DisplayName:    common.String("test-instance"),
					LifecycleState: core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("test-instance")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, ociClient.launchCalled, "LaunchInstance should not be called for existing instance")
	assert.Equal(t, ociv1beta1.OCID(existingOcid), ci.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_BindsById verifies that when Spec.ComputeInstanceId is set,
// GetInstance is called directly with that ID and no list or launch occurs.
func TestCreateOrUpdate_BindsById(t *testing.T) {
	existingOcid := "ocid1.instance.oc1..bound"
	ociClient := &fakeComputeClient{
		getFn: func(_ context.Context, req core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             common.String(existingOcid),
					DisplayName:    common.String("test-instance"),
					LifecycleState: core.InstanceLifecycleStateRunning,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("test-instance")
	ci.Spec.ComputeInstanceId = ociv1beta1.OCID(existingOcid)

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.False(t, ociClient.launchCalled, "LaunchInstance should not be called when ComputeInstanceId is set")
	assert.Equal(t, ociv1beta1.OCID(existingOcid), ci.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_InstanceFailed verifies that a TERMINATED instance causes
// IsSuccessful=false and Failed status to be set.
func TestCreateOrUpdate_InstanceFailed(t *testing.T) {
	terminatedOcid := "ocid1.instance.oc1..terminated"
	ociClient := &fakeComputeClient{
		listFn: func(_ context.Context, _ core.ListInstancesRequest) (core.ListInstancesResponse, error) {
			return core.ListInstancesResponse{Items: []core.Instance{}}, nil
		},
		getFn: func(_ context.Context, _ core.GetInstanceRequest) (core.GetInstanceResponse, error) {
			return core.GetInstanceResponse{
				Instance: core.Instance{
					Id:             common.String(terminatedOcid),
					DisplayName:    common.String("test-instance"),
					LifecycleState: core.InstanceLifecycleStateTerminated,
				},
			}, nil
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("test-instance")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful, "should be unsuccessful when instance is TERMINATED")
}

// TestDelete_CallsTerminate verifies that Delete calls TerminateInstance with the correct OCID.
func TestDelete_CallsTerminate(t *testing.T) {
	instanceOcid := "ocid1.instance.oc1..todel"
	ociClient := &fakeComputeClient{}
	mgr := newTestManager(ociClient)

	ci := &ociv1beta1.ComputeInstance{}
	ci.Name = "test-instance"
	ci.Namespace = "default"
	ci.Status.OsokStatus.Ocid = ociv1beta1.OCID(instanceOcid)

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, ociClient.terminateCalled, "TerminateInstance should be called")
	assert.Equal(t, instanceOcid, ociClient.terminateOcid, "TerminateInstance should use the correct OCID")
}

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	ociClient := &fakeComputeClient{}
	mgr := newTestManager(ociClient)

	ci := &ociv1beta1.ComputeInstance{}
	ci.Name = "test-instance"
	ci.Namespace = "default"

	done, err := mgr.Delete(context.Background(), ci)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, ociClient.terminateCalled, "TerminateInstance should not be called when no OCID")
}

// TestDelete_Error verifies Delete propagates errors from the OCI API.
func TestDelete_Error(t *testing.T) {
	ociClient := &fakeComputeClient{
		terminateFn: func(_ context.Context, _ core.TerminateInstanceRequest) (core.TerminateInstanceResponse, error) {
			return core.TerminateInstanceResponse{}, errors.New("terminate failed")
		},
	}
	mgr := newTestManager(ociClient)

	ci := &ociv1beta1.ComputeInstance{}
	ci.Name = "test-instance"
	ci.Status.OsokStatus.Ocid = "ocid1.instance.oc1..del"

	done, err := mgr.Delete(context.Background(), ci)
	assert.Error(t, err)
	assert.False(t, done)
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from a ComputeInstance object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := newTestManager(&fakeComputeClient{})

	ci := &ociv1beta1.ComputeInstance{}
	ci.Status.OsokStatus.Ocid = "ocid1.instance.oc1..xxx"

	status, err := mgr.GetCrdStatus(ci)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.instance.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := newTestManager(&fakeComputeClient{})

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-ComputeInstance objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := newTestManager(&fakeComputeClient{})

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestGetRetryPolicy_ProvisioningState verifies the retry policy retries when the
// instance is in PROVISIONING state.
func TestGetRetryPolicy_ProvisioningState(t *testing.T) {
	mgr := newTestManager(&fakeComputeClient{})
	policy := GetRetryPolicyForTest(mgr, 5)

	response := common.OCIOperationResponse{
		Response: core.GetInstanceResponse{
			Instance: core.Instance{
				LifecycleState: core.InstanceLifecycleStateProvisioning,
			},
		},
	}

	assert.True(t, policy.ShouldRetryOperation(response), "should retry when state is PROVISIONING")
}

// TestGetRetryPolicy_RunningState verifies the retry policy does not retry when the
// instance is in RUNNING state.
func TestGetRetryPolicy_RunningState(t *testing.T) {
	mgr := newTestManager(&fakeComputeClient{})
	policy := GetRetryPolicyForTest(mgr, 5)

	response := common.OCIOperationResponse{
		Response: core.GetInstanceResponse{
			Instance: core.Instance{
				LifecycleState: core.InstanceLifecycleStateRunning,
			},
		},
	}

	assert.False(t, policy.ShouldRetryOperation(response), "should not retry when state is RUNNING")
}

// TestCreateOrUpdate_LaunchError verifies CreateOrUpdate returns error when launch fails.
func TestCreateOrUpdate_LaunchError(t *testing.T) {
	ociClient := &fakeComputeClient{
		listFn: func(_ context.Context, _ core.ListInstancesRequest) (core.ListInstancesResponse, error) {
			return core.ListInstancesResponse{Items: []core.Instance{}}, nil
		},
		launchFn: func(_ context.Context, _ core.LaunchInstanceRequest) (core.LaunchInstanceResponse, error) {
			return core.LaunchInstanceResponse{}, errors.New("launch failed")
		},
	}
	mgr := newTestManager(ociClient)
	ci := makeComputeInstanceSpec("test-instance")

	resp, err := mgr.CreateOrUpdate(context.Background(), ci, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}
