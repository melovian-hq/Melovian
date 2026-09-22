// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"context"
	"fmt"
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

	publicTransportOnce sync.Once
	publicTransport     *http.Transport
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

// PublicOnlyTransport dials only public addresses. The check happens at
// connect time, so redirects and DNS answers that point at loopback,
// private, or link-local targets (cloud metadata endpoints, LAN hosts)
// are refused even though the request URL itself looked fine.
func PublicOnlyTransport() *http.Transport {
	publicTransportOnce.Do(func() {
		publicTransport = newPublicOnlyTransport()
	})
	return publicTransport
}

func newPublicOnlyTransport() *http.Transport {
	t := newPooledTransport(8, 16)
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !IsPublicIP(ip) {
				return nil, fmt.Errorf("refusing non-public address for %s", host)
			}
		}
		return dialer.DialContext(ctx, network, addr)
	}
	return t
}

// IsPublicIP reports whether ip is a globally routable unicast address.
// IsGlobalUnicast alone still accepts RFC1918 and link-local ranges, so
// those are excluded explicitly.
func IsPublicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() &&
		!ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsMulticast()
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
