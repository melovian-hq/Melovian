// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	apiTransportOnce sync.Once
	apiTransport     http.RoundTripper

	streamTransportOnce sync.Once
	streamTransport     http.RoundTripper
)

func APITransport() http.RoundTripper {
	apiTransportOnce.Do(func() {
		apiTransport = newPooledTransport(32, 64)
	})
	return apiTransport
}

func StreamTransport() http.RoundTripper {
	streamTransportOnce.Do(func() {
		streamTransport = newPooledTransport(8, 16)
	})
	return streamTransport
}

func newPooledTransport(idlePerHost, maxPerHost int) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          128,
		MaxIdleConnsPerHost:   idlePerHost,
		MaxConnsPerHost:       maxPerHost,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
