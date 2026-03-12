/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dataflow

import (
	"fmt"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocidataflow "github.com/oracle/oci-go-sdk/v65/dataflow"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
)

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
	updateNeeded := applyDataFlowBasicUpdates(&updateDetails, app, existing)
	updateNeeded = applyDataFlowExecutorUpdates(&updateDetails, app, existing) || updateNeeded
	updateNeeded = applyDataFlowArtifactUpdates(&updateDetails, app, existing) || updateNeeded
	updateNeeded = applyDataFlowTagUpdates(&updateDetails, app, existing) || updateNeeded

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

func applyDataFlowBasicUpdates(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	updateNeeded := applyDataFlowDisplayNameUpdate(updateDetails, app, existing)
	if applyDataFlowDescriptionUpdate(updateDetails, app, existing) {
		updateNeeded = true
	}
	if applyDataFlowSparkVersionUpdate(updateDetails, app, existing) {
		updateNeeded = true
	}
	if applyDataFlowLanguageUpdate(updateDetails, app, existing) {
		updateNeeded = true
	}
	if applyDataFlowLogsBucketUpdate(updateDetails, app, existing) {
		updateNeeded = true
	}
	if applyDataFlowWarehouseBucketUpdate(updateDetails, app, existing) {
		updateNeeded = true
	}
	return updateNeeded
}

func applyDataFlowDisplayNameUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if app.Spec.DisplayName == "" || safeString(existing.DisplayName) == app.Spec.DisplayName {
		return false
	}
	updateDetails.DisplayName = common.String(app.Spec.DisplayName)
	return true
}

func applyDataFlowDescriptionUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if app.Spec.Description == "" || safeString(existing.Description) == app.Spec.Description {
		return false
	}
	updateDetails.Description = common.String(app.Spec.Description)
	return true
}

func applyDataFlowSparkVersionUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if app.Spec.SparkVersion == "" || safeString(existing.SparkVersion) == app.Spec.SparkVersion {
		return false
	}
	updateDetails.SparkVersion = common.String(app.Spec.SparkVersion)
	return true
}

func applyDataFlowLanguageUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if app.Spec.Language == "" || string(existing.Language) == app.Spec.Language {
		return false
	}
	updateDetails.Language = ocidataflow.ApplicationLanguageEnum(app.Spec.Language)
	return true
}

func applyDataFlowLogsBucketUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if app.Spec.LogsBucketUri == "" || safeString(existing.LogsBucketUri) == app.Spec.LogsBucketUri {
		return false
	}
	updateDetails.LogsBucketUri = common.String(app.Spec.LogsBucketUri)
	return true
}

func applyDataFlowWarehouseBucketUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if app.Spec.WarehouseBucketUri == "" || safeString(existing.WarehouseBucketUri) == app.Spec.WarehouseBucketUri {
		return false
	}
	updateDetails.WarehouseBucketUri = common.String(app.Spec.WarehouseBucketUri)
	return true
}

func applyDataFlowExecutorUpdates(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	updateNeeded := applyDataFlowExecutorCountUpdate(updateDetails, app, existing)
	updateNeeded = applyDataFlowConfigurationUpdate(updateDetails, app, existing) || updateNeeded
	updateNeeded = applyDataFlowArgumentsUpdate(updateDetails, app, existing) || updateNeeded
	updateNeeded = applyDataFlowShapeUpdates(updateDetails, app, existing) || updateNeeded
	return updateNeeded
}

func applyDataFlowArtifactUpdates(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	updateNeeded := false
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
	return updateNeeded
}

func applyDataFlowExecutorCountUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if app.Spec.NumExecutors <= 0 || (existing.NumExecutors != nil && *existing.NumExecutors == app.Spec.NumExecutors) {
		return false
	}

	updateDetails.NumExecutors = common.Int(app.Spec.NumExecutors)
	return true
}

func applyDataFlowConfigurationUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if app.Spec.Configuration == nil || mapStringEquals(existing.Configuration, app.Spec.Configuration) {
		return false
	}

	updateDetails.Configuration = app.Spec.Configuration
	return true
}

func applyDataFlowArgumentsUpdate(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	if len(app.Spec.Arguments) == 0 || sliceEquals(existing.Arguments, app.Spec.Arguments) {
		return false
	}

	updateDetails.Arguments = app.Spec.Arguments
	return true
}

func applyDataFlowShapeUpdates(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	updateNeeded := false
	if app.Spec.DriverShape != "" && safeString(existing.DriverShape) != app.Spec.DriverShape {
		updateDetails.DriverShape = common.String(app.Spec.DriverShape)
		updateNeeded = true
	}
	if app.Spec.ExecutorShape != "" && safeString(existing.ExecutorShape) != app.Spec.ExecutorShape {
		updateDetails.ExecutorShape = common.String(app.Spec.ExecutorShape)
		updateNeeded = true
	}
	return updateNeeded
}

func applyDataFlowTagUpdates(updateDetails *ocidataflow.UpdateApplicationDetails,
	app *ociv1beta1.DataFlowApplication, existing *ocidataflow.Application) bool {
	updateNeeded := false
	if app.Spec.FreeFormTags != nil && !mapStringEquals(existing.FreeformTags, app.Spec.FreeFormTags) {
		updateDetails.FreeformTags = app.Spec.FreeFormTags
		updateNeeded = true
	}
	if app.Spec.DefinedTags != nil {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&app.Spec.DefinedTags)
		if !reflect.DeepEqual(existing.DefinedTags, desiredDefinedTags) {
			updateDetails.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	return updateNeeded
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
