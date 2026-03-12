/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import (
	"errors"
	"fmt"

	ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
)

func functionsBadRequestCode(err error) (string, bool) {
	var badRequest errorutil.BadRequestOciError
	if !errors.As(err, &badRequest) {
		return "", false
	}
	return badRequest.ErrorCode, true
}

func safeFunctionsString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func applyFunctionsCreateFailure(status *ociv1beta1.OSOKStatus, err error, log loggerutil.OSOKLogger, kind string) {
	*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), log)
	if code, ok := functionsBadRequestCode(err); ok {
		status.Message = code
		log.ErrorLog(err, fmt.Sprintf("Create %s bad request", kind))
		return
	}
	log.ErrorLog(err, fmt.Sprintf("Create %s failed", kind))
}

func setFunctionsProvisioning(status *ociv1beta1.OSOKStatus, kind, displayName string, ocid ociv1beta1.OCID,
	log loggerutil.OSOKLogger) {
	status.Ocid = ocid
	*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
		fmt.Sprintf("%s %s Provisioning", kind, displayName), log)
}

func reconcileFunctionsApplicationLifecycle(status *ociv1beta1.OSOKStatus, instance *ocifunctions.Application,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	displayName := safeFunctionsString(instance.DisplayName)
	state := string(instance.LifecycleState)

	switch instance.LifecycleState {
	case ocifunctions.ApplicationLifecycleStateFailed, ocifunctions.ApplicationLifecycleStateDeleted:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("FunctionsApplication %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("FunctionsApplication %s is %s", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false}
	case ocifunctions.ApplicationLifecycleStateActive:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("FunctionsApplication %s is %s", displayName, state), log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("FunctionsApplication %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("FunctionsApplication %s is %s, requeueing", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}
	}
}

func reconcileFunctionsFunctionLifecycle(status *ociv1beta1.OSOKStatus, instance *ocifunctions.Function,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	displayName := safeFunctionsString(instance.DisplayName)
	state := string(instance.LifecycleState)

	switch instance.LifecycleState {
	case ocifunctions.FunctionLifecycleStateFailed, ocifunctions.FunctionLifecycleStateDeleted:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("FunctionsFunction %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("FunctionsFunction %s is %s", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false}
	case ocifunctions.FunctionLifecycleStateActive:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("FunctionsFunction %s is %s", displayName, state), log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("FunctionsFunction %s is %s", displayName, state), log)
		log.InfoLog(fmt.Sprintf("FunctionsFunction %s is %s, requeueing", displayName, state))
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}
	}
}
