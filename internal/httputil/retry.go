// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

var ErrUpstreamUnavailable = errors.New("upstream unavailable")

var (
	apiHTTPClientOnce    sync.Once
	apiHTTPClient        *http.Client
	streamHTTPClientOnce sync.Once
	streamHTTPClient     *http.Client
)

func NewRetryHTTPClient() *http.Client {
	apiHTTPClientOnce.Do(func() {
		apiHTTPClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &retryTransport{
				base: APITransport(),
			},
		}
	})
	return apiHTTPClient
}

func NewStreamingHTTPClient() *http.Client {
	streamHTTPClientOnce.Do(func() {
		streamHTTPClient = &http.Client{
			Timeout: 0,
			Transport: &retryTransport{
				base: StreamTransport(),
			},
		}
	})
	return streamHTTPClient
}

type retryTransport struct {
	base http.RoundTripper
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * 200 * time.Millisecond
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(backoff):
			}
		}

		cloned := req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			cloned.Body = body
		}

		resp, err := base.RoundTrip(cloned)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 500 {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			lastErr = ErrUpstreamUnavailable
			continue
		}
		return resp, nil
	}
	if lastErr == nil {
		lastErr = ErrUpstreamUnavailable
	}
	return nil, lastErr
}

func DoWithRetry(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	if client == nil {
		client = NewRetryHTTPClient()
	}
	req = req.WithContext(ctx)
	return client.Do(req) //#nosec G704 -- caller supplies URLs for configured Subsonic/media upstreams
}
