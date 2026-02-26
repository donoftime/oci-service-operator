/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package devops_test

import (
	"context"
	"errors"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocidevops "github.com/oracle/oci-go-sdk/v65/devops"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/devops"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct {
	createCalled bool
	deleteCalled bool
}

func (f *fakeCredentialClient) CreateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	f.createCalled = true
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(_ context.Context, _, _ string) (bool, error) {
	f.deleteCalled = true
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(_ context.Context, _, _ string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(_ context.Context, _, _ string, _ map[string]string, _ map[string][]byte) (bool, error) {
	return true, nil
}

// TestDelete_NoOcid verifies deletion with no OCID set is a no-op.
func TestDelete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewDevopsProjectServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	project := &ociv1beta1.DevopsProject{}
	project.Name = "test-project"
	project.Namespace = "default"

	done, err := mgr.Delete(context.Background(), project)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called when OCID is empty")
}

// TestGetCrdStatus_ReturnsStatus verifies status extraction from a DevopsProject object.
func TestGetCrdStatus_ReturnsStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewDevopsProjectServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	project := &ociv1beta1.DevopsProject{}
	project.Status.OsokStatus.Ocid = "ocid1.devopsproject.oc1..xxx"

	status, err := mgr.GetCrdStatus(project)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.devopsproject.oc1..xxx"), status.Ocid)
}

// TestGetCrdStatus_WrongType verifies convert fails gracefully on wrong type.
func TestGetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewDevopsProjectServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestCreateOrUpdate_BadType verifies CreateOrUpdate rejects non-DevopsProject objects.
func TestCreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewDevopsProjectServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// --- Mock OCI DevOps client ---

// mockDevopsClient implements DevopsClientInterface for testing.
// Each method dispatches to a configurable function field; unset fields return zero values.
type mockDevopsClient struct {
	createProjectFn func(ctx context.Context, req ocidevops.CreateProjectRequest) (ocidevops.CreateProjectResponse, error)
	getProjectFn    func(ctx context.Context, req ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error)
	listProjectsFn  func(ctx context.Context, req ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error)
	updateProjectFn func(ctx context.Context, req ocidevops.UpdateProjectRequest) (ocidevops.UpdateProjectResponse, error)
	deleteProjectFn func(ctx context.Context, req ocidevops.DeleteProjectRequest) (ocidevops.DeleteProjectResponse, error)
}

func (m *mockDevopsClient) CreateProject(ctx context.Context, req ocidevops.CreateProjectRequest) (ocidevops.CreateProjectResponse, error) {
	if m.createProjectFn != nil {
		return m.createProjectFn(ctx, req)
	}
	return ocidevops.CreateProjectResponse{}, nil
}

func (m *mockDevopsClient) GetProject(ctx context.Context, req ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
	if m.getProjectFn != nil {
		return m.getProjectFn(ctx, req)
	}
	return ocidevops.GetProjectResponse{}, nil
}

func (m *mockDevopsClient) ListProjects(ctx context.Context, req ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
	if m.listProjectsFn != nil {
		return m.listProjectsFn(ctx, req)
	}
	return ocidevops.ListProjectsResponse{}, nil
}

func (m *mockDevopsClient) UpdateProject(ctx context.Context, req ocidevops.UpdateProjectRequest) (ocidevops.UpdateProjectResponse, error) {
	if m.updateProjectFn != nil {
		return m.updateProjectFn(ctx, req)
	}
	return ocidevops.UpdateProjectResponse{}, nil
}

func (m *mockDevopsClient) DeleteProject(ctx context.Context, req ocidevops.DeleteProjectRequest) (ocidevops.DeleteProjectResponse, error) {
	if m.deleteProjectFn != nil {
		return m.deleteProjectFn(ctx, req)
	}
	return ocidevops.DeleteProjectResponse{}, nil
}

// --- Test helpers ---

func newDevopsMgr(t *testing.T, ociClient *mockDevopsClient) *DevopsProjectServiceManager {
	t.Helper()
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
	mgr := NewDevopsProjectServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		&fakeCredentialClient{}, nil, log)
	if ociClient != nil {
		ExportSetClientForTest(mgr, ociClient)
	}
	return mgr
}

