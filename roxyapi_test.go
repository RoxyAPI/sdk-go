package roxyapi_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	roxyapi "github.com/RoxyAPI/sdk-go"
)

// captureTransport records the last request and returns a canned response, so the
// header and error-mapping tests run offline and deterministically.
type captureTransport struct {
	last   *http.Request
	status int
	body   string
}

func (t *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.last = r
	return &http.Response{
		StatusCode: t.status,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     http.Header{"Content-Type": {"application/json"}},
	}, nil
}

func TestRequestHeaders(t *testing.T) {
	tr := &captureTransport{status: 200, body: "{}"}
	roxy, err := roxyapi.NewRoxy("sk_test_key", roxyapi.WithHTTPClient(&http.Client{Transport: tr}))
	if err != nil {
		t.Fatalf("NewRoxy: %v", err)
	}
	// The response body is irrelevant here; we only assert the outgoing headers.
	_, _ = roxy.Astrology.ListZodiacSigns(context.Background(), nil)
	if got := tr.last.Header.Get("X-API-Key"); got != "sk_test_key" {
		t.Errorf("X-API-Key = %q, want sk_test_key", got)
	}
	if got, want := tr.last.Header.Get("X-SDK-Client"), "roxy-sdk-go/"+roxyapi.Version; got != want {
		t.Errorf("X-SDK-Client = %q, want %q", got, want)
	}
}

func TestErrorMapping(t *testing.T) {
	tr := &captureTransport{status: 401, body: `{"error":"Invalid API key","code":"unauthorized"}`}
	roxy, err := roxyapi.NewRoxy("sk_bad", roxyapi.WithHTTPClient(&http.Client{Transport: tr}))
	if err != nil {
		t.Fatalf("NewRoxy: %v", err)
	}
	_, err = roxy.Astrology.ListZodiacSigns(context.Background(), nil)
	var re *roxyapi.RoxyError
	if !errors.As(err, &re) {
		t.Fatalf("error = %v (%T), want *RoxyError", err, err)
	}
	if re.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", re.StatusCode)
	}
	if re.Code != "unauthorized" {
		t.Errorf("Code = %q, want unauthorized", re.Code)
	}
}

// TestLive hits production. It is skipped unless an API key is present, so CI without
// the secret stays green.
func TestLive(t *testing.T) {
	key := os.Getenv("ROXY_API_KEY")
	if key == "" {
		key = os.Getenv("ROXYAPI_KEY")
	}
	if key == "" {
		t.Skip("set ROXY_API_KEY to run the live test")
	}
	roxy, err := roxyapi.NewRoxy(key)
	if err != nil {
		t.Fatalf("NewRoxy: %v", err)
	}
	resp, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListZodiacSigns: %v", err)
	}
	if resp.StatusCode() != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		t.Fatal("JSON200 is nil")
	}
}
