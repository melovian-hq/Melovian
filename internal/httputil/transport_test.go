// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2606:4700:4700::1111", true},
		{"127.0.0.1", false},
		{"::1", false},
		{"10.0.0.5", false},
		{"172.16.0.1", false},
		{"192.168.1.1", false},
		{"169.254.169.254", false},
		{"fe80::1", false},
		{"fd00::1", false},
		{"0.0.0.0", false},
		{"224.0.0.1", false},
		{"255.255.255.255", false},
	}
	for _, tc := range cases {
		ip := net.ParseIP(tc.addr)
		if ip == nil {
			t.Fatalf("unparseable test address %s", tc.addr)
		}
		if got := IsPublicIP(ip); got != tc.want {
			t.Errorf("IsPublicIP(%s) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

func TestPublicOnlyTransportRejectsLiteralPrivateHost(t *testing.T) {
	tr := PublicOnlyTransport()
	_, err := tr.DialContext(t.Context(), "tcp", "169.254.169.254:80")
	if err == nil {
		t.Fatal("expected refusal for link-local metadata address")
	}
	_, err = tr.DialContext(t.Context(), "tcp", "127.0.0.1:80")
	if err == nil {
		t.Fatal("expected refusal for loopback")
	}
}
