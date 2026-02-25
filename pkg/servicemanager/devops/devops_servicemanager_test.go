/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package devops_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
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
