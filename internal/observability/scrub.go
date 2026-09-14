// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"net/url"
	"strings"

	"github.com/getsentry/sentry-go"
)

// sensitiveQueryKeys are stripped from any URL before it reaches a report.
// They cover Subsonic auth (u, p, t, s), bearer-style credentials, and the
// DSN itself so a misconfigured sink never receives secrets back.
var sensitiveQueryKeys = map[string]struct{}{
	"u": {}, "p": {}, "t": {}, "s": {},
	"token": {}, "access_token": {}, "refresh_token": {}, "id_token": {},
	"password": {}, "passwd": {}, "secret": {}, "client_secret": {},
	"key": {}, "apikey": {}, "api_key": {}, "auth": {}, "authorization": {},
	"sid": {}, "session": {}, "sessionid": {}, "jwt": {}, "signature": {},
	"sig": {}, "dsn": {},
}

// sensitiveHeaders are removed from request data in reports.
var sensitiveHeaders = map[string]struct{}{
	"authorization": {}, "cookie": {}, "set-cookie": {},
	"proxy-authorization": {}, "x-api-key": {}, "x-auth-token": {},
	"x-subsonic-auth": {}, "x-melovian-token": {},
}

const redactedValue = "[redacted]"

// ScrubURL drops the userinfo and redacts sensitive query parameters from a
// URL string. Unparseable URLs are dropped entirely.
func ScrubURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.User = nil
	u.RawQuery = scrubQuery(u.Query()).Encode()
	u.Fragment = ""
	return u.String()
}

func scrubQuery(values url.Values) url.Values {
	for key := range values {
		if _, sensitive := sensitiveQueryKeys[strings.ToLower(key)]; sensitive {
			values.Set(key, redactedValue)
		}
	}
	return values
}

func scrubHeaderMap(headers map[string]string) map[string]string {
	for key := range headers {
		if _, sensitive := sensitiveHeaders[strings.ToLower(key)]; sensitive {
			delete(headers, key)
		}
	}
	return headers
}

func scrubRequest(req *sentry.Request) {
	if req == nil {
		return
	}
	req.URL = ScrubURL(req.URL)
	req.QueryString = ""
	req.Cookies = ""
	req.Data = ""
	if req.Headers != nil {
		req.Headers = scrubHeaderMap(req.Headers)
	}
}

// ScrubEvent removes request secrets, cookies, and credentials from a Sentry
// event before it leaves the process.
func ScrubEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	scrubRequest(event.Request)
	for i := range event.Breadcrumbs {
		if event.Breadcrumbs[i].Data != nil {
			if raw, ok := event.Breadcrumbs[i].Data["url"].(string); ok {
				event.Breadcrumbs[i].Data["url"] = ScrubURL(raw)
			}
		}
	}
	event.User = sentry.User{ID: event.User.ID}
	return event
}

// ScrubBreadcrumb scrubs URL data on breadcrumbs before they are recorded.
func ScrubBreadcrumb(crumb *sentry.Breadcrumb, _ *sentry.BreadcrumbHint) *sentry.Breadcrumb {
	if crumb == nil {
		return nil
	}
	if crumb.Data != nil {
		if raw, ok := crumb.Data["url"].(string); ok {
			crumb.Data["url"] = ScrubURL(raw)
		}
	}
	return crumb
}
