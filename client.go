package roxyapi

import (
	"context"
	"errors"
	"net/http"
)

// Option configures NewRoxy. Every generated ClientOption (WithBaseURL,
// WithHTTPClient, WithRequestEditorFn) is an Option, as are WithMaxRetries and
// WithRetry.
type Option interface {
	applyRoxy(*roxySettings)
}

type roxySettings struct {
	clientOpts []ClientOption
	retry      RetryConfig
}

func (o ClientOption) applyRoxy(s *roxySettings) { s.clientOpts = append(s.clientOpts, o) }

// DefaultBaseURL is the production RoxyAPI v2 endpoint.
const DefaultBaseURL = "https://roxyapi.com/api/v2"

// NewRoxy returns a domain-grouped client authenticated with the given API key.
// Call methods through the domain services, for example
// roxy.Astrology.ListZodiacSigns(ctx, nil).
//
// Configuration reuses the generated client options (each an Option): pass WithBaseURL to point
// at a different host, WithHTTPClient to supply a custom *http.Client, or
// WithRequestEditorFn to add a header. The API key and SDK identification headers are
// always applied first.
//
// Failed requests are retried twice by default (see RetryConfig); pass
// WithMaxRetries(0) to turn that off.
//
// NewRoxy returns an error if apiKey is empty, so a missing ROXY_API_KEY fails here
// rather than as a confusing 401 on the first call.
func NewRoxy(apiKey string, opts ...Option) (*Roxy, error) {
	if apiKey == "" {
		return nil, errors.New("roxyapi: an API key is required (set ROXY_API_KEY)")
	}
	settings := roxySettings{retry: RetryConfig{MaxRetries: DefaultMaxRetries}}
	for _, o := range opts {
		o.applyRoxy(&settings)
	}
	all := append([]ClientOption{
		WithRequestEditorFn(apiKeyEditor(apiKey)),
		WithRequestEditorFn(sdkClientEditor),
	}, settings.clientOpts...)
	all = append(all, retryClientOption(settings.retry))
	c, err := NewClientWithResponses(DefaultBaseURL, all...)
	if err != nil {
		return nil, err
	}
	return newRoxy(c), nil
}

func apiKeyEditor(key string) RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", key)
		return nil
	}
}

func sdkClientEditor(_ context.Context, req *http.Request) error {
	req.Header.Set("X-SDK-Client", "roxy-sdk-go/"+Version)
	return nil
}
