/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newTestBaseReconciler() *BaseReconciler {
	return &BaseReconciler{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}
}

func TestRequeueResult_UsesDefaultBackoffWhenDurationMissing(t *testing.T) {
	reconciler := newTestBaseReconciler()

	result, err := reconciler.requeueResult(context.Background(), servicemanager.OSOKResponse{}, nil)
	assert.NoError(t, err)
	assert.False(t, result.Requeue)
	assert.Equal(t, defaultRequeueTime, result.RequeueAfter)
}

func TestRequeueResult_HonorsDurationWithoutError(t *testing.T) {
	reconciler := newTestBaseReconciler()

	result, err := reconciler.requeueResult(context.Background(), servicemanager.OSOKResponse{
		ShouldRequeue:   true,
		RequeueDuration: 30 * time.Second,
	}, nil)
	assert.NoError(t, err)
	assert.False(t, result.Requeue)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

func TestRequeueResult_HonorsDurationWithError(t *testing.T) {
	reconciler := newTestBaseReconciler()

	result, err := reconciler.requeueResult(context.Background(), servicemanager.OSOKResponse{
		ShouldRequeue:   true,
		RequeueDuration: 45 * time.Second,
	}, errors.New("boom"))
	assert.NoError(t, err)
	assert.False(t, result.Requeue)
	assert.Equal(t, 45*time.Second, result.RequeueAfter)
}