func makeActiveProject(id, name string) ocidevops.Project {
	return ocidevops.Project{
		Id:             common.String(id),
		Name:           common.String(name),
		CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
		LifecycleState: ocidevops.ProjectLifecycleStateActive,
		NotificationConfig: &ocidevops.NotificationConfig{
			TopicId: common.String("ocid1.onstopic.oc1..aaa"),
		},
	}
}

// --- GetDevopsProjectOcid tests ---

// TestGetDevopsProjectOcid_FoundActive verifies that a project in ACTIVE state is returned.
func TestGetDevopsProjectOcid_FoundActive(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..active"
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{
					Items: []ocidevops.ProjectSummary{
						{Id: common.String(projectId), LifecycleState: ocidevops.ProjectLifecycleStateActive},
					},
				},
			}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetDevopsProjectOcid(context.Background(), *project)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID(projectId), *ocid)
}

// TestGetDevopsProjectOcid_FoundCreating verifies that a CREATING project is returned (no duplicate create).
func TestGetDevopsProjectOcid_FoundCreating(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..creating"
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{
					Items: []ocidevops.ProjectSummary{
						{Id: common.String(projectId), LifecycleState: ocidevops.ProjectLifecycleStateCreating},
					},
				},
			}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetDevopsProjectOcid(context.Background(), *project)
	assert.NoError(t, err)
	assert.NotNil(t, ocid)
	assert.Equal(t, ociv1beta1.OCID(projectId), *ocid)
}

// TestGetDevopsProjectOcid_NotFound verifies that an empty list returns nil OCID without error.
func TestGetDevopsProjectOcid_NotFound(t *testing.T) {
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{
					Items: []ocidevops.ProjectSummary{},
				},
			}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "nonexistent"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetDevopsProjectOcid(context.Background(), *project)
	assert.NoError(t, err)
	assert.Nil(t, ocid)
}

// TestGetDevopsProjectOcid_ListError verifies that a ListProjects error propagates.
func TestGetDevopsProjectOcid_ListError(t *testing.T) {
	listErr := errors.New("OCI list error")
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{}, listErr
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	ocid, err := mgr.GetDevopsProjectOcid(context.Background(), *project)
	assert.ErrorIs(t, err, listErr)
	assert.Nil(t, ocid)
}

// --- CreateOrUpdate: create path tests ---

// TestCreateOrUpdate_CreatePath_NoExisting_Success verifies the full create path:
// no DevopsProjectId in spec, ListProjects returns empty, CreateProject succeeds,
// GetProject returns ACTIVE → IsSuccessful=true.
func TestCreateOrUpdate_CreatePath_NoExisting_Success(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..new"
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{Items: []ocidevops.ProjectSummary{}},
			}, nil
		},
		createProjectFn: func(_ context.Context, _ ocidevops.CreateProjectRequest) (ocidevops.CreateProjectResponse, error) {
			return ocidevops.CreateProjectResponse{
				Project: ocidevops.Project{Id: common.String(projectId)},
			}, nil
		},
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{
				Project: makeActiveProject(projectId, "my-project"),
			}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	project.Spec.NotificationTopicId = "ocid1.onstopic.oc1..aaa"

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(projectId), project.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_CreatePath_ListError verifies that a ListProjects error on create path
// returns IsSuccessful=false with the error.
func TestCreateOrUpdate_CreatePath_ListError(t *testing.T) {
	listErr := errors.New("OCI list error")
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{}, listErr
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.ErrorIs(t, err, listErr)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreatePath_CreateError verifies that a CreateProject OCI error
// propagates and returns IsSuccessful=false.
func TestCreateOrUpdate_CreatePath_CreateError(t *testing.T) {
	createErr := errors.New("OCI create error")
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{Items: []ocidevops.ProjectSummary{}},
			}, nil
		},
		createProjectFn: func(_ context.Context, _ ocidevops.CreateProjectRequest) (ocidevops.CreateProjectResponse, error) {
			return ocidevops.CreateProjectResponse{}, createErr
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	project.Spec.NotificationTopicId = "ocid1.onstopic.oc1..aaa"

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.ErrorIs(t, err, createErr)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreatePath_GetAfterCreateError verifies that a GetProject error after
// successful CreateProject returns IsSuccessful=false.
func TestCreateOrUpdate_CreatePath_GetAfterCreateError(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..new"
	getErr := errors.New("OCI get error")
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{Items: []ocidevops.ProjectSummary{}},
			}, nil
		},
		createProjectFn: func(_ context.Context, _ ocidevops.CreateProjectRequest) (ocidevops.CreateProjectResponse, error) {
			return ocidevops.CreateProjectResponse{
				Project: ocidevops.Project{Id: common.String(projectId)},
			}, nil
		},
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{}, getErr
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	project.Spec.NotificationTopicId = "ocid1.onstopic.oc1..aaa"

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.ErrorIs(t, err, getErr)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreatePath_FailedState verifies that a FAILED project lifecycle state
// results in IsSuccessful=false with no error.
func TestCreateOrUpdate_CreatePath_FailedState(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..failed"
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{Items: []ocidevops.ProjectSummary{}},
			}, nil
		},
		createProjectFn: func(_ context.Context, _ ocidevops.CreateProjectRequest) (ocidevops.CreateProjectResponse, error) {
			return ocidevops.CreateProjectResponse{
				Project: ocidevops.Project{Id: common.String(projectId)},
			}, nil
		},
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{
				Project: ocidevops.Project{
					Id:             common.String(projectId),
					Name:           common.String("my-project"),
					CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
					LifecycleState: ocidevops.ProjectLifecycleStateFailed,
					NotificationConfig: &ocidevops.NotificationConfig{
						TopicId: common.String("ocid1.onstopic.oc1..aaa"),
					},
				},
			}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	project.Spec.NotificationTopicId = "ocid1.onstopic.oc1..aaa"

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.NoError(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_CreatePath_ExistingByName verifies that when ListProjects returns a project
// with ACTIVE state, GetProject is called to bind it and the operation succeeds.
func TestCreateOrUpdate_CreatePath_ExistingByName(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..existing"
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{
					Items: []ocidevops.ProjectSummary{
						{Id: common.String(projectId), LifecycleState: ocidevops.ProjectLifecycleStateActive},
					},
				},
			}, nil
		},
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{
				Project: makeActiveProject(projectId, "my-project"),
			}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(projectId), project.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_CreatePath_GetExistingError verifies that GetProject error when binding
