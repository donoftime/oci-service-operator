/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package util

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestLogUtil_LogInfo_NoPanic(t *testing.T) {
	lu := &LogUtil{Log: ctrl.Log.WithName("test")}
	assert.NotPanics(t, func() {
		lu.LogInfo("info message")
		lu.LogInfo("info with args", "key", "value")
	})
}

func TestLogUtil_LogDebug_NoPanic(t *testing.T) {
	lu := &LogUtil{Log: ctrl.Log.WithName("test")}
	assert.NotPanics(t, func() {
		lu.LogDebug("debug message")
		lu.LogDebug("debug with args", "key", "value")
	})
}

func TestLogUtil_LogError_NoPanic(t *testing.T) {
	lu := &LogUtil{Log: ctrl.Log.WithName("test")}
	err := errors.New("test error")
	assert.NotPanics(t, func() {
		lu.LogError(err, "error message")
		lu.LogError(err, "error with args", "key", "value")
	})
}

func TestLogUtil_LogInfo_NilArgs(t *testing.T) {
	lu := &LogUtil{Log: ctrl.Log.WithName("test")}
	assert.NotPanics(t, func() {
		lu.LogInfo("msg with nil keysAndValues")
	})
}
