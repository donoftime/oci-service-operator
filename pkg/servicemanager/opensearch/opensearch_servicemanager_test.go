/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package opensearch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	sdkopensearch "github.com/oracle/oci-go-sdk/v65/opensearch"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/opensearch"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ---------------------------------------------------------------------------
// Fake credential client
// ---------------------------------------------------------------------------

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct{}

func (f *fakeCredentialClient) CreateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}
func (f *fakeCredentialClient) DeleteSecret(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
func (f *fakeCredentialClient) GetSecret(_ context.Context, _, _ string) (map[string][]byte, error) {
	return nil, nil
}
func (f *fakeCredentialClient) UpdateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}

// ---------------------------------------------------------------------------
// Mock OCI client
// ---------------------------------------------------------------------------

// mockOpensearchClient implements OpensearchClusterClientInterface for unit tests.
type mockOpensearchClient struct {
	createFn func(context.Context, sdkopensearch.CreateOpensearchClusterRequest) (sdkopensearch.CreateOpensearchClusterResponse, error)
	getFn    func(context.Context, sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error)
	listFn   func(context.Context, sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error)
	updateFn func(context.Context, sdkopensearch.UpdateOpensearchClusterRequest) (sdkopensearch.UpdateOpensearchClusterResponse, error)
	deleteFn func(context.Context, sdkopensearch.DeleteOpensearchClusterRequest) (sdkopensearch.DeleteOpensearchClusterResponse, error)
}

func (m *mockOpensearchClient) CreateOpensearchCluster(ctx context.Context, req sdkopensearch.CreateOpensearchClusterRequest) (sdkopensearch.CreateOpensearchClusterResponse, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return sdkopensearch.CreateOpensearchClusterResponse{}, nil
}

func (m *mockOpensearchClient) GetOpensearchCluster(ctx context.Context, req sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, req)
	}
	return sdkopensearch.GetOpensearchClusterResponse{}, nil
}

func (m *mockOpensearchClient) ListOpensearchClusters(ctx context.Context, req sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, req)
	}
	return sdkopensearch.ListOpensearchClustersResponse{}, nil
}

func (m *mockOpensearchClient) UpdateOpensearchCluster(ctx context.Context, req sdkopensearch.UpdateOpensearchClusterRequest) (sdkopensearch.UpdateOpensearchClusterResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return sdkopensearch.UpdateOpensearchClusterResponse{}, nil
}

func (m *mockOpensearchClient) DeleteOpensearchCluster(ctx context.Context, req sdkopensearch.DeleteOpensearchClusterRequest) (sdkopensearch.DeleteOpensearchClusterResponse, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, req)
	}
	return sdkopensearch.DeleteOpensearchClusterResponse{}, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func makeManager() *OpenSearchClusterServiceManager {
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	m := &metrics.Metrics{Logger: log}
	return NewOpenSearchClusterServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		&fakeCredentialClient{}, nil, log, m)
}

func makeManagerWithClient(mock *mockOpensearchClient) *OpenSearchClusterServiceManager {
	mgr := makeManager()
	ExportSetClientForTest(mgr, mock)
	return mgr
}

// makeCluster builds a minimal OpenSearchCluster spec for test inputs.
func makeCluster(displayName string) *ociv1beta1.OpenSearchCluster {
	return &ociv1beta1.OpenSearchCluster{
		Spec: ociv1beta1.OpenSearchClusterSpec{
			CompartmentId: "ocid1.compartment.oc1..xxx",
			DisplayName:   displayName,
			SoftwareVersion: "2.11.0",
			MasterNodeCount:        3,
			MasterNodeHostType:     "FLEX",
			MasterNodeHostOcpuCount: 4,
			MasterNodeHostMemoryGB: 32,
			DataNodeCount:          2,
			DataNodeHostType:       "FLEX",
			DataNodeHostOcpuCount:  4,
			DataNodeHostMemoryGB:   32,
			DataNodeStorageGB:      50,
			OpendashboardNodeCount:          1,
			OpendashboardNodeHostOcpuCount:  4,
			OpendashboardNodeHostMemoryGB:   32,
			VcnId:               "ocid1.vcn.oc1..xxx",
			SubnetId:            "ocid1.subnet.oc1..xxx",
			VcnCompartmentId:    "ocid1.compartment.oc1..xxx",
			SubnetCompartmentId: "ocid1.compartment.oc1..xxx",
		},
	}
}

