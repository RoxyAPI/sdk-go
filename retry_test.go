package roxyapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// retryServer answers each request from the scripted replies in order, repeating the
// last one, and counts the requests and the bodies it received.
type reply struct {
	status int
	header map[string]string
	body   string
}

func retryServer(t *testing.T, replies ...reply) (*httptest.Server, *atomic.Int32, *[]string) {
	t.Helper()
	var n atomic.Int32
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		i := int(n.Add(1)) - 1
		rep := replies[min(i, len(replies)-1)]
		for k, v := range rep.header {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rep.status)
		_, _ = w.Write([]byte(rep.body))
	}))
	t.Cleanup(srv.Close)
	return srv, &n, &bodies
}

// retryClient builds a client whose retry wrapper has fixed randomness and records
// the waits instead of sleeping.
func retryClient(t *testing.T, url string, cfg RetryConfig, rnd float64) (*Roxy, *[]time.Duration) {
	t.Helper()
	var waits []time.Duration
	c, err := NewClientWithResponses(url, retryClientOption(cfg))
	if err != nil {
		t.Fatal(err)
	}
	d := c.ClientInterface.(*Client).Client.(*retryDoer)
	d.rnd = func() float64 { return rnd }
	d.sleep = func(_ context.Context, w time.Duration) error { waits = append(waits, w); return nil }
	return newRoxy(c), &waits
}

const perMinuteBody = `{"error":"slow down","code":"rate_limit_perMinuteBody","doc_url":"https://roxyapi.com/docs/errors#rate_limit_perMinuteBody"}`

func TestRetry429PerMinuteThenSuccess(t *testing.T) {
	srv, n, _ := retryServer(t,
		reply{429, map[string]string{"Retry-After": "1"}, perMinuteBody},
		reply{200, nil, `[]`})
	roxy, waits := retryClient(t, srv.URL, RetryConfig{MaxRetries: 2}, 0.5)
	if _, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil); err != nil {
		t.Fatalf("err = %v", err)
	}
	if n.Load() != 2 {
		t.Errorf("requests = %d, want 2", n.Load())
	}
	// retryAfter 1s + 0.5 * 500ms jitter
	if len(*waits) != 1 || (*waits)[0] != 1250*time.Millisecond {
		t.Errorf("waits = %v, want [1.25s]", *waits)
	}
}

func TestRetryAfterBeyondMaxDelaySurfacesError(t *testing.T) {
	srv, n, _ := retryServer(t, reply{429, map[string]string{"Retry-After": "86400"},
		`{"error":"quota","code":"rate_limit_exceeded","doc_url":"https://roxyapi.com/docs/errors#rate_limit_exceeded"}`})
	roxy, waits := retryClient(t, srv.URL, RetryConfig{MaxRetries: 3}, 0)
	_, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil)
	var re *RoxyError
	if !errors.As(err, &re) {
		t.Fatalf("err = %v, want *RoxyError", err)
	}
	if n.Load() != 1 || len(*waits) != 0 {
		t.Errorf("requests = %d waits = %v, want 1 and none", n.Load(), *waits)
	}
	if re.RetryAfter != 24*time.Hour || re.Code != "rate_limit_exceeded" ||
		re.DocURL != "https://roxyapi.com/docs/errors#rate_limit_exceeded" || re.StatusCode != 429 {
		t.Errorf("RoxyError = %+v", *re)
	}
}

func TestRetry503ComputeSaturated(t *testing.T) {
	srv, n, _ := retryServer(t,
		reply{503, nil, `{"error":"busy","code":"compute_saturated"}`},
		reply{503, nil, `{"error":"busy","code":"compute_saturated"}`},
		reply{200, nil, `[]`})
	roxy, waits := retryClient(t, srv.URL, RetryConfig{MaxRetries: 3}, 0.5)
	if _, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	// full jitter: 0.5 * 500ms, then 0.5 * 1s
	if n.Load() != 3 || len(*waits) != 2 || (*waits)[0] != 250*time.Millisecond || (*waits)[1] != 500*time.Millisecond {
		t.Errorf("requests = %d waits = %v", n.Load(), *waits)
	}
}

func TestRetryGivesUpAfterMaxRetries(t *testing.T) {
	srv, n, _ := retryServer(t, reply{503, nil, `{"error":"busy","code":"compute_saturated"}`})
	roxy, _ := retryClient(t, srv.URL, RetryConfig{MaxRetries: 2}, 0)
	_, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil)
	var re *RoxyError
	if !errors.As(err, &re) || re.Code != "compute_saturated" || n.Load() != 3 {
		t.Errorf("err = %v requests = %d, want compute_saturated after 3", err, n.Load())
	}
}

