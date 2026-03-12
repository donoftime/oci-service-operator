/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compute

import (
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const computeInstanceRequeueDuration = 30 * time.Second

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func setCreatedAtIfUnset(status *ociv1beta1.OSOKStatus) {
	if status.CreatedAt != nil {
		return
	}
	now := metav1.NewTime(metav1.Now().Time)
	status.CreatedAt = &now
}

func resolveInstanceID(statusID, specID ociv1beta1.OCID) (ociv1beta1.OCID, error) {
	if statusID != "" {
		return statusID, nil
	}
	if specID != "" {
		return specID, nil
	}
	return "", fmt.Errorf("compute instance ocid is empty")
}

func isNotFoundServiceError(err error) bool {
	serviceErr, ok := err.(common.ServiceError)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func reconcileLifecycleStatus(status *ociv1beta1.OSOKStatus, instance *core.Instance,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	status.Ocid = ociv1beta1.OCID(safeString(instance.Id))

	switch instance.LifecycleState {
	case core.InstanceLifecycleStateRunning,
		core.InstanceLifecycleStateStopped:
		setCreatedAtIfUnset(status)
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("ComputeInstance %s is %s", safeString(instance.DisplayName), instance.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	case core.InstanceLifecycleStateProvisioning,
		core.InstanceLifecycleStateStarting,
		core.InstanceLifecycleStateStopping:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("ComputeInstance %s is %s", safeString(instance.DisplayName), instance.LifecycleState), log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    false,
			ShouldRequeue:   true,
			RequeueDuration: computeInstanceRequeueDuration,
		}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("ComputeInstance %s is %s", safeString(instance.DisplayName), instance.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}
