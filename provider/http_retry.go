package provider

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const (
	retryMaxAttempts = 3
	retryBaseDelay   = 250 * time.Millisecond
	retryMaxDelay    = 2 * time.Second
)

// retryTransport wraps an http.RoundTripper and retries requests that fail
// with a transient network error or a 502/503/504 response. Requests whose
// body cannot be replayed (io.MultiReader over a streamed file, in
// uploadFlashStream) are never retried, since RoundTrip is not permitted to
// read a request body twice.
type retryTransport struct {
	base http.RoundTripper
}

func newRetryTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &retryTransport{base: base}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil && req.GetBody == nil {
		return t.base.RoundTrip(req)
	}

	var resp *http.Response
	var err error
	delay := retryBaseDelay

	for attempt := 1; attempt <= retryMaxAttempts; attempt++ {
		if attempt > 1 && req.GetBody != nil {
			if rewound, rerr := req.GetBody(); rerr == nil {
				req.Body = rewound
			}
		}

		resp, err = t.base.RoundTrip(req)
		if !shouldRetry(resp, err) {
			return resp, err
		}

		if attempt == retryMaxAttempts {
			break
		}
		if resp != nil {
			_ = resp.Body.Close()
		}

		select {
		case <-req.Context().Done():
			return resp, err
		case <-time.After(delay):
		}

		delay *= 2
		if delay > retryMaxDelay {
			delay = retryMaxDelay
		}
	}

	return resp, err
}

func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	switch resp.StatusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
