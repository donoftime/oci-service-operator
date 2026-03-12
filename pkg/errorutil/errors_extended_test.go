/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorStrings(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name: "oci errors",
			err: OciErrors{
				HTTPStatusCode: 500,
				ErrorCode:      InternalServerError,
				OpcRequestID:   "req-123",
				Description:    "something went wrong",
			},
			contains: []string{"something went wrong", "500", "req-123"},
		},
		{
			name:     "bad request",
			err:      BadRequestOciError{HTTPStatusCode: 400, Description: "bad", OpcRequestID: "r1"},
			contains: []string{"bad", "400"},
		},
		{
			name:     "not authenticated",
			err:      NotAuthenticatedOciError{HTTPStatusCode: 401, Description: "not authed", OpcRequestID: "r2"},
			contains: []string{"not authed", "401"},
		},
		{
			name:     "sign up required",
			err:      SignUpRequiredOciError{HTTPStatusCode: 402, Description: "sign up", OpcRequestID: "r3"},
			contains: []string{"sign up", "402"},
		},
		{
			name:     "unauthorized and not found",
			err:      UnauthorizedAndNotFoundOciError{HTTPStatusCode: 403, Description: "not found", OpcRequestID: "r4"},
			contains: []string{"not found", "403"},
		},
		{
			name:     "method not allowed",
			err:      MethodNotAllowedOciError{HTTPStatusCode: 405, Description: "not allowed", OpcRequestID: "r5"},
			contains: []string{"not allowed", "405"},
		},
		{
			name:     "conflict",
			err:      ConflictOciError{HTTPStatusCode: 409, Description: "conflict", OpcRequestID: "r6"},
			contains: []string{"conflict", "409"},
		},
		{
			name:     "no etag match",
			err:      NoEtagMatchOciError{HTTPStatusCode: 412, Description: "no etag", OpcRequestID: "r7"},
			contains: []string{"no etag", "412"},
		},
		{
			name:     "too many requests",
			err:      TooManyRequestsOciError{HTTPStatusCode: 429, Description: "too many", OpcRequestID: "r8"},
			contains: []string{"too many", "429"},
		},
		{
			name:     "internal server error",
			err:      InternalServerErrorOciError{HTTPStatusCode: 500, Description: "internal", OpcRequestID: "r9"},
			contains: []string{"internal", "500"},
		},
		{
			name:     "method not implemented",
			err:      MethodNotImplementedOciError{HTTPStatusCode: 501, Description: "not impl", OpcRequestID: "r10"},
			contains: []string{"not impl", "501"},
		},
		{
			name:     "service unavailable",
			err:      ServiceUnavailableOciError{HTTPStatusCode: 503, Description: "unavailable", OpcRequestID: "r11"},
			contains: []string{"unavailable", "503"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := tc.err.Error()
			for _, expected := range tc.contains {
				assert.True(t, strings.Contains(msg, expected))
			}
		})
	}
}

func TestSpecificErrorMappings(t *testing.T) {
	testCases := []struct {
		name      string
		code      string
		status    int
		requestID string
		message   string
		check     func(*testing.T, error)
	}{
		{
			name:      "default 400",
			code:      "UnknownCode400",
			status:    400,
			requestID: "req-400",
			message:   "unknown 400",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Equal(t, "Bad Request", badRequest.Description)
			},
		},
		{
			name:      "cannot parse request",
			code:      CannotParseRequest,
			status:    400,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Equal(t, "The request is incorrectly formatted", badRequest.Description)
			},
		},
		{
			name:      "limit exceeded",
			code:      LimitExceeded,
			status:    400,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Contains(t, badRequest.Description, "limit")
			},
		},
		{
			name:      "quota exceeded",
			code:      QuotaExceeded,
			status:    400,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Contains(t, badRequest.Description, "quota")
			},
		},
		{
			name:      "related resource not authorized or not found",
			code:      RelatedResourceNotAuthorizedOrNotFound,
			status:    400,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Contains(t, badRequest.Description, "not found")
			},
		},
		{
			name:      "invalid parameters",
			code:      InvalidParameters,
			status:    400,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Contains(t, badRequest.Description, "invalid")
			},
		},
		{
			name:      "missing parameters",
			code:      MissingParameters,
			status:    400,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Contains(t, badRequest.Description, "missing")
			},
		},
		{
			name:      "unknown 401",
			code:      "SomeOtherCode",
			status:    401,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[NotAuthenticatedOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "unknown 402",
			code:      "Unknown402",
			status:    402,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[SignUpRequiredOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "unknown 403",
			code:      "Unknown403",
			status:    403,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[UnauthorizedAndNotFoundOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "unknown 404",
			code:      "Unknown404",
			status:    404,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[UnauthorizedAndNotFoundOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "unknown 405",
			code:      "Unknown405",
			status:    405,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[MethodNotAllowedOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "default 406",
			code:      "SomeCode",
			status:    406,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "406")
			},
		},
		{
			name:      "invalidated retry token",
			code:      InvalidatedRetryToken,
			status:    409,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				conflict := assertErrorAs[ConflictOciError](t, err)
				assert.Contains(t, conflict.Description, "retry token")
			},
		},
		{
			name:      "unknown 409",
			code:      "Unknown409",
			status:    409,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[ConflictOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "unknown 412",
			code:      "Unknown412",
			status:    412,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[NoEtagMatchOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "unknown 429",
			code:      "Unknown429",
			status:    429,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[TooManyRequestsOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "default 502",
			code:      "SomeCode",
			status:    502,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "502")
			},
		},
		{
			name:      "unknown 500",
			code:      "Unknown500",
			status:    500,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[InternalServerErrorOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "unknown 501",
			code:      "Unknown501",
			status:    501,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[MethodNotImplementedOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "unknown 503",
			code:      "Unknown503",
			status:    503,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assertNotErrorAs[ServiceUnavailableOciError](t, err)
				assert.NotNil(t, err)
			},
		},
		{
			name:      "status outside range",
			code:      "SomeCode",
			status:    200,
			requestID: "req",
			message:   "msg",
			check: func(t *testing.T, err error) {
				assert.Nil(t, err)
			},
		},
		{
			name:      "raw oci error string",
			code:      "WeirdCode",
			status:    406,
			requestID: "opcid-abc",
			message:   "some message",
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				msg := err.Error()
				assert.Contains(t, msg, "WeirdCode")
				assert.Contains(t, msg, "406")
				assert.Contains(t, msg, "opcid-abc")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := NewServiceFailureFromResponse(tc.code, tc.status, tc.requestID, tc.message)
			if tc.status >= 400 {
				assert.False(t, resp)
			} else {
				assert.True(t, resp)
			}
			tc.check(t, err)
		})
	}
}
