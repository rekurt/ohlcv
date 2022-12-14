package market

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
)

type errorProcessor interface {
	Decode(r *fasthttp.Response) error
}

type errorCode interface {
	Error() string
	StatusCode() int
}

type httpError struct {
	Code    int
	Message string
}

// Error returns a text message corresponding to the given error.
func (e *httpError) Error() string {
	return e.Message
}

// StatusCode returns an HTTP status code corresponding to the given error.
func (e *httpError) StatusCode() int {
	return e.Code
}

type errorResponse struct {
	Error            bool              `json:"error"`
	ErrorText        string            `json:"errorText"`
	AdditionalErrors map[string]string `json:"additionalErrors"`
	Data             *struct{}         `json:"data"`
}

// ErrorProcessor ...
type ErrorProcessor struct {
	defaultCode int
	errors      map[string]string
}

//Encode writes a svc error to the given http.ResponseWriter.
func (e *ErrorProcessor) Encode(ctx context.Context, r *fasthttp.Response, err error) {
	errorText := err.Error()
	if idx := strings.Index(err.Error(), ":"); idx != -1 {
		numberOfError := err.Error()[:idx]
		if text, ok := e.errors[numberOfError]; ok {
			errorText = text
		}
	}
	res := errorResponse{
		Error:     true,
		ErrorText: errorText,
	}
	if er, ok := err.(errorCode); ok {
		r.SetStatusCode(er.StatusCode())
	} else {
		r.SetStatusCode(e.defaultCode)
	}
	r.Header.Set("Content-Type", "application/json")
	body, err := json.Marshal(res)
	if err != nil {
		return
	}
	r.SetBody(body)
}

// Decode reads a Service error from the given *http.Response.
func (e *ErrorProcessor) Decode(r *fasthttp.Response) error {
	msgBytes := r.Body()
	msg := strings.TrimSpace(string(msgBytes))
	if msg == "" {
		msg = http.StatusText(r.StatusCode())
	}
	return &httpError{
		Code:    r.StatusCode(),
		Message: msg,
	}
}

// NewErrorProcessor ...
func NewErrorProcessor(errors map[string]string) *ErrorProcessor {
	return &ErrorProcessor{
		defaultCode: http.StatusOK,
		errors:      errors,
	}
}
