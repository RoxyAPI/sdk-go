package roxyapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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
// Message is human readable and its wording may change. DocURL links to the fix for
// that exact code, and RetryAfter is the wait the response asked for, zero when it
// sent none.
type RoxyError struct {
	StatusCode int
	Code       string
	Message    string
	Issues     []RoxyErrorIssue // populated on 400 validation errors
	Allow      []string         // populated on 405, the methods the path accepts
	DocURL     string           // documentation link for this exact code (the doc_url field)
	Docs       string           // domain documentation link, present on a path-miss 404 (the docs field)
	RetryAfter time.Duration    // the Retry-After wait, zero when the response sent none
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
// calls it only after confirming the transport error is nil, so r is never nil. h is
// the raw HTTP response, read for Retry-After.
func asRoxyError(r roxyResponse, h *http.Response) error {
	status := r.StatusCode()
	if status < 400 {
		return nil
	}
	e := &RoxyError{StatusCode: status, Code: "error", Message: http.StatusText(status)}
	if h != nil {
		e.RetryAfter, _ = parseRetryAfter(h.Header, time.Now())
	}
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
		e.DocURL = strVal(body.DocURL)
		e.Docs = strVal(body.Docs)
	}
	return e
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// parseRetryAfter reads a Retry-After header, either whole seconds or an HTTP-date,
// and reports whether one was present and valid. A date in the past is a zero wait.
func parseRetryAfter(h http.Header, now time.Time) (time.Duration, bool) {
	v := strings.TrimSpace(h.Get("Retry-After"))
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.ParseInt(v, 10, 64); err == nil {
		if secs < 0 {
			return 0, false
		}
		if secs > int64(time.Duration(1<<62)/time.Second) {
			return time.Duration(1 << 62), true
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(0, t.Sub(now)), true
	}
	return 0, false
}
