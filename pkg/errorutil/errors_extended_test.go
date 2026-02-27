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

// ---------------------------------------------------------------------------
// Error() method tests for all exported error types
// ---------------------------------------------------------------------------

func TestOciErrors_Error(t *testing.T) {
	e := OciErrors{
		HTTPStatusCode: 500,
		ErrorCode:      "InternalServerError",
		OpcRequestID:   "req-123",
		Description:    "something went wrong",
	}
	msg := e.Error()
	assert.True(t, strings.Contains(msg, "something went wrong"))
	assert.True(t, strings.Contains(msg, "500"))
	assert.True(t, strings.Contains(msg, "req-123"))
}

func TestBadRequestOciError_Error(t *testing.T) {
	e := BadRequestOciError{HTTPStatusCode: 400, Description: "bad", OpcRequestID: "r1"}
	assert.Contains(t, e.Error(), "bad")
	assert.Contains(t, e.Error(), "400")
}

func TestNotAuthenticatedOciError_Error(t *testing.T) {
	e := NotAuthenticatedOciError{HTTPStatusCode: 401, Description: "not authed", OpcRequestID: "r2"}
	assert.Contains(t, e.Error(), "not authed")
	assert.Contains(t, e.Error(), "401")
}

func TestSignUpRequiredOciError_Error(t *testing.T) {
	e := SignUpRequiredOciError{HTTPStatusCode: 402, Description: "sign up", OpcRequestID: "r3"}
	assert.Contains(t, e.Error(), "sign up")
	assert.Contains(t, e.Error(), "402")
}

func TestUnauthorizedAndNotFoundOciError_Error(t *testing.T) {
	e := UnauthorizedAndNotFoundOciError{HTTPStatusCode: 403, Description: "not found", OpcRequestID: "r4"}
	assert.Contains(t, e.Error(), "not found")
	assert.Contains(t, e.Error(), "403")
}

func TestMethodNotAllowedOciError_Error(t *testing.T) {
	e := MethodNotAllowedOciError{HTTPStatusCode: 405, Description: "not allowed", OpcRequestID: "r5"}
	assert.Contains(t, e.Error(), "not allowed")
	assert.Contains(t, e.Error(), "405")
}

func TestConflictOciError_Error(t *testing.T) {
	e := ConflictOciError{HTTPStatusCode: 409, Description: "conflict", OpcRequestID: "r6"}
	assert.Contains(t, e.Error(), "conflict")
	assert.Contains(t, e.Error(), "409")
}

func TestNoEtagMatchOciError_Error(t *testing.T) {
	e := NoEtagMatchOciError{HTTPStatusCode: 412, Description: "no etag", OpcRequestID: "r7"}
	assert.Contains(t, e.Error(), "no etag")
	assert.Contains(t, e.Error(), "412")
}

func TestTooManyRequestsOciError_Error(t *testing.T) {
	e := TooManyRequestsOciError{HTTPStatusCode: 429, Description: "too many", OpcRequestID: "r8"}
	assert.Contains(t, e.Error(), "too many")
	assert.Contains(t, e.Error(), "429")
}

func TestInternalServerErrorOciError_Error(t *testing.T) {
	e := InternalServerErrorOciError{HTTPStatusCode: 500, Description: "internal", OpcRequestID: "r9"}
	assert.Contains(t, e.Error(), "internal")
	assert.Contains(t, e.Error(), "500")
}

func TestMethodNotImplementedOciError_Error(t *testing.T) {
	e := MethodNotImplementedOciError{HTTPStatusCode: 501, Description: "not impl", OpcRequestID: "r10"}
	assert.Contains(t, e.Error(), "not impl")
	assert.Contains(t, e.Error(), "501")
}

func TestServiceUnavailableOciError_Error(t *testing.T) {
	e := ServiceUnavailableOciError{HTTPStatusCode: 503, Description: "unavailable", OpcRequestID: "r11"}
	assert.Contains(t, e.Error(), "unavailable")
	assert.Contains(t, e.Error(), "503")
}

