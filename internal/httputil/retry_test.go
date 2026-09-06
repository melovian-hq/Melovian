// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRetryTransportRetriesServerErrors(t *testing.T) {
	var attempts atomic.Int32
	transport := &retryTransport{
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			count := attempts.Add(1)
			if count < 3 {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(stringsReader("")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(stringsReader("ok")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected success after retries, got %d", resp.StatusCode)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetryTransportReturnsLastError(t *testing.T) {
	transport := &retryTransport{
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	_, err := transport.RoundTrip(req)
	if err == nil || err.Error() != "network down" {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestRetryTransportHonorsContextCancellation(t *testing.T) {
	transport := &retryTransport{
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil).WithContext(ctx)
	_, err := transport.RoundTrip(req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

type stringsReader string

func (s stringsReader) Read(p []byte) (int, error) {
	if len(s) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s)
	return n, nil
}

func (s stringsReader) Close() error { return nil }
