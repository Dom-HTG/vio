package model

import (
	"errors"
	"fmt"
)

var (
	ErrConfig          = errors.New("model: invalid configuration")
	ErrAuth            = errors.New("model: authentication failed")
	ErrHTTP            = errors.New("model: http request failed")
	ErrTimeout         = errors.New("model: request timed out")
	ErrInvalidResponse = errors.New("model: invalid response")
	ErrInvalidToolCall = errors.New("model: malformed tool call")
)

type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("model: http %d: %s", e.StatusCode, e.Body)
}

func (e *HTTPStatusError) Unwrap() error {
	if e.StatusCode == 401 || e.StatusCode == 403 {
		return ErrAuth
	}
	return ErrHTTP
}