// by name returns IsSuccessful=false.
func TestCreateOrUpdate_CreatePath_GetExistingError(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..existing"
	getErr := errors.New("OCI get error")
	ociClient := &mockDevopsClient{
		listProjectsFn: func(_ context.Context, _ ocidevops.ListProjectsRequest) (ocidevops.ListProjectsResponse, error) {
			return ocidevops.ListProjectsResponse{
				ProjectCollection: ocidevops.ProjectCollection{
					Items: []ocidevops.ProjectSummary{
						{Id: common.String(projectId), LifecycleState: ocidevops.ProjectLifecycleStateActive},
					},
				},
			}, nil
		},
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{}, getErr
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.ErrorIs(t, err, getErr)
	assert.False(t, resp.IsSuccessful)
}

// --- CreateOrUpdate: update path tests ---

// TestCreateOrUpdate_UpdatePath_Success verifies the update path when DevopsProjectId is set
// and there are no changes needed (no description, notification topic unchanged).
func TestCreateOrUpdate_UpdatePath_Success(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..existing"
	ociClient := &mockDevopsClient{
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{
				Project: makeActiveProject(projectId, "my-project"),
			}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	project.Spec.DevopsProjectId = ociv1beta1.OCID(projectId)
	project.Status.OsokStatus.Ocid = ociv1beta1.OCID(projectId)

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, ociv1beta1.OCID(projectId), project.Status.OsokStatus.Ocid)
}

// TestCreateOrUpdate_UpdatePath_GetError verifies that a GetProject error in the update path
// returns IsSuccessful=false.
func TestCreateOrUpdate_UpdatePath_GetError(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..existing"
	getErr := errors.New("OCI get error")
	ociClient := &mockDevopsClient{
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{}, getErr
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.DevopsProjectId = ociv1beta1.OCID(projectId)

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.ErrorIs(t, err, getErr)
	assert.False(t, resp.IsSuccessful)
}

// TestCreateOrUpdate_UpdatePath_NotificationTopicChange verifies that when the notification
// topic ID changes, UpdateProject is called with the new topic.
func TestCreateOrUpdate_UpdatePath_NotificationTopicChange(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..existing"
	oldTopicId := "ocid1.onstopic.oc1..old"
	newTopicId := "ocid1.onstopic.oc1..new"
	updateCalled := false

	ociClient := &mockDevopsClient{
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{
				Project: ocidevops.Project{
					Id:             common.String(projectId),
					Name:           common.String("my-project"),
					CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
					LifecycleState: ocidevops.ProjectLifecycleStateActive,
					NotificationConfig: &ocidevops.NotificationConfig{
						TopicId: common.String(oldTopicId),
					},
				},
			}, nil
		},
		updateProjectFn: func(_ context.Context, req ocidevops.UpdateProjectRequest) (ocidevops.UpdateProjectResponse, error) {
			updateCalled = true
			assert.Equal(t, newTopicId, *req.UpdateProjectDetails.NotificationConfig.TopicId)
			return ocidevops.UpdateProjectResponse{}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	project.Spec.DevopsProjectId = ociv1beta1.OCID(projectId)
	project.Spec.NotificationTopicId = ociv1beta1.OCID(newTopicId)
	project.Status.OsokStatus.Ocid = ociv1beta1.OCID(projectId)

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.True(t, updateCalled, "UpdateProject should have been called")
}

// TestCreateOrUpdate_UpdatePath_UpdateError verifies that an UpdateProject error in the update
// path returns IsSuccessful=false.
func TestCreateOrUpdate_UpdatePath_UpdateError(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..existing"
	updateErr := errors.New("OCI update error")

	ociClient := &mockDevopsClient{
		getProjectFn: func(_ context.Context, _ ocidevops.GetProjectRequest) (ocidevops.GetProjectResponse, error) {
			return ocidevops.GetProjectResponse{
				Project: ocidevops.Project{
					Id:             common.String(projectId),
					Name:           common.String("my-project"),
					CompartmentId:  common.String("ocid1.compartment.oc1..xxx"),
					LifecycleState: ocidevops.ProjectLifecycleStateActive,
					NotificationConfig: &ocidevops.NotificationConfig{
						TopicId: common.String("ocid1.onstopic.oc1..old"),
					},
				},
			}, nil
		},
		updateProjectFn: func(_ context.Context, _ ocidevops.UpdateProjectRequest) (ocidevops.UpdateProjectResponse, error) {
			return ocidevops.UpdateProjectResponse{}, updateErr
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Spec.Name = "my-project"
	project.Spec.CompartmentId = "ocid1.compartment.oc1..xxx"
	project.Spec.DevopsProjectId = ociv1beta1.OCID(projectId)
	project.Spec.NotificationTopicId = "ocid1.onstopic.oc1..new"
	project.Status.OsokStatus.Ocid = ociv1beta1.OCID(projectId)

	resp, err := mgr.CreateOrUpdate(context.Background(), project, ctrl.Request{})
	assert.ErrorIs(t, err, updateErr)
	assert.False(t, resp.IsSuccessful)
}

// --- Delete tests ---

// TestDelete_WithOcid verifies that Delete calls DeleteProject when OCID is set.
func TestDelete_WithOcid(t *testing.T) {
	projectId := "ocid1.devopsproject.oc1..existing"
	deleteCalled := false
	ociClient := &mockDevopsClient{
		deleteProjectFn: func(_ context.Context, req ocidevops.DeleteProjectRequest) (ocidevops.DeleteProjectResponse, error) {
			deleteCalled = true
			assert.Equal(t, projectId, *req.ProjectId)
			return ocidevops.DeleteProjectResponse{}, nil
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Status.OsokStatus.Ocid = ociv1beta1.OCID(projectId)

	done, err := mgr.Delete(context.Background(), project)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.True(t, deleteCalled, "DeleteProject should have been called")
}

// TestDelete_DeleteError verifies that a DeleteProject OCI error returns false with the error.
func TestDelete_DeleteError(t *testing.T) {
	deleteErr := errors.New("OCI delete error")
	ociClient := &mockDevopsClient{
		deleteProjectFn: func(_ context.Context, _ ocidevops.DeleteProjectRequest) (ocidevops.DeleteProjectResponse, error) {
			return ocidevops.DeleteProjectResponse{}, deleteErr
		},
	}
	mgr := newDevopsMgr(t, ociClient)

	project := &ociv1beta1.DevopsProject{}
	project.Status.OsokStatus.Ocid = "ocid1.devopsproject.oc1..existing"

	done, err := mgr.Delete(context.Background(), project)
	assert.ErrorIs(t, err, deleteErr)
	assert.False(t, done)
}