// makeActiveClusterInstance returns a mock OCI OpensearchCluster in Active state.
func makeActiveClusterInstance(id, displayName string) sdkopensearch.OpensearchCluster {
	return sdkopensearch.OpensearchCluster{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: sdkopensearch.OpensearchClusterLifecycleStateActive,
	}
}

// makeClusterWithState returns a mock OCI OpensearchCluster in the given lifecycle state.
func makeClusterWithState(id, displayName string, state sdkopensearch.OpensearchClusterLifecycleStateEnum) sdkopensearch.OpensearchCluster {
	return sdkopensearch.OpensearchCluster{
		Id:             common.String(id),
		DisplayName:    common.String(displayName),
		LifecycleState: state,
	}
}

// makeClusterSummary returns a minimal OpensearchClusterSummary.
func makeClusterSummary(id string, state sdkopensearch.OpensearchClusterLifecycleStateEnum) sdkopensearch.OpensearchClusterSummary {
	return sdkopensearch.OpensearchClusterSummary{
		Id:             common.String(id),
		LifecycleState: state,
	}
}

// ---------------------------------------------------------------------------
// GetCrdStatus tests
// ---------------------------------------------------------------------------

// TestGetCrdStatus_ReturnsStatus verifies status extraction from an OpenSearchCluster object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	mgr := makeManager()

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..xxx"

	status, err := mgr.GetCrdStatus(cluster)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.opensearchcluster.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	mgr := makeManager()

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert type assertion")
}

