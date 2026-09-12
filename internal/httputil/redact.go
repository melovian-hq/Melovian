// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"errors"
	"fmt"
	"net/url"
)

// SanitizeErrorURL rewrites a *url.Error so credentials embedded in a request
// URL (query tokens, userinfo) never reach logs or API responses. Other error
// types are returned unchanged.
func SanitizeErrorURL(err error) error {
	var uerr *url.Error
	if !errors.As(err, &uerr) {
		return err
	}
	u, perr := url.Parse(uerr.URL)
	if perr != nil {
		return fmt.Errorf("%s: %w", uerr.Op, uerr.Err)
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil
	return &url.Error{Op: uerr.Op, URL: u.String(), Err: uerr.Err}
}

// RedactURLUserinfo strips embedded basic auth credentials from a URL string
// before it is logged. Unparseable or credential-free values pass through.
func RedactURLUserinfo(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = nil
	return u.String()
}