// ---------------------------------------------------------------------------
// Default/fallback cases in specific4xxError and specific5xxError functions
// ---------------------------------------------------------------------------

// specific400Error — unknown code hits the default BadRequestResponse.
func TestBadRequestResponse_DefaultCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("UnknownCode400", 400, "req-400", "unknown 400")
	assert.False(t, resp)
	_, ok := err.(BadRequestOciError)
	assert.True(t, ok, "default 400 should return BadRequestOciError")
	assert.Equal(t, "Bad Request", err.(BadRequestOciError).Description)
}

// specific400Error — CannotParseRequest code.
func TestBadRequestResponse_CannotParseRequest(t *testing.T) {
	resp, err := NewServiceFailureFromResponse(CannotParseRequest, 400, "req", "msg")
	assert.False(t, resp)
	br, ok := err.(BadRequestOciError)
	assert.True(t, ok)
	assert.Equal(t, "The request is incorrectly formatted", br.Description)
}

// specific400Error — LimitExceeded.
func TestBadRequestResponse_LimitExceeded(t *testing.T) {
	resp, err := NewServiceFailureFromResponse(LimitExceeded, 400, "req", "msg")
	assert.False(t, resp)
	br, ok := err.(BadRequestOciError)
	assert.True(t, ok)
	assert.Contains(t, br.Description, "limit")
}

// specific400Error — QuotaExceeded.
func TestBadRequestResponse_QuotaExceeded(t *testing.T) {
	resp, err := NewServiceFailureFromResponse(QuotaExceeded, 400, "req", "msg")
	assert.False(t, resp)
	br, ok := err.(BadRequestOciError)
	assert.True(t, ok)
	assert.Contains(t, br.Description, "quota")
}

// specific400Error — RelatedResourceNotAuthorizedOrNotFound.
func TestBadRequestResponse_RelatedResourceNotAuthorizedOrNotFound(t *testing.T) {
	resp, err := NewServiceFailureFromResponse(RelatedResourceNotAuthorizedOrNotFound, 400, "req", "msg")
	assert.False(t, resp)
	br, ok := err.(BadRequestOciError)
	assert.True(t, ok)
	assert.Contains(t, br.Description, "not found")
}

// specific400Error — InvalidParameters (plural).
func TestBadRequestResponse_InvalidParameters(t *testing.T) {
	resp, err := NewServiceFailureFromResponse(InvalidParameters, 400, "req", "msg")
	assert.False(t, resp)
	br, ok := err.(BadRequestOciError)
	assert.True(t, ok)
	assert.Contains(t, br.Description, "invalid")
}

// specific400Error — MissingParameters (plural).
func TestBadRequestResponse_MissingParameters(t *testing.T) {
	resp, err := NewServiceFailureFromResponse(MissingParameters, 400, "req", "msg")
	assert.False(t, resp)
	br, ok := err.(BadRequestOciError)
	assert.True(t, ok)
	assert.Contains(t, br.Description, "missing")
}

// specific401Error — unknown code returns raw ocierrors.
func TestSpecific401Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("SomeOtherCode", 401, "req", "msg")
	assert.False(t, resp)
	// Should return the raw ocierrors, not NotAuthenticatedOciError
	_, isNotAuth := err.(NotAuthenticatedOciError)
	assert.False(t, isNotAuth)
	assert.NotNil(t, err)
}

// specific402Error — unknown code returns raw ocierrors.
func TestSpecific402Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown402", 402, "req", "msg")
	assert.False(t, resp)
	_, isSignUp := err.(SignUpRequiredOciError)
	assert.False(t, isSignUp)
	assert.NotNil(t, err)
}

// specific403Error — unknown code returns raw ocierrors.
func TestSpecific403Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown403", 403, "req", "msg")
	assert.False(t, resp)
	_, isUA := err.(UnauthorizedAndNotFoundOciError)
	assert.False(t, isUA)
	assert.NotNil(t, err)
}

