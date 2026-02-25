/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/functions"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

// fakeCredentialClient implements credhelper.CredentialClient for testing.
type fakeCredentialClient struct {
	createSecretFn func(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error)
	deleteSecretFn func(ctx context.Context, name, ns string) (bool, error)
	createCalled   bool
	deleteCalled   bool
}

func (f *fakeCredentialClient) CreateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	f.createCalled = true
	if f.createSecretFn != nil {
		return f.createSecretFn(ctx, name, ns, labels, data)
	}
	return true, nil
}

func (f *fakeCredentialClient) DeleteSecret(ctx context.Context, name, ns string) (bool, error) {
	f.deleteCalled = true
	if f.deleteSecretFn != nil {
		return f.deleteSecretFn(ctx, name, ns)
	}
	return true, nil
}

func (f *fakeCredentialClient) GetSecret(ctx context.Context, name, ns string) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCredentialClient) UpdateSecret(ctx context.Context, name, ns string, labels map[string]string, data map[string][]byte) (bool, error) {
	return true, nil
}

// --- FunctionsApplication tests ---

// TestFunctionsApplication_Delete_NoOcid verifies deletion with no OCID set is a no-op.
func TestFunctionsApplication_Delete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	app := &ociv1beta1.FunctionsApplication{}
	app.Name = "test-app"
	app.Namespace = "default"

	done, err := mgr.Delete(context.Background(), app)
	assert.NoError(t, err)
	assert.True(t, done)
}

// TestFunctionsApplication_GetCrdStatus verifies status extraction.
func TestFunctionsApplication_GetCrdStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	app := &ociv1beta1.FunctionsApplication{}
	app.Status.OsokStatus.Ocid = "ocid1.fnapp.oc1..xxx"

	status, err := mgr.GetCrdStatus(app)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.fnapp.oc1..xxx"), status.Ocid)
}

// TestFunctionsApplication_GetCrdStatus_WrongType verifies convert fails on wrong type.
func TestFunctionsApplication_GetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestFunctionsApplication_CreateOrUpdate_BadType verifies CreateOrUpdate rejects non-FunctionsApplication objects.
func TestFunctionsApplication_CreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsApplicationServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// --- FunctionsFunction tests ---

// TestFunctionsFunction_Delete_NoOcid verifies deletion with no OCID set is a no-op.
func TestFunctionsFunction_Delete_NoOcid(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Name = "test-fn"
	fn.Namespace = "default"

	done, err := mgr.Delete(context.Background(), fn)
	assert.NoError(t, err)
	assert.True(t, done)
	assert.False(t, credClient.deleteCalled, "DeleteSecret should not be called when OCID is empty")
}

// TestFunctionsFunction_GetCrdStatus verifies status extraction.
func TestFunctionsFunction_GetCrdStatus(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	fn := &ociv1beta1.FunctionsFunction{}
	fn.Status.OsokStatus.Ocid = "ocid1.fnfunc.oc1..xxx"

	status, err := mgr.GetCrdStatus(fn)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.fnfunc.oc1..xxx"), status.Ocid)
}

// TestFunctionsFunction_GetCrdStatus_WrongType verifies convert fails on wrong type.
func TestFunctionsFunction_GetCrdStatus_WrongType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed type assertion")
}

// TestFunctionsFunction_CreateOrUpdate_BadType verifies CreateOrUpdate rejects non-FunctionsFunction objects.
func TestFunctionsFunction_CreateOrUpdate_BadType(t *testing.T) {
	credClient := &fakeCredentialClient{}
	log := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}

	mgr := NewFunctionsFunctionServiceManager(
		common.NewRawConfigurationProvider("", "", "", "", "", nil),
		credClient, nil, log)

	stream := &ociv1beta1.Stream{}
	resp, err := mgr.CreateOrUpdate(context.Background(), stream, ctrl.Request{})
	assert.Error(t, err)
	assert.False(t, resp.IsSuccessful)
}

// TestGetFunctionCredentialMap verifies the secret credential map is built correctly.
func TestGetFunctionCredentialMap(t *testing.T) {
	fn := ocifunctions.Function{
		Id:             common.String("ocid1.fnfunc.oc1..xxx"),
		DisplayName:    common.String("test-fn"),
		InvokeEndpoint: common.String("https://xyz.functions.oci.example.com/20181201/functions/ocid1.fnfunc.oc1..xxx/actions/invoke"),
	}

	credMap := GetFunctionCredentialMapForTest(fn)
	assert.Equal(t, "ocid1.fnfunc.oc1..xxx", string(credMap["functionId"]))
	assert.Contains(t, string(credMap["invokeEndpoint"]), "invoke")
}

// TestGetFunctionCredentialMap_NilFields verifies nil pointer fields are handled gracefully.
func TestGetFunctionCredentialMap_NilFields(t *testing.T) {
	fn := ocifunctions.Function{
		Id: common.String("ocid1.fnfunc.oc1..xxx"),
	}
	credMap := GetFunctionCredentialMapForTest(fn)
	assert.NotContains(t, credMap, "invokeEndpoint")
	assert.Equal(t, "ocid1.fnfunc.oc1..xxx", string(credMap["functionId"]))
}
