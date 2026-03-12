/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway_test

import (
	"context"
	"fmt"
	"testing"
	"testing/quick"

	"github.com/oracle/oci-go-sdk/v65/apigateway"
	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestGatewayServiceManager_PropertyRetryableStatesRequeue(t *testing.T) {
	states := []apigateway.GatewayLifecycleStateEnum{
		apigateway.GatewayLifecycleStateCreating,
		apigateway.GatewayLifecycleStateUpdating,
		apigateway.GatewayLifecycleStateDeleting,
	}

	property := func(seed uint8) bool {
		state := states[int(seed)%len(states)]
		gatewayID := fmt.Sprintf("ocid1.apigateway.oc1..%d", seed)
		credClient := &fakeCredentialClient{}
		gwClient := &mockGatewayClient{
			getGatewayFn: func(_ context.Context, _ apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error) {
				return apigateway.GetGatewayResponse{
					Gateway: apigateway.Gateway{
						Id:             common.String(gatewayID),
						DisplayName:    common.String("prop-gateway"),
						LifecycleState: state,
					},
				}, nil
			},
		}

		mgr := makeGatewayManager(gwClient, credClient)
		obj := &ociv1beta1.ApiGateway{}
		obj.Name = "prop-gateway"
		obj.Namespace = "default"
		obj.Spec.ApiGatewayId = ociv1beta1.OCID(gatewayID)
		obj.Spec.DisplayName = "prop-gateway"

		resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
		return err == nil && !resp.IsSuccessful && resp.ShouldRequeue && !credClient.createCalled
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDeploymentServiceManager_PropertyRetryableStatesRequeue(t *testing.T) {
	states := []apigateway.DeploymentLifecycleStateEnum{
		apigateway.DeploymentLifecycleStateCreating,
		apigateway.DeploymentLifecycleStateUpdating,
		apigateway.DeploymentLifecycleStateDeleting,
	}

	property := func(seed uint8) bool {
		state := states[int(seed)%len(states)]
		deploymentID := fmt.Sprintf("ocid1.apideployment.oc1..%d", seed)
		depClient := &mockDeploymentClient{
			getDeploymentFn: func(_ context.Context, _ apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
				return apigateway.GetDeploymentResponse{
					Deployment: apigateway.Deployment{
						Id:             common.String(deploymentID),
						DisplayName:    common.String("prop-deployment"),
						LifecycleState: state,
					},
				}, nil
			},
		}

		mgr := makeDeploymentManager(depClient, &fakeCredentialClient{})
		obj := &ociv1beta1.ApiGatewayDeployment{}
		obj.Name = "prop-deployment"
		obj.Namespace = "default"
		obj.Spec.DeploymentId = ociv1beta1.OCID(deploymentID)
		obj.Spec.DisplayName = "prop-deployment"

		resp, err := mgr.CreateOrUpdate(context.Background(), obj, ctrl.Request{})
		return err == nil && !resp.IsSuccessful && resp.ShouldRequeue
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGatewayServiceManager_PropertyDeleteFallsBackToSpecID(t *testing.T) {
	property := func(seed uint16) bool {
		gatewayID := fmt.Sprintf("ocid1.apigateway.oc1..delete-%d", seed)
		var deletedID string
		gwClient := &mockGatewayClient{
			deleteGatewayFn: func(_ context.Context, req apigateway.DeleteGatewayRequest) (apigateway.DeleteGatewayResponse, error) {
				deletedID = *req.GatewayId
				return apigateway.DeleteGatewayResponse{}, nil
			},
			getGatewayFn: func(_ context.Context, req apigateway.GetGatewayRequest) (apigateway.GetGatewayResponse, error) {
				return apigateway.GetGatewayResponse{
					Gateway: apigateway.Gateway{
						Id:             req.GatewayId,
						DisplayName:    common.String("prop-gateway"),
						LifecycleState: apigateway.GatewayLifecycleStateDeleted,
					},
				}, nil
			},
		}

		mgr := makeGatewayManager(gwClient, &fakeCredentialClient{})
		obj := &ociv1beta1.ApiGateway{}
		obj.Spec.ApiGatewayId = ociv1beta1.OCID(gatewayID)

		done, err := mgr.Delete(context.Background(), obj)
		return err == nil && done && deletedID == gatewayID
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDeploymentServiceManager_PropertyDeleteFallsBackToSpecID(t *testing.T) {
	property := func(seed uint16) bool {
		deploymentID := fmt.Sprintf("ocid1.apideployment.oc1..delete-%d", seed)
		var deletedID string
		depClient := &mockDeploymentClient{
			deleteDeploymentFn: func(_ context.Context, req apigateway.DeleteDeploymentRequest) (apigateway.DeleteDeploymentResponse, error) {
				deletedID = *req.DeploymentId
				return apigateway.DeleteDeploymentResponse{}, nil
			},
			getDeploymentFn: func(_ context.Context, req apigateway.GetDeploymentRequest) (apigateway.GetDeploymentResponse, error) {
				return apigateway.GetDeploymentResponse{
					Deployment: apigateway.Deployment{
						Id:             req.DeploymentId,
						DisplayName:    common.String("prop-deployment"),
						LifecycleState: apigateway.DeploymentLifecycleStateDeleted,
					},
				}, nil
			},
		}

		mgr := makeDeploymentManager(depClient, &fakeCredentialClient{})
		obj := &ociv1beta1.ApiGatewayDeployment{}
		obj.Spec.DeploymentId = ociv1beta1.OCID(deploymentID)

		done, err := mgr.Delete(context.Background(), obj)
		return err == nil && done && deletedID == deploymentID
	}

	if err := quick.Check(property, nil); err != nil {
		t.Fatal(err)
	}
}
