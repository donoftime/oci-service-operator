/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertErrorAs[T error](t *testing.T, err error) T {
	t.Helper()

	var target T
	if !assert.ErrorAs(t, err, &target) {
		t.FailNow()
	}

	return target
}

func assertNotErrorAs[T error](t *testing.T, err error) {
	t.Helper()

	var target T
	assert.False(t, errors.As(err, &target))
}