// specific404Error — unknown code returns raw ocierrors.
func TestSpecific404Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown404", 404, "req", "msg")
	assert.False(t, resp)
	_, isUA := err.(UnauthorizedAndNotFoundOciError)
	assert.False(t, isUA)
	assert.NotNil(t, err)
}

// specific405Error — unknown code returns raw ocierrors.
func TestSpecific405Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown405", 405, "req", "msg")
	assert.False(t, resp)
	_, isMNA := err.(MethodNotAllowedOciError)
	assert.False(t, isMNA)
	assert.NotNil(t, err)
}

// check4xxFailures default case — unhandled 4xx code (e.g., 406).
func TestCheck4xxFailures_DefaultCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("SomeCode", 406, "req", "msg")
	assert.False(t, resp)
	assert.NotNil(t, err)
	// Should be a raw ocierrors.
	assert.Contains(t, err.Error(), "406")
}

// specific409Error — InvalidatedRetryToken.
func TestConflictResponse_InvalidatedRetryToken(t *testing.T) {
	resp, err := NewServiceFailureFromResponse(InvalidatedRetryToken, 409, "req", "msg")
	assert.False(t, resp)
	c, ok := err.(ConflictOciError)
	assert.True(t, ok)
	assert.Contains(t, c.Description, "retry token")
}

// specific409Error — unknown code returns raw ocierrors.
func TestSpecific409Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown409", 409, "req", "msg")
	assert.False(t, resp)
	_, isConflict := err.(ConflictOciError)
	assert.False(t, isConflict)
	assert.NotNil(t, err)
}

// specific412Error — unknown code returns raw ocierrors.
func TestSpecific412Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown412", 412, "req", "msg")
	assert.False(t, resp)
	_, isNEM := err.(NoEtagMatchOciError)
	assert.False(t, isNEM)
	assert.NotNil(t, err)
}

// specific429Error — unknown code returns raw ocierrors.
func TestSpecific429Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown429", 429, "req", "msg")
	assert.False(t, resp)
	_, isTMR := err.(TooManyRequestsOciError)
	assert.False(t, isTMR)
	assert.NotNil(t, err)
}

// check5xxFailures default case — unhandled 5xx (e.g., 502).
func TestCheck5xxFailures_DefaultCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("SomeCode", 502, "req", "msg")
	assert.False(t, resp)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "502")
}

// specific500Error — unknown code returns raw ocierrors.
func TestSpecific500Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown500", 500, "req", "msg")
	assert.False(t, resp)
	_, isISE := err.(InternalServerErrorOciError)
	assert.False(t, isISE)
	assert.NotNil(t, err)
}

// specific501Error — unknown code returns raw ocierrors.
func TestSpecific501Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown501", 501, "req", "msg")
	assert.False(t, resp)
	_, isMNI := err.(MethodNotImplementedOciError)
	assert.False(t, isMNI)
	assert.NotNil(t, err)
}

// specific503Error — unknown code returns raw ocierrors.
func TestSpecific503Error_UnknownCode(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("Unknown503", 503, "req", "msg")
	assert.False(t, resp)
	_, isSU := err.(ServiceUnavailableOciError)
	assert.False(t, isSU)
	assert.NotNil(t, err)
}

// NewServiceFailureFromResponse — status code outside 4xx/5xx (default).
func TestNewServiceFailureFromResponse_OutsideRange(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("SomeCode", 200, "req", "msg")
	assert.True(t, resp)
	assert.Nil(t, err)
}

// ocierrors internal Error() — exercised via check4xxFailures default path.
func TestOcierrors_Error_ViaDefault4xx(t *testing.T) {
	resp, err := NewServiceFailureFromResponse("WeirdCode", 406, "opcid-abc", "some message")
	assert.False(t, resp)
	assert.NotNil(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "WeirdCode")
	assert.Contains(t, msg, "406")
	assert.Contains(t, msg, "opcid-abc")
}
