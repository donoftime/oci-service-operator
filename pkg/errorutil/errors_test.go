/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceFailureResponses(t *testing.T) {
	testCases := []struct {
		name      string
		code      string
		status    int
		requestID string
		message   string
		check     func(*testing.T, error)
	}{
		{
			name:      "bad request invalid parameter",
			code:      InvalidParameter,
			status:    400,
			requestID: "12-35-67",
			message:   "Invalid Parameter",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Equal(t, "Parameter is invalid or incorrectly formatted", badRequest.Description)
				assert.Equal(t, 400, badRequest.HTTPStatusCode)
			},
		},
		{
			name:      "bad request missing parameter",
			code:      MissingParameter,
			status:    400,
			requestID: "12-35-89",
			message:   "Missing Parameter in the body",
			check: func(t *testing.T, err error) {
				badRequest := assertErrorAs[BadRequestOciError](t, err)
				assert.Equal(t, "Required parameter is missing", badRequest.Description)
				assert.Equal(t, 400, badRequest.HTTPStatusCode)
			},
		},
		{
			name:      "not authenticated",
			code:      NotAuthenticated,
			status:    401,
			requestID: "12-35-89",
			message:   "Not Authenticated to perform operation",
			check: func(t *testing.T, err error) {
				notAuthenticated := assertErrorAs[NotAuthenticatedOciError](t, err)
				assert.Equal(t, "The required authentication information was not provided or was incorrect", notAuthenticated.Description)
				assert.Equal(t, 401, notAuthenticated.HTTPStatusCode)
			},
		},
		{
			name:      "sign up required",
			code:      SignUpRequired,
			status:    402,
			requestID: "12-35-89-09",
			message:   "Sign up required",
			check: func(t *testing.T, err error) {
				signUpRequired := assertErrorAs[SignUpRequiredOciError](t, err)
				assert.Equal(t, "This operation requires opt-in before it may be called", signUpRequired.Description)
				assert.Equal(t, 402, signUpRequired.HTTPStatusCode)
			},
		},
		{
			name:      "not authorized",
			code:      NotAuthorized,
			status:    403,
			requestID: "12-89-87-98",
			message:   "Not Authorized to perform operation",
			check: func(t *testing.T, err error) {
				notFound := assertErrorAs[UnauthorizedAndNotFoundOciError](t, err)
				assert.Equal(t, "You do not have authorization to update one or more of the fields included in this request", notFound.Description)
				assert.Equal(t, 403, notFound.HTTPStatusCode)
			},
		},
		{
			name:      "not found",
			code:      NotFound,
			status:    404,
			requestID: "76-98-57-03",
			message:   "Resource not Found",
			check: func(t *testing.T, err error) {
				notFound := assertErrorAs[UnauthorizedAndNotFoundOciError](t, err)
				assert.Equal(t, "There is no operation supported at the URI path and HTTP method you specified in the request", notFound.Description)
				assert.Equal(t, 404, notFound.HTTPStatusCode)
			},
		},
		{
			name:      "not authorized or not found",
			code:      NotAuthorizedOrNotFound,
			status:    404,
			requestID: "12-35-89-343",
			message:   "Not Authorized to perform action Or Resource Not Found",
			check: func(t *testing.T, err error) {
				notFound := assertErrorAs[UnauthorizedAndNotFoundOciError](t, err)
				assert.Equal(t, "A resource specified via the URI (path or query parameters) of the request was not found, or you do not have authorization to access that resource", notFound.Description)
				assert.Equal(t, 404, notFound.HTTPStatusCode)
			},
		},
		{
			name:      "method not allowed",
			code:      MethodNotAllowed,
			status:    405,
			requestID: "12-35-8324sd9",
			message:   "Method don't Allow http",
			check: func(t *testing.T, err error) {
				notAllowed := assertErrorAs[MethodNotAllowedOciError](t, err)
				assert.Equal(t, "The target resource does not support the HTTP method", notAllowed.Description)
				assert.Equal(t, 405, notAllowed.HTTPStatusCode)
			},
		},
		{
			name:      "incorrect state conflict",
			code:      IncorrectState,
			status:    409,
			requestID: "12-89-23-234",
			message:   "Requested state conflict with present state",
			check: func(t *testing.T, err error) {
				conflict := assertErrorAs[ConflictOciError](t, err)
				assert.Equal(t, "The requested state for the resource conflicts with its current state", conflict.Description)
				assert.Equal(t, 409, conflict.HTTPStatusCode)
			},
		},
		{
			name:      "resource already exists conflict",
			code:      NotAuthorizedOrResourceAlreadyExists,
			status:    409,
			requestID: "12-35-89",
			message:   "Not Authorized Or Resource Already Exists",
			check: func(t *testing.T, err error) {
				conflict := assertErrorAs[ConflictOciError](t, err)
				assert.Equal(t, "You do not have authorization to perform this request, or the resource you are attempting to create already exists", conflict.Description)
				assert.Equal(t, 409, conflict.HTTPStatusCode)
			},
		},
		{
			name:      "no etag match",
			code:      NoEtagMatch,
			status:    412,
			requestID: "12-35-89",
			message:   "No Etag Match",
			check: func(t *testing.T, err error) {
				noEtagMatch := assertErrorAs[NoEtagMatchOciError](t, err)
				assert.Equal(t, "The ETag specified in the request does not match the ETag for the resource", noEtagMatch.Description)
				assert.Equal(t, 412, noEtagMatch.HTTPStatusCode)
			},
		},
		{
			name:      "too many requests",
			code:      TooManyRequests,
			status:    429,
			requestID: "12-35-89",
			message:   "Too Many Requests",
			check: func(t *testing.T, err error) {
				tooManyRequests := assertErrorAs[TooManyRequestsOciError](t, err)
				assert.Equal(t, "You have issued too many requests to the Oracle Cloud Infrastructure APIs in too short of an amount of time", tooManyRequests.Description)
				assert.Equal(t, 429, tooManyRequests.HTTPStatusCode)
			},
		},
		{
			name:      "internal server error",
			code:      InternalServerError,
			status:    500,
			requestID: "12-35-89-93-213",
			message:   "InternalServerError",
			check: func(t *testing.T, err error) {
				internalServerError := assertErrorAs[InternalServerErrorOciError](t, err)
				assert.Equal(t, "An internal server error occurred", internalServerError.Description)
				assert.Equal(t, 500, internalServerError.HTTPStatusCode)
			},
		},
		{
			name:      "method not implemented",
			code:      MethodNotImplemented,
			status:    501,
			requestID: "12-35-89",
			message:   "Not Authenticated to perform operation",
			check: func(t *testing.T, err error) {
				notImplemented := assertErrorAs[MethodNotImplementedOciError](t, err)
				assert.Equal(t, "The HTTP request target does not recognize the HTTP method", notImplemented.Description)
				assert.Equal(t, 501, notImplemented.HTTPStatusCode)
			},
		},
		{
			name:      "service unavailable",
			code:      ServiceUnavailable,
			status:    503,
			requestID: "12-35-89",
			message:   "Service is Unavailable for now",
			check: func(t *testing.T, err error) {
				serviceUnavailable := assertErrorAs[ServiceUnavailableOciError](t, err)
				assert.Equal(t, "The service is currently unavailable", serviceUnavailable.Description)
				assert.Equal(t, 503, serviceUnavailable.HTTPStatusCode)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := NewServiceFailureFromResponse(tc.code, tc.status, tc.requestID, tc.message)
			assert.False(t, resp)
			tc.check(t, err)
		})
	}
}
