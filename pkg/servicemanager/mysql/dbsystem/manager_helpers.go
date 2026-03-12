/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const mysqlRequeueDuration = 30 * time.Second

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func isNotFoundServiceError(err error) bool {
	serviceErr, ok := err.(common.ServiceError)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func isRetryableReadServiceError(err error) bool {
	serviceErr, ok := common.IsServiceError(err)
	if !ok {
		return false
	}

	statusCode := serviceErr.GetHTTPStatusCode()
	return statusCode == 429 || statusCode >= 500
}

func setCreatedAtIfUnset(status *ociv1beta1.OSOKStatus) {
	if status.CreatedAt != nil {
		return
	}
	now := metav1.NewTime(metav1.Now().Time)
	status.CreatedAt = &now
}

func reconcileLifecycleStatus(status *ociv1beta1.OSOKStatus, dbSystem *mysql.DbSystem,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	status.Ocid = ociv1beta1.OCID(safeString(dbSystem.Id))

	switch dbSystem.LifecycleState {
	case mysql.DbSystemLifecycleStateActive:
		setCreatedAtIfUnset(status)
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("MySqlDbSystem %s is %s", safeString(dbSystem.DisplayName), dbSystem.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	case mysql.DbSystemLifecycleStateCreating,
		mysql.DbSystemLifecycleStateUpdating,
		mysql.DbSystemLifecycleStateInactive:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("MySqlDbSystem %s is %s", safeString(dbSystem.DisplayName), dbSystem.LifecycleState), log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    false,
			ShouldRequeue:   true,
			RequeueDuration: mysqlRequeueDuration,
		}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("MySqlDbSystem %s is %s", safeString(dbSystem.DisplayName), dbSystem.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}

func requeueForTransientReadFailure(status *ociv1beta1.OSOKStatus, resourceID ociv1beta1.OCID, displayName string,
	operation string, err error, log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	if resourceID != "" {
		status.Ocid = resourceID
	}

	*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
		fmt.Sprintf("Transient MySqlDbSystem read failure while %s for %s: %s", operation, displayName, err.Error()), log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    false,
		ShouldRequeue:   true,
		RequeueDuration: mysqlRequeueDuration,
	}
}
