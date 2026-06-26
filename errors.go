package roxyapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// RoxyErrorIssue is a single field-level validation failure, present on a 400.
type RoxyErrorIssue struct {
	Path     string
	Message  string
	Code     string
	Expected string
}

// RoxyError is returned by every facade method (roxy.Astrology.X and friends) when
// the API responds with a 4xx or 5xx. Switch on Code, which is a stable identifier;
// Message is human readable and its wording may change.
type RoxyError struct {
	StatusCode int
	Code       string
	Message    string
	Issues     []RoxyErrorIssue // populated on 400 validation errors
	Allow      []string         // populated on 405, the methods the path accepts
	Docs       string           // documentation link, when the API supplies one
}

func (e *RoxyError) Error() string {
	return fmt.Sprintf("roxyapi: %d %s: %s", e.StatusCode, e.Code, e.Message)
}

// roxyResponse is the subset of every generated ClientWithResponses response type
// the error mapper needs. Bytes is enabled via output-options.client-response-bytes-function.
type roxyResponse interface {
	StatusCode() int
	Bytes() []byte
}

// asRoxyError maps a non-2xx response to a *RoxyError, or returns nil. The facade
// calls it only after confirming the transport error is nil, so r is never nil.
func asRoxyError(r roxyResponse) error {
	status := r.StatusCode()
	if status < 400 {
		return nil
	}
	e := &RoxyError{StatusCode: status, Code: "error", Message: http.StatusText(status)}
	var body ErrorResponse
	if err := json.Unmarshal(r.Bytes(), &body); err == nil {
		if body.Code != "" {
			e.Code = body.Code
		}
		if body.Error != "" {
			e.Message = body.Error
		}
		if body.Issues != nil {
			for _, is := range *body.Issues {
				e.Issues = append(e.Issues, RoxyErrorIssue{
					Path:     strVal(is.Path),
					Message:  strVal(is.Message),
					Code:     strVal(is.Code),
					Expected: strVal(is.Expected),
				})
			}
		}
		if body.Allow != nil {
			e.Allow = *body.Allow
		}
		if body.Docs != nil {
			e.Docs = *body.Docs
		}
	}
	return e
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
