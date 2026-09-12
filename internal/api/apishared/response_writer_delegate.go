// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

func DelegateHijack(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.(http.Hijacker); ok {
		return h.Hijack()
	}
	if u, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
		return DelegateHijack(u.Unwrap())
	}
	return nil, nil, fmt.Errorf("response writer does not support hijacking")
}

func DelegateFlush(w http.ResponseWriter) error {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
		return nil
	}
	if u, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
		return DelegateFlush(u.Unwrap())
	}
	return fmt.Errorf("response writer does not support flushing")
}
