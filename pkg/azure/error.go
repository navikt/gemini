package azure

import (
	"encoding/json"
	"errors"

	"golang.org/x/oauth2"
)

type ApiError struct {
	ErrorIdentifier  string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorCodes       []int  `json:"error_codes"`
	Timestamp        string `json:"timestamp"`
	TraceId          string `json:"trace_id"`
	CorrelationId    string `json:"correlation_id"`
	ErrorUri         string `json:"error_uri"`
}

func (apiError ApiError) Error() string {
	return apiError.ErrorDescription
}

func DecodeOauth2ApiError(err error) *ApiError {
	var e *oauth2.RetrieveError
	switch {
	case errors.As(err, &e):
		apiError := &ApiError{}
		err = json.Unmarshal(e.Body, apiError)
		if err != nil {
			return nil
		}
		return apiError
	case err == nil:
		err = errors.Unwrap(err)
		if err == nil {
			return nil
		}
		return DecodeOauth2ApiError(err)
	}
	return nil
}