func TestRetryOn500(t *testing.T) {
	srv, n, _ := retryServer(t, reply{500, nil, `{"error":"boom","code":"internal_error"}`}, reply{200, nil, `[]`})
	roxy, waits := retryClient(t, srv.URL, RetryConfig{MaxRetries: 3}, 0.5)
	if _, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil); err != nil || n.Load() != 2 || len(*waits) != 1 {
		t.Errorf("err = %v requests = %d waits = %v, want success after 2", err, n.Load(), *waits)
	}
}

func retryDoerOf(roxy *Roxy) *retryDoer {
	d, _ := roxy.client.ClientInterface.(*Client).Client.(*retryDoer)
	return d
}

func TestNewRoxyRetriesTwiceByDefault(t *testing.T) {
	srv, n, _ := retryServer(t, reply{503, nil, `{"error":"busy","code":"compute_saturated"}`})
	roxy, err := NewRoxy("sk_test", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	d := retryDoerOf(roxy)
	if d == nil {
		t.Fatal("NewRoxy installed no retry wrapper")
	}
	d.sleep = func(context.Context, time.Duration) error { return nil }
	if _, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil); err == nil || n.Load() != 3 {
		t.Errorf("err = %v requests = %d, want an error after 3 (1 + 2 retries)", err, n.Load())
	}
}

func TestZeroMaxRetriesNeverRetries(t *testing.T) {
	srv, n, _ := retryServer(t, reply{503, nil, `{"error":"busy","code":"compute_saturated"}`})
	roxy, err := NewRoxy("sk_test", WithBaseURL(srv.URL), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := roxy.Astrology.ListZodiacSigns(context.Background(), nil); err == nil || n.Load() != 1 {
		t.Errorf("err = %v requests = %d, want an error after 1", err, n.Load())
	}
}

type countingDoer struct {
	inner HttpRequestDoer
	calls atomic.Int32
}

func (c *countingDoer) Do(r *http.Request) (*http.Response, error) {
	c.calls.Add(1)
	return c.inner.Do(r)
}

func TestRetryComposesWithHTTPClientInAnyOrder(t *testing.T) {
	for name, order := range map[string]func(HttpRequestDoer) []Option{
		"retry first": func(d HttpRequestDoer) []Option { return []Option{WithMaxRetries(1), WithHTTPClient(d)} },
		"retry last":  func(d HttpRequestDoer) []Option { return []Option{WithHTTPClient(d), WithMaxRetries(1)} },
	} {
		t.Run(name, func(t *testing.T) {
			srv, n, _ := retryServer(t, reply{503, nil, `{"error":"busy","code":"compute_saturated"}`})
			custom := &countingDoer{inner: http.DefaultClient}
			roxy, err := NewRoxy("sk_test", append([]Option{WithBaseURL(srv.URL)}, order(custom)...)...)
			if err != nil {
				t.Fatal(err)
			}
			retryDoerOf(roxy).sleep = func(context.Context, time.Duration) error { return nil }
			_, _ = roxy.Astrology.ListZodiacSigns(context.Background(), nil)
			if custom.calls.Load() != 2 || n.Load() != 2 {
				t.Errorf("custom client calls = %d, requests = %d, want 2 and 2", custom.calls.Load(), n.Load())
			}
		})
	}
}

func TestRetryReplaysBody(t *testing.T) {
	srv, n, bodies := retryServer(t, reply{502, nil, `{"error":"bad gateway","code":"error"}`}, reply{200, nil, `{}`})
	c, err := NewClientWithResponses(srv.URL, retryClientOption(RetryConfig{MaxRetries: 1}))
	if err != nil {
		t.Fatal(err)
	}
	d := c.ClientInterface.(*Client).Client.(*retryDoer)
	d.sleep = func(context.Context, time.Duration) error { return nil }
	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(`{"a":1}`))
	resp, err := d.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if n.Load() != 2 || (*bodies)[0] != `{"a":1}` || (*bodies)[1] != `{"a":1}` {
		t.Errorf("requests = %d bodies = %q", n.Load(), *bodies)
	}
}

