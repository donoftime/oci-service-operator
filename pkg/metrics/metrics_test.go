/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package metrics

import (
	"context"
	"testing"

	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func testMetrics() *Metrics {
	return &Metrics{
		Name:        defaultMetricsNamespace,
		ServiceName: "test-service",
		Logger:      loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("test")},
	}
}

func TestAddFixedLogMapEntries(t *testing.T) {
	ctx := context.Background()
	result := AddFixedLogMapEntries(ctx, "my-resource", "my-namespace")
	assert.NotNil(t, result)

	val := result.Value(loggerutil.FixedLogMapCtxKey)
	assert.NotNil(t, val)
	m, ok := val.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "my-resource", m["name"])
	assert.Equal(t, "my-namespace", m["namespace"])
}

func TestAddReconcileSuccessMetrics_NoPanic(t *testing.T) {
	m := testMetrics()
	ctx := context.Background()
	assert.NotPanics(t, func() {
		m.AddReconcileSuccessMetrics(ctx, "TestComponent", "reconciled ok", "my-resource", "default")
	})
}

func TestAddReconcileFaultMetrics_NoPanic(t *testing.T) {
	m := testMetrics()
	ctx := context.Background()
	assert.NotPanics(t, func() {
		m.AddReconcileFaultMetrics(ctx, "TestComponent", "reconcile failed", "my-resource", "default")
	})
}

func TestAddCRSuccessMetrics_NoPanic(t *testing.T) {
	m := testMetrics()
	ctx := context.Background()
	assert.NotPanics(t, func() {
		m.AddCRSuccessMetrics(ctx, "TestComponent", "cr created", "my-resource", "default")
	})
}

func TestAddCRFaultMetrics_NoPanic(t *testing.T) {
	m := testMetrics()
	ctx := context.Background()
	assert.NotPanics(t, func() {
		m.AddCRFaultMetrics(ctx, "TestComponent", "cr fault", "my-resource", "default")
	})
}

func TestAddCRDeleteSuccessMetrics_NoPanic(t *testing.T) {
	m := testMetrics()
	ctx := context.Background()
	assert.NotPanics(t, func() {
		m.AddCRDeleteSuccessMetrics(ctx, "TestComponent", "cr deleted", "my-resource", "default")
	})
}

func TestAddCRDeleteFaultMetrics_NoPanic(t *testing.T) {
	m := testMetrics()
	ctx := context.Background()
	assert.NotPanics(t, func() {
		m.AddCRDeleteFaultMetrics(ctx, "TestComponent", "cr delete failed", "my-resource", "default")
	})
}

func TestAddCRCountMetrics_NoPanic(t *testing.T) {
	m := testMetrics()
	ctx := context.Background()
	assert.NotPanics(t, func() {
		m.AddCRCountMetrics(ctx, "TestComponent", "cr counted", "my-resource", "default")
	})
}

func TestAddSecretCountMetrics_NoPanic(t *testing.T) {
	m := testMetrics()
	ctx := context.Background()
	assert.NotPanics(t, func() {
		m.AddSecretCountMetrics(ctx, "TestComponent", "secret counted", "my-resource", "default")
	})
}

func TestMetrics_Fields(t *testing.T) {
	m := testMetrics()
	assert.Equal(t, defaultMetricsNamespace, m.Name)
	assert.Equal(t, "test-service", m.ServiceName)
}
