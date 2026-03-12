/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package util

import (
	"archive/zip"
	"context"
	"github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

func RequeueWithError(ctx context.Context, err error, duration time.Duration, log loggerutil.OSOKLogger) (ctrl.Result, error) {
	log.InfoLogWithFixedMessage(ctx, "requeue after", "error", err.Error(), "duration", duration.String())
	return ctrl.Result{RequeueAfter: duration}, nil
}

func RequeueWithoutError(ctx context.Context, duration time.Duration, log loggerutil.OSOKLogger) (ctrl.Result, error) {
	log.InfoLogWithFixedMessage(ctx, "requeue after", "duration", duration.String())
	return ctrl.Result{RequeueAfter: duration}, nil
}

func DoNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func GetOSOKStatusCondition(status v1beta1.OSOKStatus, conditionType v1beta1.OSOKConditionType, log loggerutil.OSOKLogger) *v1beta1.OSOKCondition {
	for cnt := range status.Conditions {
		if status.Conditions[cnt].Type == conditionType {
			return &status.Conditions[cnt]
		}
	}
	return nil
}

func getOSOKStatusConditionIndex(status v1beta1.OSOKStatus, conditionType v1beta1.OSOKConditionType) int {
	for cnt := range status.Conditions {
		if status.Conditions[cnt].Type == conditionType {
			return cnt
		}
	}
	return -1
}

func UpdateOSOKStatusCondition(osokStatus v1beta1.OSOKStatus, conditionType v1beta1.OSOKConditionType,
	status v1.ConditionStatus, reason string, message string, log loggerutil.OSOKLogger) v1beta1.OSOKStatus {
	currentTime := metav1.Now()

	existingConditionIndex := getOSOKStatusConditionIndex(osokStatus, conditionType)
	if existingConditionIndex == -1 {
		condition := v1beta1.OSOKCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: &currentTime,
			Message:            message,
			Reason:             reason,
		}
		osokStatus.Conditions = append(osokStatus.Conditions, condition)
		return osokStatus
	}

	existingCondition := osokStatus.Conditions[existingConditionIndex]
	if existingCondition.Status == status && existingCondition.Reason == reason && existingCondition.Message == message {
		return osokStatus
	}

	osokStatus.Conditions[existingConditionIndex] = v1beta1.OSOKCondition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: &currentTime,
		Message:            message,
		Reason:             reason,
	}
	return osokStatus
}

func UnzipWallet(filename string) (map[string][]byte, error) {
	data := map[string][]byte{}

	reader, err := zip.OpenReader(filename)
	if err != nil {
		return data, err
	}

	defer reader.Close()
	for _, file := range reader.File {
		reader, err := file.Open()
		if err != nil {
			return data, err
		}

		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return data, err
		}

		data[file.Name] = content
	}

	return data, nil
}

func ConvertToOciDefinedTags(osokDef *map[string]v1beta1.MapValue) *map[string]map[string]interface{} {
	ociDefTags := make(map[string]map[string]interface{})

	for outKey, outVal := range *osokDef {
		inMap := make(map[string]interface{})
		for inKey, inVal := range outVal {
			inMap[inKey] = inVal
		}
		ociDefTags[outKey] = inMap
	}

	return &ociDefTags
}
