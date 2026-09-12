// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"errors"
	"net/url"
	"strings"
)

var (
	// ErrInvalidURL marks input that does not parse into a URL with a host.
	ErrInvalidURL = errors.New("url is not valid")
	// ErrURLScheme marks input whose scheme is not http or https.
	ErrURLScheme = errors.New("url must use http or https")
	// ErrURLUserinfo marks input with embedded credentials, which would end
	// up stored or logged.
	ErrURLUserinfo = errors.New("credentials in the url are not allowed")
)

// ParseHTTPURL parses raw and requires an http or https scheme and a host.
// Callers that store the URL should also reject userinfo so credentials do
// not end up persisted inside a URL.
func ParseHTTPURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrInvalidURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrURLScheme
	}
	if u.Host == "" {
		return nil, ErrInvalidURL
	}
	return u, nil
}

// NormalizeHTTPURL is ParseHTTPURL plus it drops the query and fragment and
// any trailing slash, for storing a clean base URL.
func NormalizeHTTPURL(raw string) (string, error) {
	u, err := ParseHTTPURL(raw)
	if err != nil {
		return "", err
	}
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}