func TestRetryNeverSleepsPastDeadline(t *testing.T) {
	srv, n, _ := retryServer(t, reply{429, map[string]string{"Retry-After": "20"}, perMinuteBody})
	roxy, waits := retryClient(t, srv.URL, RetryConfig{MaxRetries: 3}, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := roxy.Astrology.ListZodiacSigns(ctx, nil)
	var re *RoxyError
	if !errors.As(err, &re) || re.Code != "rate_limit_perMinuteBody" || n.Load() != 1 || len(*waits) != 0 {
		t.Errorf("err = %v requests = %d waits = %v", err, n.Load(), *waits)
	}
}

func TestRetryRetriesRefusedConnection(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	url := "http://" + ln.Addr().String()
	_ = ln.Close() // nothing listens here any more: the dial is refused
	roxy, waits := retryClient(t, url, RetryConfig{MaxRetries: 2}, 0.5)
	_, err = roxy.Astrology.ListZodiacSigns(context.Background(), nil)
	if err == nil || len(*waits) != 2 {
		t.Errorf("err = %v waits = %v, want an error after 2 retries", err, *waits)
	}
}

func TestBackoffBounds(t *testing.T) {
	d := &retryDoer{cfg: RetryConfig{InitialDelay: 500 * time.Millisecond, MaxDelay: 30 * time.Second}}
	for _, tc := range []struct {
		attempt int
		ceiling time.Duration
	}{{0, 500 * time.Millisecond}, {3, 4 * time.Second}, {10, 30 * time.Second}, {80, 30 * time.Second}} {
		d.rnd = func() float64 { return 0 }
		if got := d.backoff(tc.attempt); got != 0 {
			t.Errorf("attempt %d low = %v, want 0", tc.attempt, got)
		}
		d.rnd = func() float64 { return 0.999999 }
		if got := d.backoff(tc.attempt); got <= 0 || got > tc.ceiling {
			t.Errorf("attempt %d high = %v, want within (0, %v]", tc.attempt, got, tc.ceiling)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"7", 7 * time.Second, true},
		{"Thu, 01 Jan 2026 00:01:00 GMT", time.Minute, true},
		{"Wed, 31 Dec 2025 00:00:00 GMT", 0, true},
		{"", 0, false},
		{"soon", 0, false},
		{"-3", 0, false},
	} {
		h := http.Header{}
		if tc.in != "" {
			h.Set("Retry-After", tc.in)
		}
		got, ok := parseRetryAfter(h, now)
		if got != tc.want || ok != tc.ok {
			t.Errorf("parseRetryAfter(%q) = %v %v, want %v %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

type closeSpy struct {
	io.Reader
	closed bool
}

func (c *closeSpy) Close() error { c.closed = true; return nil }

type scriptedDoer struct {
	bodies []*closeSpy
	status []int
}

func (s *scriptedDoer) Do(*http.Request) (*http.Response, error) {
	i := len(s.bodies)
	b := &closeSpy{Reader: strings.NewReader("{}")}
	s.bodies = append(s.bodies, b)
	return &http.Response{StatusCode: s.status[min(i, len(s.status)-1)], Header: http.Header{}, Body: b}, nil
}

func TestRetryClosesDiscardedResponseOnly(t *testing.T) {
	inner := &scriptedDoer{status: []int{503, 200}}
	d := &retryDoer{inner: inner, cfg: RetryConfig{MaxRetries: 1, InitialDelay: time.Second, MaxDelay: time.Minute},
		rnd: func() float64 { return 0 }, now: time.Now,
		sleep: func(context.Context, time.Duration) error {
			if !inner.bodies[0].closed {
				t.Error("discarded response was not closed before the wait")
			}
			return nil
		}}
	req, _ := http.NewRequest(http.MethodGet, "http://example.invalid", nil)
	resp, err := d.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("resp = %v err = %v", resp, err)
	}
	if inner.bodies[1].closed {
		t.Error("the returned response was closed")
	}
}

func TestRetryStopsWhenContextCancelledDuringWait(t *testing.T) {
	inner := &scriptedDoer{status: []int{503}}
	ctx, cancel := context.WithCancel(context.Background())
	d := &retryDoer{inner: inner, cfg: RetryConfig{MaxRetries: 3, InitialDelay: time.Hour, MaxDelay: time.Hour},
		rnd: func() float64 { return 0.9 }, now: time.Now, sleep: sleepCtx}
	time.AfterFunc(20*time.Millisecond, cancel)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", nil)
	if _, err := d.Do(req); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if len(inner.bodies) != 1 {
		t.Errorf("attempts = %d, want 1", len(inner.bodies))
	}
}
