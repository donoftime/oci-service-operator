/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"fmt"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const adbRequeueDuration = 30 * time.Second

const (
	autonomousDatabaseKindName = "AutonomousDatabases"
)

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

func setCreatedAtIfUnset(status *ociv1beta1.OSOKStatus) {
	if status.CreatedAt != nil {
		return
	}
	now := metav1.NewTime(metav1.Now().Time)
	status.CreatedAt = &now
}

func walletSecretName(adb *ociv1beta1.AutonomousDatabases) string {
	if adb.Spec.Wallet.WalletName != "" {
		return adb.Spec.Wallet.WalletName
	}
	if adb.Name == "" {
		return ""
	}
	return fmt.Sprintf("%s-wallet", adb.Name)
}

func shouldUpdateOptionalBool(hasDesired bool, desired bool, existing *bool) bool {
	return hasDesired && (existing == nil || desired != *existing)
}

func reconcileLifecycleStatus(status *ociv1beta1.OSOKStatus, adbInstance *database.AutonomousDatabase,
	log loggerutil.OSOKLogger) servicemanager.OSOKResponse {
	status.Ocid = ociv1beta1.OCID(safeString(adbInstance.Id))

	switch adbInstance.LifecycleState {
	case database.AutonomousDatabaseLifecycleStateAvailable,
		database.AutonomousDatabaseLifecycleStateAvailableNeedsAttention:
		setCreatedAtIfUnset(status)
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("AutonomousDatabase %s is %s", safeString(adbInstance.DisplayName), adbInstance.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: true}
	case database.AutonomousDatabaseLifecycleStateProvisioning,
		database.AutonomousDatabaseLifecycleStateUpdating,
		database.AutonomousDatabaseLifecycleStateStarting,
		database.AutonomousDatabaseLifecycleStateStopping,
		database.AutonomousDatabaseLifecycleStateMaintenanceInProgress,
		database.AutonomousDatabaseLifecycleStateRestarting,
		database.AutonomousDatabaseLifecycleStateScaleInProgress,
		database.AutonomousDatabaseLifecycleStateBackupInProgress,
		database.AutonomousDatabaseLifecycleStateRestoreInProgress:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Provisioning, v1.ConditionTrue, "",
			fmt.Sprintf("AutonomousDatabase %s is %s", safeString(adbInstance.DisplayName), adbInstance.LifecycleState), log)
		return servicemanager.OSOKResponse{
			IsSuccessful:    false,
			ShouldRequeue:   true,
			RequeueDuration: adbRequeueDuration,
		}
	default:
		*status = util.UpdateOSOKStatusCondition(*status, ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("AutonomousDatabase %s is %s", safeString(adbInstance.DisplayName), adbInstance.LifecycleState), log)
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}
}
