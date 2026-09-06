// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func FuzzPingResponse(f *testing.F) {
	f.Add([]byte(`{"subsonic-response":{"status":"ok","version":"1.16.1","type":"navidrome","serverVersion":"0.58.0"}}`))
	f.Add([]byte(`{"subsonic-response":{"status":"ok","version":"1.16.1","server":{"name":"Navidrome","version":"0.54.0"}}}`))
	f.Add([]byte(`{"subsonic-response":{"status":"failed","error":{"code":40,"message":"Wrong password"}}}`))
	f.Add([]byte(`<subsonic-response status="ok" type="navidrome" serverVersion="0.58.0"></subsonic-response>`))
	f.Add([]byte(`<subsonic-response status="failed"><error code="40" message="nope"/></subsonic-response>`))
	f.Add([]byte(`not-json-or-xml`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, body []byte) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/rest/ping.view") {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(body)
		}))
		t.Cleanup(srv.Close)

		client := NewClient(srv.URL, "alice", "secret")
		name, version, err := client.Ping()
		if err != nil {
			if name != "" || version != "" {
				t.Fatalf("error path must return empty identity: name=%q version=%q err=%v", name, version, err)
			}
			return
		}
		if strings.TrimSpace(name) == "" {
			t.Fatal("successful ping must return a non-empty server name")
		}
		// Successful parse must be consistent with the OpenSubsonic / legacy shapes.
		var asJSON map[string]any
		if json.Unmarshal(body, &asJSON) == nil {
			resp, _ := asJSON["subsonic-response"].(map[string]any)
			if resp != nil {
				if status, _ := resp["status"].(string); status != "" && status != "ok" {
					t.Fatalf("non-ok JSON status should error, got name=%q", name)
				}
			}
		}
	})
}
