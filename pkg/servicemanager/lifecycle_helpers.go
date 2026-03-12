/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ResolveResourceID(statusID, specID ociv1beta1.OCID) (ociv1beta1.OCID, error) {
	if statusID != "" {
		return statusID, nil
	}
	if specID != "" {
		return specID, nil
	}
	return "", fmt.Errorf("resource ocid is empty")
}

func SetCreatedAtIfUnset(status *ociv1beta1.OSOKStatus) {
	if status.CreatedAt != nil {
		return
	}
	now := metav1.NewTime(metav1.Now().Time)
	status.CreatedAt = &now
}

func IsNotFoundServiceError(err error) bool {
	serviceErr, ok := err.(common.ServiceError)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func IsNotFoundErrorString(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") || strings.Contains(msg, "notfound") || strings.Contains(msg, "not found")
}

func IsSecretNotFoundError(err error) bool {
	return k8serrors.IsNotFound(err) || IsNotFoundErrorString(err)
}

func containsLifecycleState(target string, states []string) bool {
	for _, state := range states {
		if state == target {
			return true
		}
	}
	return false
}

func ReconcileLifecycleStatus(status *ociv1beta1.OSOKStatus, kind, displayName, lifecycleState string,
	ocid ociv1beta1.OCID, log loggerutil.OSOKLogger, activeStates, retryableStates []string) OSOKResponse {
	status.Ocid = ocid

	switch {
	case containsLifecycleState(lifecycleState, activeStates):
		SetCreatedAtIfUnset(status)
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("%s %s is %s", kind, displayName, lifecycleState), log)
		return OSOKResponse{IsSuccessful: true}
	case containsLifecycleState(lifecycleState, retryableStates):
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("%s %s is %s", kind, displayName, lifecycleState), log)
		return OSOKResponse{IsSuccessful: false, ShouldRequeue: true}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("%s %s is %s", kind, displayName, lifecycleState), log)
		return OSOKResponse{IsSuccessful: false}
	}
}
