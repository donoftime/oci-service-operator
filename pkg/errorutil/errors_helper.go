/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"fmt"
)

type OciErrors struct {
	HTTPStatusCode int    `json:"http-status-code,omitempty"`
	ErrorCode      string `json:"error-code,omitempty"`
	OpcRequestID   string `json:"opc-request-id,omitempty"`
	Description    string `json:"description,omitempty"`
}

func (bd OciErrors) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		bd.Description, bd.HTTPStatusCode, bd.OpcRequestID)
}

// For 400 errors
type BadRequestOciError OciErrors

func (bd BadRequestOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		bd.Description, bd.HTTPStatusCode, bd.OpcRequestID)
}

// For 401 errors
type NotAuthenticatedOciError OciErrors

func (na NotAuthenticatedOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		na.Description, na.HTTPStatusCode, na.OpcRequestID)
}

// For 402 errors
type SignUpRequiredOciError OciErrors

func (sr SignUpRequiredOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		sr.Description, sr.HTTPStatusCode, sr.OpcRequestID)
}

// For 403 and 404 errors
type UnauthorizedAndNotFoundOciError OciErrors

func (ua UnauthorizedAndNotFoundOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		ua.Description, ua.HTTPStatusCode, ua.OpcRequestID)
}

// For 405 errors
type MethodNotAllowedOciError OciErrors

func (mna MethodNotAllowedOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		mna.Description, mna.HTTPStatusCode, mna.OpcRequestID)
}

// For 409 errors
type ConflictOciError OciErrors

func (c ConflictOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		c.Description, c.HTTPStatusCode, c.OpcRequestID)
}

// For 412 errors
type NoEtagMatchOciError OciErrors

func (nem NoEtagMatchOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		nem.Description, nem.HTTPStatusCode, nem.OpcRequestID)
}

// For 429 Error
type TooManyRequestsOciError OciErrors

func (tmr TooManyRequestsOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		tmr.Description, tmr.HTTPStatusCode, tmr.OpcRequestID)
}

// For 500 Error
type InternalServerErrorOciError OciErrors

func (nf InternalServerErrorOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		nf.Description, nf.HTTPStatusCode, nf.OpcRequestID)
}

// For 501 Error
type MethodNotImplementedOciError OciErrors

func (mni MethodNotImplementedOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		mni.Description, mni.HTTPStatusCode, mni.OpcRequestID)
}

// For 503 Error
type ServiceUnavailableOciError OciErrors

func (su ServiceUnavailableOciError) Error() string {
	return fmt.Sprintf("Service error:%s. http status code: %d. Opc request id: %s",
		su.Description, su.HTTPStatusCode, su.OpcRequestID)
}

const (
	CannotParseRequest                     string = "CannotParseRequest"
	InvalidParameters                      string = "InvalidParameters"
	InvalidParameter                       string = "InvalidParameter"
	LimitExceeded                          string = "LimitExceeded"
	MissingParameters                      string = "MissingParameters"
	MissingParameter                       string = "MissingParameter"
	QuotaExceeded                          string = "QuotaExceeded"
	RelatedResourceNotAuthorizedOrNotFound string = "RelatedResourceNotAuthorizedOrNotFound"
	NotAuthenticated                       string = "NotAuthenticated"
	SignUpRequired                         string = "SignUpRequired"
	NotAuthorizedOrNotFound                string = "NotAuthorizedOrNotFound"
	NotFound                               string = "NotFound"
	MethodNotAllowed                       string = "MethodNotAllowed"
	IncorrectState                         string = "IncorrectState"
	InvalidatedRetryToken                  string = "InvalidatedRetryToken"
	NotAuthorizedOrResourceAlreadyExists   string = "NotAuthorizedOrResourceAlreadyExists"
	NotAuthorized                          string = "NotAuthorized"
	NoEtagMatch                            string = "NoEtagMatch"
	TooManyRequests                        string = "TooManyRequests"
	InternalServerError                    string = "InternalServerError"
	MethodNotImplemented                   string = "MethodNotImplemented"
	ServiceUnavailable                     string = "ServiceUnavailable"
)

func buildOciError(err ocierrors, description string) OciErrors {
	return OciErrors{
		ErrorCode:      err.ErrorCode,
		HTTPStatusCode: err.HTTPStatusCode,
		Description:    description,
		OpcRequestID:   err.OpcRequestID,
	}
}

// For 400 errors
func BadRequestResponse(err ocierrors, description string) error {
	return BadRequestOciError(buildOciError(err, description))
}

// For 401 errros
func NotAuthenticatedResponse(err ocierrors, description string) error {
	return NotAuthenticatedOciError(buildOciError(err, description))
}

// For 402 errors
func SignUpRequiredResponse(err ocierrors, description string) error {
	return SignUpRequiredOciError(buildOciError(err, description))
}

// For 403 and 404 errors
func UnauthorizedAndNotFoundResponse(err ocierrors, description string) error {
	return UnauthorizedAndNotFoundOciError(buildOciError(err, description))
}

// For 405 errors
func MethodNotAllowedResponse(err ocierrors, description string) error {
	return MethodNotAllowedOciError(buildOciError(err, description))
}

// For 409 errors
func ConflictResponse(err ocierrors, description string) error {
	return ConflictOciError(buildOciError(err, description))
}

// For 412 errors
func NoEtagMatchResponse(err ocierrors, description string) error {
	return NoEtagMatchOciError(buildOciError(err, description))
}

// For 429 Error
func TooManyRequestsResponse(err ocierrors, description string) error {
	return TooManyRequestsOciError(buildOciError(err, description))
}

// For 500 error
func InternalServerErrorResponse(err ocierrors, description string) error {
	return InternalServerErrorOciError(buildOciError(err, description))
}

// For 501 error
func MethodNotImplementedResponse(err ocierrors, description string) error {
	return MethodNotImplementedOciError(buildOciError(err, description))
}

// For 503 error
func ServiceUnavailableResponse(err ocierrors, description string) error {
	return ServiceUnavailableOciError(buildOciError(err, description))
}
