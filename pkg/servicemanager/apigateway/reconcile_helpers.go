/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"errors"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/apigateway"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
)

func apiGatewayBadRequestCode(err error) (string, bool) {
	var badRequest errorutil.BadRequestOciError
	if !errors.As(err, &badRequest) {
		return "", false
	}
	return badRequest.ErrorCode, true
}

func safeGatewayString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func applyGatewayCreateFailure(status *ociv1beta1.OSOKStatus, err error, log loggerutil.OSOKLogger, kind string) {
	*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), log)
	if code, ok := apiGatewayBadRequestCode(err); ok {
		status.Message = code
		log.ErrorLog(err, fmt.Sprintf("Create %s bad request", kind))
		return
	}
	log.ErrorLog(err, fmt.Sprintf("Create %s failed", kind))
}

func setGatewayProvisioning(status *ociv1beta1.OSOKStatus, kind, displayName string, ocid ociv1beta1.OCID,
	log loggerutil.OSOKLogger) {
	status.Ocid = ocid
	*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
		fmt.Sprintf("%s %s Provisioning", kind, displayName), log)
}

func reconcileGatewayLifecycle(status *ociv1beta1.OSOKStatus, instance *apigateway.Gateway,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	displayName := safeGatewayString(instance.DisplayName)
	state := string(instance.LifecycleState)

	switch instance.LifecycleState {
	case apigateway.GatewayLifecycleStateFailed, apigateway.GatewayLifecycleStateDeleted:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ApiGateway %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGateway %s is %s", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false}
	case apigateway.GatewayLifecycleStateActive:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGateway %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGateway %s is Active", displayName))
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGateway %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGateway %s is %s, requeueing", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}
	}
}

func reconcileDeploymentLifecycle(status *ociv1beta1.OSOKStatus, instance *apigateway.Deployment,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	displayName := safeGatewayString(instance.DisplayName)
	state := string(instance.LifecycleState)

	switch instance.LifecycleState {
	case apigateway.DeploymentLifecycleStateFailed, apigateway.DeploymentLifecycleStateDeleted:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ApiGatewayDeployment %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is %s", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false}
	case apigateway.DeploymentLifecycleStateActive:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGatewayDeployment %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is Active", displayName))
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("ApiGatewayDeployment %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("ApiGatewayDeployment %s is %s, requeueing", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}
	}
}
