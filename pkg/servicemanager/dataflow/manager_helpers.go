/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dataflow

import (
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocidataflow "github.com/oracle/oci-go-sdk/v65/dataflow"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
)

const dataFlowRequeueDuration = 30 * time.Second

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func resolveApplicationID(statusID, specID ociv1beta1.OCID) (ociv1beta1.OCID, error) {
	if statusID != "" {
		return statusID, nil
	}
	if specID != "" {
		return specID, nil
	}
	return "", fmt.Errorf("dataflow application ocid is empty")
}

func isNotFoundServiceError(err error) bool {
	serviceErr, ok := err.(common.ServiceError)
	return ok && serviceErr.GetHTTPStatusCode() == 404
}

func dataFlowUpdateNeeded(app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) (ocidataflow.UpdateApplicationDetails, bool) {
	updateDetails := ocidataflow.UpdateApplicationDetails{}
	updateNeeded := false

	if app.Spec.DisplayName != "" && safeString(existing.DisplayName) != app.Spec.DisplayName {
		updateDetails.DisplayName = common.String(app.Spec.DisplayName)
		updateNeeded = true
	}
	if app.Spec.Description != "" && safeString(existing.Description) != app.Spec.Description {
		updateDetails.Description = common.String(app.Spec.Description)
		updateNeeded = true
	}
	if app.Spec.NumExecutors > 0 && (existing.NumExecutors == nil || *existing.NumExecutors != app.Spec.NumExecutors) {
		updateDetails.NumExecutors = common.Int(app.Spec.NumExecutors)
		updateNeeded = true
	}
	if app.Spec.Configuration != nil && !mapStringEquals(existing.Configuration, app.Spec.Configuration) {
		updateDetails.Configuration = app.Spec.Configuration
		updateNeeded = true
	}
	if len(app.Spec.Arguments) > 0 && !sliceEquals(existing.Arguments, app.Spec.Arguments) {
		updateDetails.Arguments = app.Spec.Arguments
		updateNeeded = true
	}
	if app.Spec.SparkVersion != "" && safeString(existing.SparkVersion) != app.Spec.SparkVersion {
		updateDetails.SparkVersion = common.String(app.Spec.SparkVersion)
		updateNeeded = true
	}
	if app.Spec.DriverShape != "" && safeString(existing.DriverShape) != app.Spec.DriverShape {
		updateDetails.DriverShape = common.String(app.Spec.DriverShape)
		updateNeeded = true
	}
	if app.Spec.ExecutorShape != "" && safeString(existing.ExecutorShape) != app.Spec.ExecutorShape {
		updateDetails.ExecutorShape = common.String(app.Spec.ExecutorShape)
		updateNeeded = true
	}
	if app.Spec.FileUri != "" && safeString(existing.FileUri) != app.Spec.FileUri {
		updateDetails.FileUri = common.String(app.Spec.FileUri)
		updateNeeded = true
	}
	if app.Spec.ClassName != "" && safeString(existing.ClassName) != app.Spec.ClassName {
		updateDetails.ClassName = common.String(app.Spec.ClassName)
		updateNeeded = true
	}
	if app.Spec.ArchiveUri != "" && safeString(existing.ArchiveUri) != app.Spec.ArchiveUri {
		updateDetails.ArchiveUri = common.String(app.Spec.ArchiveUri)
		updateNeeded = true
	}

	return updateDetails, updateNeeded
}

func mapStringEquals(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for k, v := range left {
		if right[k] != v {
			return false
		}
	}
	return true
}

func sliceEquals(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func markDeletedStatus(app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application, log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
		ociv1beta1.Failed, v1.ConditionFalse, "",
		fmt.Sprintf("DataFlowApplication %s has been deleted externally", safeString(existing.DisplayName)), log)
	return servicemanager.OSOKResponse{IsSuccessful: false}
}

func reconcileLifecycleStatus(app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	app.Status.OsokStatus.Ocid = ociv1beta1.OCID(safeString(existing.Id))

	switch existing.LifecycleState {
	case ocidataflow.ApplicationLifecycleStateActive,
		ocidataflow.ApplicationLifecycleStateInactive:
		servicemanager.SetCreatedAtIfUnset(&app.Status.OsokStatus)
		app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("DataFlowApplication %s is %s", safeString(existing.DisplayName), existing.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	default:
		app.Status.OsokStatus = util.UpdateOSOKStatusCondition(app.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("DataFlowApplication %s is %s", safeString(existing.DisplayName), existing.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}
