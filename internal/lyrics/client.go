// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"net/http"
	"time"

	"melovian/internal/httputil"
)

const ProviderRequestTimeout = 20 * time.Second

func NewProviderHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   ProviderRequestTimeout,
		Transport: httputil.APITransport(),
	}
}
