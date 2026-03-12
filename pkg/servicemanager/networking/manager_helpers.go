/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking

import (
	"fmt"
	"strings"

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

func hasResourceID(id ociv1beta1.OCID) bool {
	return strings.TrimSpace(string(id)) != ""
}

type networkingCreateOrUpdateOps[T any] struct {
	SpecID         ociv1beta1.OCID
	Status         *ociv1beta1.OSOKStatus
	Get            func(ociv1beta1.OCID) (*T, error)
	Update         func() error
	Lookup         func() (*ociv1beta1.OCID, error)
	Create         func() (*T, error)
	OnCreateError  func(error)
	Log            loggerutil.OSOKLogger
	GetExistingMsg string
	GetStatusMsg   string
	GetByOCIDMsg   string
	UpdateMsg      string
}

func reconcileNetworkingResource[T any](ops networkingCreateOrUpdateOps[T]) (*T, error) {
	if hasResourceID(ops.SpecID) {
		return bindSpecifiedNetworkingResource(ops)
	}

	instance, err := resumeManagedNetworkingResource(ops)
	if err != nil || instance != nil {
		return instance, err
	}

	return findOrCreateNetworkingResource(ops)
}

func bindSpecifiedNetworkingResource[T any](ops networkingCreateOrUpdateOps[T]) (*T, error) {
	instance, err := ops.Get(ops.SpecID)
	if err != nil {
		ops.Log.ErrorLog(err, ops.GetExistingMsg)
		return nil, err
	}

	ops.Status.Ocid = ops.SpecID
	if err := ops.Update(); err != nil {
		ops.Log.ErrorLog(err, ops.UpdateMsg)
		return nil, err
	}

	return instance, nil
}

func resumeManagedNetworkingResource[T any](ops networkingCreateOrUpdateOps[T]) (*T, error) {
	if !hasResourceID(ops.Status.Ocid) {
		return nil, nil
	}

	instance, err := ops.Get(ops.Status.Ocid)
	if err != nil {
		if !isNotFoundServiceError(err) {
			ops.Log.ErrorLog(err, ops.GetStatusMsg)
			return nil, err
		}

		ops.Status.Ocid = ""
		return nil, nil
	}

	if err := ops.Update(); err != nil {
		ops.Log.ErrorLog(err, ops.UpdateMsg)
		return nil, err
	}

	return instance, nil
}

func findOrCreateNetworkingResource[T any](ops networkingCreateOrUpdateOps[T]) (*T, error) {
	resourceOCID, err := ops.Lookup()
	if err != nil {
		return nil, err
	}

	if resourceOCID == nil {
		instance, createErr := ops.Create()
		if createErr != nil {
			if ops.OnCreateError != nil {
				ops.OnCreateError(createErr)
			}
			return nil, createErr
		}
		return instance, nil
	}

	instance, err := ops.Get(*resourceOCID)
	if err != nil {
		ops.Log.ErrorLog(err, ops.GetByOCIDMsg)
		return nil, err
	}

	ops.Status.Ocid = *resourceOCID
	if err := ops.Update(); err != nil {
		ops.Log.ErrorLog(err, ops.UpdateMsg)
		return nil, err
	}

	return instance, nil
}

type networkingUpdateOps[Existing any, Details any] struct {
	StatusID             ociv1beta1.OCID
	SpecID               ociv1beta1.OCID
	DesiredCompartmentID ociv1beta1.OCID
	Get                  func(ociv1beta1.OCID) (*Existing, error)
	ExistingCompartment  func(*Existing) *string
	ValidateUnsupported  func(*Existing) error
	ChangeCompartment    func(ociv1beta1.OCID, ociv1beta1.OCID) error
	BuildDetails         func(*Existing) (Details, bool)
	Update               func(ociv1beta1.OCID, Details) error
}

func updateSimpleNetworkingResource[Existing any, Details any](ops networkingUpdateOps[Existing, Details]) error {
	targetID, err := resolveResourceID(ops.StatusID, ops.SpecID)
	if err != nil {
		return err
	}

	existing, err := ops.Get(targetID)
	if err != nil {
		return err
	}

	if ops.ValidateUnsupported != nil {
		if err := ops.ValidateUnsupported(existing); err != nil {
			return err
		}
	}

	if err := changeCompartmentIfNeeded(ops.ExistingCompartment(existing), ops.DesiredCompartmentID,
		func(compartmentID ociv1beta1.OCID) error {
			return ops.ChangeCompartment(targetID, compartmentID)
		}); err != nil {
		return err
	}

	updateDetails, updateNeeded := ops.BuildDetails(existing)
	if !updateNeeded {
		return nil
	}

	return ops.Update(targetID, updateDetails)
}

func changeCompartmentIfNeeded(existingCompartment *string, desiredCompartment ociv1beta1.OCID, changeFn func(ociv1beta1.OCID) error) error {
	if desiredCompartment == "" {
		return nil
	}
	if existingCompartment != nil && *existingCompartment == string(desiredCompartment) {
		return nil
	}
	return changeFn(desiredCompartment)
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
