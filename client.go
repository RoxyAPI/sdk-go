package roxyapi

import (
	"context"
	"errors"
	"net/http"
)

// DefaultBaseURL is the production RoxyAPI v2 endpoint.
const DefaultBaseURL = "https://roxyapi.com/api/v2"

// NewRoxy returns a domain-grouped client authenticated with the given API key.
// Call methods through the domain services, for example
// roxy.Astrology.ListZodiacSigns(ctx, nil).
//
// Configuration reuses the generated ClientOption values: pass WithBaseURL to point
// at a different host, WithHTTPClient to supply a custom *http.Client, or
// WithRequestEditorFn to add a header. The API key and SDK identification headers are
// always applied first.
//
// NewRoxy returns an error if apiKey is empty, so a missing ROXYAPI_KEY fails here
// rather than as a confusing 401 on the first call.
func NewRoxy(apiKey string, opts ...ClientOption) (*Roxy, error) {
	if apiKey == "" {
		return nil, errors.New("roxyapi: an API key is required (set ROXYAPI_KEY)")
	}
	all := append([]ClientOption{
		WithRequestEditorFn(apiKeyEditor(apiKey)),
		WithRequestEditorFn(sdkClientEditor),
	}, opts...)
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
