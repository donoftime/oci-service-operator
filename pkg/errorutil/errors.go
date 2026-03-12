/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package errorutil

import (
	"errors"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
)

type ocierrors struct {
	HTTPStatusCode int
	ErrorCode      string `json:"code,omitempty"`
	Description    string `json:"message,omitempty"`
	OpcRequestID   string `json:"opc-request-id"`
}

func (o ocierrors) Error() string {
	return fmt.Sprintf("Service error:%s. %s. http status code: %d. Opc request id: %s",
		o.ErrorCode, o.Description, o.HTTPStatusCode, o.OpcRequestID)
}

var fourXXHandlers = map[int]func(ocierrors) (bool, error){
	400: specific400Error,
	401: specific401Error,
	402: specific402Error,
	403: specific403Error,
	404: specific404Error,
	405: specific405Error,
	409: specific409Error,
	412: specific412Error,
	429: specific429Error,
}

var fiveXXHandlers = map[int]func(ocierrors) (bool, error){
	500: specific500Error,
	501: specific501Error,
	503: specific503Error,
}

func OciErrorTypeResponse(err error) (bool, error) {
	serviceErr, ok := asServiceError(err)
	if !ok {
		return false, err
	}
	return NewServiceFailureFromResponse(
		serviceErr.GetCode(),
		serviceErr.GetHTTPStatusCode(),
		serviceErr.GetOpcRequestID(),
		serviceErr.GetMessage(),
	)
}

func asServiceError(err error) (common.ServiceError, bool) {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return nil, false
	}
	return serviceErr, true
}

func NewServiceFailureFromResponse(code string, statusCode int, opcRequestId string, message string) (bool, error) {
	se := ocierrors{
		ErrorCode:      code,
		HTTPStatusCode: statusCode,
		OpcRequestID:   opcRequestId,
		Description:    message,
	}

	switch {
	case statusCode >= 400 && statusCode <= 499:
		return check4xxFailures(se)
	case statusCode >= 500 && statusCode <= 599:
		return check5xxFailures(se)
	default:
		return true, nil
	}
}

// Return the specific 4xx error Object
func check4xxFailures(se ocierrors) (bool, error) {
	handler, ok := fourXXHandlers[se.HTTPStatusCode]
	if !ok {
		return false, se
	}
	return handler(se)
}

// Return the specific 5xx error Object
func check5xxFailures(se ocierrors) (bool, error) {
	handler, ok := fiveXXHandlers[se.HTTPStatusCode]
	if !ok {
		return false, se
	}
	return handler(se)
}

// Return specific 400 error. Their are different types of 400 error ex. Invalid-Parameter or Missing-Parameter etc.
func specific400Error(se ocierrors) (bool, error) {
	switch se.ErrorCode {
	case CannotParseRequest:
		return false, BadRequestResponse(se, "The request is incorrectly formatted")
	case InvalidParameter, InvalidParameters:
		return false, BadRequestResponse(se, "Parameter is invalid or incorrectly formatted")
	case MissingParameter, MissingParameters:
		return false, BadRequestResponse(se, "Required parameter is missing")
	case LimitExceeded:
		return false, BadRequestResponse(se, "Fulfilling this request exceeds the Oracle-defined "+
			"limit for this tenancy for this resource type")
	case QuotaExceeded:
		return false, BadRequestResponse(se, "Fulfilling this request exceeds the "+
			"administrator-defined quota for this compartment for this resource")
	case RelatedResourceNotAuthorizedOrNotFound:
		return false, BadRequestResponse(se, "A resource specified in the body of the request was "+
			"not found, or you do not have authorization to access that resource")
	default:
		return false, BadRequestResponse(se, "Bad Request")
	}
}

func specific401Error(se ocierrors) (bool, error) {
	if se.ErrorCode == NotAuthenticated {
		return false, NotAuthenticatedResponse(se, "The required authentication information was not "+
			"provided or was incorrect")
	}
	return false, se
}

func specific402Error(se ocierrors) (bool, error) {
	if se.ErrorCode == SignUpRequired {
		return false, SignUpRequiredResponse(se, "This operation requires opt-in before it may be called")
	}
	return false, se
}

func specific403Error(se ocierrors) (bool, error) {
	if se.ErrorCode == NotAuthorized {
		return false, UnauthorizedAndNotFoundResponse(se, "You do not have authorization to update one "+
			"or more of the fields included in this request")
	}
	return false, se
}

func specific404Error(se ocierrors) (bool, error) {
	switch se.ErrorCode {
	case NotFound:
		return false, UnauthorizedAndNotFoundResponse(se, "There is no operation supported at the "+
			"URI path and HTTP method you specified in the request")
	case NotAuthorizedOrNotFound:
		return false, UnauthorizedAndNotFoundResponse(se, "A resource specified via the URI (path or "+
			"query parameters) of the request was not found, or you do not have authorization to access that resource")
	default:
		return false, se
	}
}

func specific405Error(se ocierrors) (bool, error) {
	if se.ErrorCode == MethodNotAllowed {
		return false, MethodNotAllowedResponse(se, "The target resource does not support the HTTP method")
	}
	return false, se
}

func specific409Error(se ocierrors) (bool, error) {
	switch se.ErrorCode {
	case IncorrectState:
		return false, ConflictResponse(se, "The requested state for the resource conflicts with "+
			"its current state")
	case InvalidatedRetryToken:
		return false, ConflictResponse(se, "The provided retry token was used in an earlier request "+
			"that resulted in a system update, but a subsequent operation invalidated "+
			"the token")
	case NotAuthorizedOrResourceAlreadyExists:
		return false, ConflictResponse(se, "You do not have authorization to perform this request, or "+
			"the resource you are attempting to create already exists")
	default:
		return false, se
	}
}

func specific412Error(se ocierrors) (bool, error) {
	if se.ErrorCode == NoEtagMatch {
		return false, NoEtagMatchResponse(se, "The ETag specified in the request does not match the ETag for "+
			"the resource")
	}
	return false, se
}

func specific429Error(se ocierrors) (bool, error) {
	if se.ErrorCode == TooManyRequests {
		return false, TooManyRequestsResponse(se, "You have issued too many requests to the Oracle Cloud "+
			"Infrastructure APIs in too short of an amount of time")
	}
	return false, se
}

func specific500Error(se ocierrors) (bool, error) {
	if se.ErrorCode == InternalServerError {
		return false, InternalServerErrorResponse(se, "An internal server error occurred")
	}
	return false, se
}

func specific501Error(se ocierrors) (bool, error) {
	if se.ErrorCode == MethodNotImplemented {
		return false, MethodNotImplementedResponse(se, "The HTTP request target does not recognize the HTTP method")
	}
	return false, se
}

func specific503Error(se ocierrors) (bool, error) {
	if se.ErrorCode == ServiceUnavailable {
		return false, ServiceUnavailableResponse(se, "The service is currently unavailable")
	}
	return false, se
}
