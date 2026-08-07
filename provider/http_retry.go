package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"
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
// read a request body twice. Only requests that are safe to issue more than
// once are retried at all (see retryableRequest).
type retryTransport struct {
	base http.RoundTripper
}

// retryableRequest reports whether req may safely be issued more than once.
// The BMC performs mutations via GET (opt=set&type=...), so a transport
// error does not prove the request never took effect: the daemon may have
// acted and only the response was lost. Retrying in that window would
// re-fire the mutation (a second reset reboots the node again; a second
// flash init while the first transfer is starting can wedge the BMC in the
// stuck-flash state that needs opt=set&type=reload to clear). Reads, the two
// idempotent sets, and the authenticate POST are the only requests worth a
// second attempt (#153).
func retryableRequest(req *http.Request) bool {
	if strings.HasSuffix(req.URL.Path, "/api/bmc/authenticate") {
		return true
	}
	q := req.URL.Query()
	switch q.Get("opt") {
	case "get":
		return true
	case "set":
		switch q.Get("type") {
		case "power", "reload":
			return true
		}
	}
	return false
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
	if !retryableRequest(req) {
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
			// resp.Body was closed above; returning resp here would hand
			// the caller a response it can no longer read (#152).
			return nil, req.Context().Err()
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
