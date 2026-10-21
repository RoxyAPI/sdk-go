package roxyapi

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"time"
)

const (
	defaultRetryInitialDelay = 500 * time.Millisecond
	defaultRetryMaxDelay     = 30 * time.Second
)

// DefaultMaxRetries is the retry count NewRoxy installs when no retry option is given.
const DefaultMaxRetries = 2

// RetryConfig configures retries. NewRoxy retries twice by default; MaxRetries 0
// turns retries off.
//
// Retried: HTTP 429, 500, 502, 503 and 504, plus a failure to connect (refused or
// DNS), where the request provably never reached the server. A read timeout is
// never retried. A Retry-After longer than MaxDelay is not waited out: the call
// returns the API error, whose RetryAfter field carries the wait.
type RetryConfig struct {
	MaxRetries   int           // retries after the first attempt, 0 disables
	InitialDelay time.Duration // base of the backoff and the jitter on Retry-After, default 500ms
	MaxDelay     time.Duration // cap on any single wait, default 30s
}

// WithMaxRetries sets the retry count with the default delays; WithMaxRetries(0)
// turns retries off. It is an Option for NewRoxy and combines with the client
// options in any order.
func WithMaxRetries(n int) Option {
	return WithRetry(RetryConfig{MaxRetries: n})
}

// WithRetry sets the retry policy. A request body is replayed on each retry, and a
// wait never runs past the deadline of the request context. NewRoxy wraps the final
// HTTP client after every client option is applied, so the order relative to
// WithHTTPClient does not matter.
func WithRetry(cfg RetryConfig) Option {
	return retryOption(cfg)
}

type retryOption RetryConfig

func (o retryOption) applyRoxy(s *roxySettings) { s.retry = RetryConfig(o) }

// retryClientOption wraps whichever HTTP client the options before it set, or the
// default one. A zero or negative MaxRetries installs no wrapper.
func retryClientOption(cfg RetryConfig) ClientOption {
	return func(c *Client) error {
		if cfg.MaxRetries <= 0 {
			return nil
		}
		if cfg.InitialDelay <= 0 {
			cfg.InitialDelay = defaultRetryInitialDelay
		}
		if cfg.MaxDelay <= 0 {
			cfg.MaxDelay = defaultRetryMaxDelay
		}
		inner := c.Client
		if inner == nil {
			inner = &http.Client{}
		}
		c.Client = &retryDoer{inner: inner, cfg: cfg, rnd: rand.Float64, sleep: sleepCtx, now: time.Now}
		return nil
	}
}

type retryDoer struct {
	inner HttpRequestDoer
	cfg   RetryConfig
	rnd   func() float64 // uniform in [0, 1)
	sleep func(context.Context, time.Duration) error
	now   func() time.Time
}

func (d *retryDoer) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	for attempt := 0; ; attempt++ {
		resp, err := d.inner.Do(req)
		if attempt >= d.cfg.MaxRetries {
			return resp, err
		}
		delay, ok := d.nextDelay(resp, err, attempt)
		if !ok || !d.fitsDeadline(ctx, delay) {
			return resp, err
		}
		next, ok := replay(req)
		if !ok {
			return resp, err
		}
		if resp != nil {
			_, _ = io.CopyN(io.Discard, resp.Body, 64<<10)
			_ = resp.Body.Close()
		}
		if err := d.sleep(ctx, delay); err != nil {
			return nil, err
		}
		req = next
	}
}

// nextDelay decides whether the outcome is retryable and how long to wait first.
func (d *retryDoer) nextDelay(resp *http.Response, err error, attempt int) (time.Duration, bool) {
	if err != nil {
		if !neverSent(err) {
			return 0, false
		}
		return d.backoff(attempt), true
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
	default:
		return 0, false
	}
	if ra, ok := parseRetryAfter(resp.Header, d.now()); ok {
		if ra > d.cfg.MaxDelay {
			return 0, false
		}
		return min(d.cfg.MaxDelay, ra+d.jitter(d.cfg.InitialDelay)), true
	}
	return d.backoff(attempt), true
}

// backoff is full jitter: uniform in [0, min(MaxDelay, InitialDelay * 2^attempt)].
func (d *retryDoer) backoff(attempt int) time.Duration {
	ceiling := d.cfg.InitialDelay
	for i := 0; i < attempt && ceiling < d.cfg.MaxDelay; i++ {
		ceiling *= 2
	}
	return d.jitter(min(ceiling, d.cfg.MaxDelay))
}

func (d *retryDoer) jitter(limit time.Duration) time.Duration {
	return time.Duration(d.rnd() * float64(limit))
}

func (d *retryDoer) fitsDeadline(ctx context.Context, delay time.Duration) bool {
	dl, ok := ctx.Deadline()
	return !ok || d.now().Add(delay).Before(dl)
}

// replay clones the request with a fresh body, or reports false when the body
// cannot be replayed.
func replay(req *http.Request) (*http.Request, bool) {
	next := req.Clone(req.Context())
	if req.Body == nil || req.Body == http.NoBody {
		return next, true
	}
	if req.GetBody == nil {
		return nil, false
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, false
	}
	next.Body = body
	return next, true
}

// neverSent reports a failure to connect, which cannot have delivered the request.
func neverSent(err error) bool {
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return true
	}
	var op *net.OpError
	return errors.As(err, &op) && op.Op == "dial"
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