// ---------------------------------------------------------------------------
// CreateOrUpdate — type conversion failure
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-OpenSearchCluster objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	mgr := makeManager()

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// CreateOrUpdate — explicit OCID paths
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_ExplicitOcid_ActiveNoUpdate verifies that binding to an existing cluster
// by explicit OCID with no update needed results in an Active, successful response.
func TestCreateOrUpdate_ExplicitOcid_ActiveNoUpdate(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..aaa"
	mock := &mockOpensearchClient{
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeActiveClusterInstance(clusterID, "my-cluster"),
			}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")
	cluster.Spec.OpenSearchClusterId = ociv1beta1.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(clusterID), cluster.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_ExplicitOcid_UpdateDisplayName verifies that a display-name change
// triggers UpdateOpenSearchCluster and sets Updating status.
func TestCreateOrUpdate_ExplicitOcid_UpdateDisplayName(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..bbb"
	updateCalled := false
	mock := &mockOpensearchClient{
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeActiveClusterInstance(clusterID, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ sdkopensearch.UpdateOpensearchClusterRequest) (sdkopensearch.UpdateOpensearchClusterResponse, error) {
			updateCalled = true
			return sdkopensearch.UpdateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("new-name")
	cluster.Spec.OpenSearchClusterId = ociv1beta1.OCID(clusterID)
	cluster.Status.OsokStatus.Ocid = ociv1beta1.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}

// TestCreateOrUpdate_ExplicitOcid_UpdateFails verifies that an update error is propagated.
func TestCreateOrUpdate_ExplicitOcid_UpdateFails(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..ccc"
	mock := &mockOpensearchClient{
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeActiveClusterInstance(clusterID, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ sdkopensearch.UpdateOpensearchClusterRequest) (sdkopensearch.UpdateOpensearchClusterResponse, error) {
			return sdkopensearch.UpdateOpensearchClusterResponse{}, errors.New("update failed")
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("new-name")
	cluster.Spec.OpenSearchClusterId = ociv1beta1.OCID(clusterID)
	cluster.Status.OsokStatus.Ocid = ociv1beta1.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_ExplicitOcid_GetFails verifies that a Get error is propagated.
func TestCreateOrUpdate_ExplicitOcid_GetFails(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..ddd"
	mock := &mockOpensearchClient{
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{}, errors.New("get failed")
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")
	cluster.Spec.OpenSearchClusterId = ociv1beta1.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_ExplicitOcid_LifecycleCreating verifies CREATING state returns Provisioning.
func TestCreateOrUpdate_ExplicitOcid_LifecycleCreating(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..eee"
	mock := &mockOpensearchClient{
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeClusterWithState(clusterID, "my-cluster", sdkopensearch.OpensearchClusterLifecycleStateCreating),
			}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")
	cluster.Spec.OpenSearchClusterId = ociv1beta1.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	// Default lifecycle state (not Active, not Failed) sets Provisioning
	found := false
	for _, cond := range cluster.Status.OsokStatus.Conditions {
		if cond.Type == ociv1beta1.Provisioning {
			found = true
		}
	}
	assert.True(t, found, "expected Provisioning condition for CREATING lifecycle state")
}

// TestCreateOrUpdate_ExplicitOcid_LifecycleFailed verifies FAILED state sets Failed status.
func TestCreateOrUpdate_ExplicitOcid_LifecycleFailed(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..fff"
	mock := &mockOpensearchClient{
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeClusterWithState(clusterID, "my-cluster", sdkopensearch.OpensearchClusterLifecycleStateFailed),
			}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")
	cluster.Spec.OpenSearchClusterId = ociv1beta1.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	found := false
	for _, cond := range cluster.Status.OsokStatus.Conditions {
		if cond.Type == ociv1beta1.Failed {
			found = true
		}
	}
	assert.True(t, found, "expected Failed condition for FAILED lifecycle state")
}

// ---------------------------------------------------------------------------
// CreateOrUpdate — no explicit OCID (lookup by name)
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_NoOcid_CreateNew verifies that when no cluster exists, CreateOpensearchCluster
// is called and the response signals provisioning in progress.
func TestCreateOrUpdate_NoOcid_CreateNew(t *testing.T) {
	createCalled := false
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{Items: []sdkopensearch.OpensearchClusterSummary{}},
			}, nil
		},
		createFn: func(_ context.Context, _ sdkopensearch.CreateOpensearchClusterRequest) (sdkopensearch.CreateOpensearchClusterResponse, error) {
			createCalled = true
			return sdkopensearch.CreateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("new-cluster")

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful) // provisioning — not yet done
	assert.True(t, createCalled)
}

// TestCreateOrUpdate_NoOcid_CreateFails verifies create errors are propagated.
func TestCreateOrUpdate_NoOcid_CreateFails(t *testing.T) {
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{Items: []sdkopensearch.OpensearchClusterSummary{}},
			}, nil
		},
		createFn: func(_ context.Context, _ sdkopensearch.CreateOpensearchClusterRequest) (sdkopensearch.CreateOpensearchClusterResponse, error) {
			return sdkopensearch.CreateOpensearchClusterResponse{}, errors.New("create failed")
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("new-cluster")

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_NoOcid_GetOcidFails verifies list errors during OCID lookup are propagated.
func TestCreateOrUpdate_NoOcid_GetOcidFails(t *testing.T) {
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{}, errors.New("list failed")
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_NoOcid_ExistingCluster_Active verifies the path when the cluster is found
// by name lookup, is Active, and no update is needed.
func TestCreateOrUpdate_NoOcid_ExistingCluster_Active(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..ggg"
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{
					Items: []sdkopensearch.OpensearchClusterSummary{
						makeClusterSummary(clusterID, sdkopensearch.OpensearchClusterLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeActiveClusterInstance(clusterID, "my-cluster"),
			}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster") // same display name — no update

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(clusterID), cluster.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_NoOcid_ExistingCluster_Update verifies that a display-name change triggers
// an update when the cluster was found by name.
func TestCreateOrUpdate_NoOcid_ExistingCluster_Update(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..hhh"
	updateCalled := false
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{
					Items: []sdkopensearch.OpensearchClusterSummary{
						makeClusterSummary(clusterID, sdkopensearch.OpensearchClusterLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeActiveClusterInstance(clusterID, "old-name"),
			}, nil
		},
		updateFn: func(_ context.Context, _ sdkopensearch.UpdateOpensearchClusterRequest) (sdkopensearch.UpdateOpensearchClusterResponse, error) {
			updateCalled = true
			return sdkopensearch.UpdateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("new-name")
	cluster.Status.OsokStatus.Ocid = ociv1beta1.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}

// TestCreateOrUpdate_NoOcid_ExistingCluster_GetFails verifies that a Get failure after OCID
// lookup is propagated.
func TestCreateOrUpdate_NoOcid_ExistingCluster_GetFails(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..iii"
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{
					Items: []sdkopensearch.OpensearchClusterSummary{
						makeClusterSummary(clusterID, sdkopensearch.OpensearchClusterLifecycleStateActive),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{}, errors.New("get failed")
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// ---------------------------------------------------------------------------
// CreateOrUpdate — endpoint fields in cluster response
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_ExplicitOcid_WithEndpointFields verifies that clusters with endpoint
// fields (OpensearchFqdn, OpensearchPrivateIp) are handled correctly.
func TestCreateOrUpdate_ExplicitOcid_WithEndpointFields(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..jjj"
	clusterWithEndpoints := sdkopensearch.OpensearchCluster{
		Id:                   common.String(clusterID),
		DisplayName:          common.String("my-cluster"),
		LifecycleState:       sdkopensearch.OpensearchClusterLifecycleStateActive,
		OpensearchFqdn:       common.String("search.us-phoenix-1.oci.oraclecloud.com"),
		OpensearchPrivateIp:  common.String("10.0.1.100"),
		OpendashboardFqdn:    common.String("dashboard.us-phoenix-1.oci.oraclecloud.com"),
		OpendashboardPrivateIp: common.String("10.0.1.101"),
	}
	mock := &mockOpensearchClient{
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: clusterWithEndpoints,
			}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")
	cluster.Spec.OpenSearchClusterId = ociv1beta1.OCID(clusterID)

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(clusterID), cluster.Status.OsokStatus.Ocid)
}

// ---------------------------------------------------------------------------
// GetOpenSearchClusterOCID — lifecycle state filtering
// ---------------------------------------------------------------------------

// TestGetClusterOcid_ActiveState verifies that an Active cluster OCID is returned.
func TestGetClusterOcid_ActiveState(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..active"
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{
					Items: []sdkopensearch.OpensearchClusterSummary{
						makeClusterSummary(clusterID, sdkopensearch.OpensearchClusterLifecycleStateActive),
					},
				},
			}, nil
		},
	}
	mgr := makeManagerWithClient(mock)
	cluster := makeCluster("my-cluster")

	// Exercise via CreateOrUpdate (no explicit OCID) which calls GetOpenSearchClusterOCID.
	// getFn not set, so after OCID lookup, Get will return empty cluster → nil clusterInstance → no crash.
	mock.getFn = func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
		return sdkopensearch.GetOpensearchClusterResponse{
			OpensearchCluster: makeActiveClusterInstance(clusterID, "my-cluster"),
		}, nil
	}

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(clusterID), cluster.Status.OsokStatus.Ocid)
}

// TestGetClusterOcid_CreatingState verifies that a Creating cluster OCID is returned.
func TestGetClusterOcid_CreatingState(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..creating"
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{
					Items: []sdkopensearch.OpensearchClusterSummary{
						makeClusterSummary(clusterID, sdkopensearch.OpensearchClusterLifecycleStateCreating),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeClusterWithState(clusterID, "my-cluster", sdkopensearch.OpensearchClusterLifecycleStateCreating),
			}, nil
		},
	}
	mgr := makeManagerWithClient(mock)
	cluster := makeCluster("my-cluster")

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(clusterID), cluster.Status.OsokStatus.Ocid)
}

// TestGetClusterOcid_UpdatingState verifies that an Updating cluster OCID is returned.
func TestGetClusterOcid_UpdatingState(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..updating"
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{
					Items: []sdkopensearch.OpensearchClusterSummary{
						makeClusterSummary(clusterID, sdkopensearch.OpensearchClusterLifecycleStateUpdating),
					},
				},
			}, nil
		},
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			return sdkopensearch.GetOpensearchClusterResponse{
				OpensearchCluster: makeClusterWithState(clusterID, "my-cluster", sdkopensearch.OpensearchClusterLifecycleStateUpdating),
			}, nil
		},
	}
	mgr := makeManagerWithClient(mock)
	cluster := makeCluster("my-cluster")

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
}

// TestGetClusterOcid_FailedState verifies that a Failed cluster is not returned (filtered out).
// With no usable OCID found, CreateOpensearchCluster is called instead.
func TestGetClusterOcid_FailedState(t *testing.T) {
	createCalled := false
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{
					Items: []sdkopensearch.OpensearchClusterSummary{
						makeClusterSummary("ocid1.opensearchcluster.oc1..failed", sdkopensearch.OpensearchClusterLifecycleStateFailed),
					},
				},
			}, nil
		},
		createFn: func(_ context.Context, _ sdkopensearch.CreateOpensearchClusterRequest) (sdkopensearch.CreateOpensearchClusterResponse, error) {
			createCalled = true
			return sdkopensearch.CreateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithClient(mock)
	cluster := makeCluster("my-cluster")

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful) // provisioning — create was triggered
	assert.True(t, createCalled, "expected Create to be called when Failed cluster is filtered out")
}

// TestGetClusterOcid_EmptyList verifies that an empty list results in cluster creation.
func TestGetClusterOcid_EmptyList(t *testing.T) {
	createCalled := false
	mock := &mockOpensearchClient{
		listFn: func(_ context.Context, _ sdkopensearch.ListOpensearchClustersRequest) (sdkopensearch.ListOpensearchClustersResponse, error) {
			return sdkopensearch.ListOpensearchClustersResponse{
				OpensearchClusterCollection: sdkopensearch.OpensearchClusterCollection{Items: []sdkopensearch.OpensearchClusterSummary{}},
			}, nil
		},
		createFn: func(_ context.Context, _ sdkopensearch.CreateOpensearchClusterRequest) (sdkopensearch.CreateOpensearchClusterResponse, error) {
			createCalled = true
			return sdkopensearch.CreateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithClient(mock)
	cluster := makeCluster("my-cluster")

	_, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, createCalled)
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	mgr := makeManager()

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestDelete_WithStatusOcid verifies deletion is attempted when status OCID is set.
func TestDelete_WithStatusOcid(t *testing.T) {
	deleteCalled := false
	mock := &mockOpensearchClient{
		deleteFn: func(_ context.Context, _ sdkopensearch.DeleteOpensearchClusterRequest) (sdkopensearch.DeleteOpensearchClusterResponse, error) {
			deleteCalled = true
			return sdkopensearch.DeleteOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := &ociv1beta1.OpenSearchCluster{}
	cluster.Name = "test-cluster"
	cluster.Namespace = "default"
	cluster.Status.OsokStatus.Ocid = "ocid1.opensearchcluster.oc1..xxx"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// TestDelete_WrongType verifies Delete handles wrong object type gracefully.
func TestDelete_WrongType(t *testing.T) {
	mgr := makeManager()

	stream := &ociv1beta1.Stream{}
	done, err := mgr.Delete(context.Background(), stream)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestDelete_WithSpecOcid verifies deletion uses spec OCID when status OCID is empty.
func TestDelete_WithSpecOcid(t *testing.T) {
	deleteCalled := false
	mock := &mockOpensearchClient{
		deleteFn: func(_ context.Context, _ sdkopensearch.DeleteOpensearchClusterRequest) (sdkopensearch.DeleteOpensearchClusterResponse, error) {
			deleteCalled = true
			return sdkopensearch.DeleteOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")
	cluster.Spec.OpenSearchClusterId = "ocid1.opensearchcluster.oc1..specid"

	done, err := mgr.Delete(context.Background(), cluster)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled)
}

// ---------------------------------------------------------------------------
// FreeformTags / DefinedTags update detection
// ---------------------------------------------------------------------------

// TestCreateOrUpdate_FreeformTagsUpdate verifies that freeform tag changes trigger an update.
func TestCreateOrUpdate_FreeformTagsUpdate(t *testing.T) {
	const clusterID = "ocid1.opensearchcluster.oc1..tags"
	updateCalled := false
	mock := &mockOpensearchClient{
		getFn: func(_ context.Context, _ sdkopensearch.GetOpensearchClusterRequest) (sdkopensearch.GetOpensearchClusterResponse, error) {
			inst := makeActiveClusterInstance(clusterID, "my-cluster")
			inst.FreeformTags = map[string]string{"env": "prod"}
			return sdkopensearch.GetOpensearchClusterResponse{OpensearchCluster: inst}, nil
		},
		updateFn: func(_ context.Context, _ sdkopensearch.UpdateOpensearchClusterRequest) (sdkopensearch.UpdateOpensearchClusterResponse, error) {
			updateCalled = true
			return sdkopensearch.UpdateOpensearchClusterResponse{}, nil
		},
	}
	mgr := makeManagerWithClient(mock)

	cluster := makeCluster("my-cluster")
	cluster.Spec.OpenSearchClusterId = ociv1beta1.OCID(clusterID)
	cluster.Status.OsokStatus.Ocid = ociv1beta1.OCID(clusterID)
	cluster.Spec.FreeFormTags = map[string]string{"env": "dev"} // differs from "prod"

	resp, err := mgr.CreateOrUpdate(context.Background(), cluster, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled)
}
