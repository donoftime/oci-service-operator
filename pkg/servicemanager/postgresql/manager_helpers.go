/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package postgresql

import (
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/psql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const postgresRequeueDuration = 30 * time.Second

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

func resolveDbSystemID(statusID, specID ociv1beta1.OCID) (ociv1beta1.OCID, error) {
	if statusID != "" {
		return statusID, nil
	}
	if specID != "" {
		return specID, nil
	}
	return "", fmt.Errorf("postgres db system ocid is empty")
}

func isNotFoundServiceError(err error) bool {
	serviceErr, ok := err.(common.ServiceError)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func reconcileLifecycleStatus(status *ociv1beta1.OSOKStatus, dbSystem *psql.DbSystem,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	status.Ocid = ociv1beta1.OCID(safeString(dbSystem.Id))

	switch dbSystem.LifecycleState {
	case psql.DbSystemLifecycleStateActive:
		setCreatedAtIfUnset(status)
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("PostgresDbSystem %s is %s", safeString(dbSystem.DisplayName), dbSystem.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	case psql.DbSystemLifecycleStateCreating,
		psql.DbSystemLifecycleStateUpdating:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("PostgresDbSystem %s is %s", safeString(dbSystem.DisplayName), dbSystem.LifecycleState), log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    false,
			ShouldRequeue:   true,
			RequeueDuration: postgresRequeueDuration,
		}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("PostgresDbSystem %s is %s", safeString(dbSystem.DisplayName), dbSystem.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}
