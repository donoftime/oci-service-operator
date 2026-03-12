/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking

import (
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func resolveResourceID(statusID, specID ociv1beta1.OCID) (ociv1beta1.OCID, error) {
	if statusID != "" {
		return statusID, nil
	}
	if specID != "" {
		return specID, nil
	}
	return "", fmt.Errorf("resource ocid is empty")
}

func isNotFoundServiceError(err error) bool {
	serviceErr, ok := err.(common.ServiceError)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func isPendingLifecycleState(state string) bool {
	return state == "PROVISIONING" || state == "UPDATING"
}

func isReadyLifecycleState(state string) bool {
	return state == "AVAILABLE"
}

func setCreatedAtIfUnset(status *ociv1beta1.OSOKStatus) {
	if status.CreatedAt != nil {
		return
	}
	now := metav1.NewTime(metav1.Now().Time)
	status.CreatedAt = &now
}

func reconcileLifecycleStatus(status *ociv1beta1.OSOKStatus, kind, displayName, lifecycleState string,
	ocid ociv1beta1.OCID, log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	status.Ocid = ocid

	switch {
	case isReadyLifecycleState(lifecycleState):
		setCreatedAtIfUnset(status)
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("%s %s is %s", kind, displayName, lifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	case isPendingLifecycleState(lifecycleState):
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("%s %s is %s", kind, displayName, lifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: false, ShouldRequeue: true}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("%s %s is %s", kind, displayName, lifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}

func deleteResourceAndWait(deleteFn func() error, getFn func() error) (bool, error) {
	if err := deleteFn(); err != nil && !isNotFoundServiceError(err) {
		return false, err
	}

	err := getFn()
	if err == nil {
		return false, nil
	}
	if isNotFoundServiceError(err) {
		return true, nil
	}
	return false, err
}
