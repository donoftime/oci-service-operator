/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	. "github.com/oracle/oci-service-operator/pkg/servicemanager/apigateway"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
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

func makeLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")}
}

func TestGatewayServiceManager_GetCrdStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	mgr := NewGatewayServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeCredentialClient{}, scheme, makeLogger())

	gw := &ociv1beta1.ApiGateway{}
	gw.Status.OsokStatus.Ocid = "ocid1.apigateway.oc1..xxx"

	status, err := mgr.GetCrdStatus(gw)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.apigateway.oc1..xxx"), status.Ocid)
}

func TestGatewayServiceManager_GetCrdStatus_WrongType(t *testing.T) {
	scheme := runtime.NewScheme()
	mgr := NewGatewayServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeCredentialClient{}, scheme, makeLogger())

	// Pass a wrong type
	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
}

func TestDeploymentServiceManager_GetCrdStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	mgr := NewDeploymentServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeCredentialClient{}, scheme, makeLogger())

	dep := &ociv1beta1.ApiGatewayDeployment{}
	dep.Status.OsokStatus.Ocid = "ocid1.apigateway.deployment.oc1..xxx"

	status, err := mgr.GetCrdStatus(dep)
	assert.NoError(t, err)
	assert.Equal(t, ociv1beta1.OCID("ocid1.apigateway.deployment.oc1..xxx"), status.Ocid)
}

func TestDeploymentServiceManager_GetCrdStatus_WrongType(t *testing.T) {
	scheme := runtime.NewScheme()
	mgr := NewDeploymentServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeCredentialClient{}, scheme, makeLogger())

	stream := &ociv1beta1.Stream{}
	_, err := mgr.GetCrdStatus(stream)
	assert.Error(t, err)
}

func TestDeploymentServiceManager_Delete_NoOcid(t *testing.T) {
	scheme := runtime.NewScheme()
	mgr := NewDeploymentServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeCredentialClient{}, scheme, makeLogger())

	dep := &ociv1beta1.ApiGatewayDeployment{}
	// No OCID — should return true without error
	done, err := mgr.Delete(context.Background(), dep)
	assert.NoError(t, err)
	assert.True(t, done)
}

func TestGatewayServiceManager_Delete_NoOcid(t *testing.T) {
	scheme := runtime.NewScheme()
	mgr := NewGatewayServiceManager(common.NewRawConfigurationProvider("", "", "", "", "", nil), &fakeCredentialClient{}, scheme, makeLogger())

	gw := &ociv1beta1.ApiGateway{}
	// No OCID — should return true without error
	done, err := mgr.Delete(context.Background(), gw)
	assert.NoError(t, err)
	assert.True(t, done)
}
