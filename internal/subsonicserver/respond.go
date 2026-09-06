// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonicserver

import (
	"encoding/json"
	"encoding/xml"
	"maps"
	"net/http"
	"strings"
)

func wantsJSON(r *http.Request) bool {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("f")))
	return format == "json" || format == "jsonp"
}

func writeOK(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	maps.Copy(payload, baseOK())
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"subsonic-response": payload})
		return
	}
	writeXMLResponse(w, payload)
}

func writeXMLResponse(w http.ResponseWriter, payload map[string]any) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	type response struct {
		XMLName       xml.Name `xml:"subsonic-response"`
		Status        string   `xml:"status,attr"`
		Version       string   `xml:"version,attr"`
		Type          string   `xml:"type,attr"`
		ServerVersion string   `xml:"serverVersion,attr"`
		OpenSubsonic  string   `xml:"openSubsonic,attr"`
		Rest          xmlRest  `xml:",any"`
	}
	rest := mapToXMLRest(payload)
	out := response{
		Status:        "ok",
		Version:       Version,
		Type:          ServerType,
		ServerVersion: ServerName,
		OpenSubsonic:  OpenSubsonic,
		Rest:          rest,
	}
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	_ = enc.Encode(out)
}

type xmlRest struct {
	Items []xmlRestItem
}

type xmlRestItem struct {
	XMLName xml.Name
	Attrs   []xml.Attr    `xml:",any,attr"`
	Inner   string        `xml:",chardata"`
	Child   []xmlRestItem `xml:",any"`
}

func mapToXMLRest(payload map[string]any) xmlRest {
	skip := map[string]bool{
		"status": true, "version": true, "type": true,
		"serverVersion": true, "openSubsonic": true,
	}
	var items []xmlRestItem
	for key, value := range payload {
		if skip[key] {
			continue
		}
		items = append(items, valueToXMLItem(key, value))
	}
	return xmlRest{Items: items}
}

func valueToXMLItem(name string, value any) xmlRestItem {
	switch typed := value.(type) {
	case map[string]any:
		item := xmlRestItem{XMLName: xml.Name{Local: name}}
		for key, child := range typed {
			item.Child = append(item.Child, valueToXMLItem(key, child))
		}
		return item
	case []map[string]any:
		item := xmlRestItem{XMLName: xml.Name{Local: name}}
		for _, child := range typed {
			for key, grandchild := range child {
				item.Child = append(item.Child, valueToXMLItem(key, grandchild))
			}
		}
		return item
	case string:
		return xmlRestItem{XMLName: xml.Name{Local: name}, Inner: typed}
	case int:
		return xmlRestItem{
			XMLName: xml.Name{Local: name},
			Attrs:   []xml.Attr{{Name: xml.Name{Local: name}, Value: itoa(typed)}},
		}
	case bool:
		value := "false"
		if typed {
			value = "true"
		}
		return xmlRestItem{
			XMLName: xml.Name{Local: name},
			Attrs:   []xml.Attr{{Name: xml.Name{Local: name}, Value: value}},
		}
	default:
		return xmlRestItem{XMLName: xml.Name{Local: name}}
	}
}

func writeError(w http.ResponseWriter, r *http.Request, code int, message string) {
	payload := map[string]any{
		"status":  "failed",
		"version": Version,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"subsonic-response": payload})
		return
	}
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write([]byte(`<subsonic-response status="failed" version="` + Version + `">`))
	_, _ = w.Write([]byte(`<error code="` + itoa(code) + `" message="` + xmlEscape(message) + `"/>`)) //#nosec G705 -- message is xmlEscape'd
	_, _ = w.Write([]byte(`</subsonic-response>`))
}

func baseOK() map[string]any {
	return map[string]any{
		"status":        "ok",
		"version":       Version,
		"type":          ServerType,
		"serverVersion": ServerName,
		"openSubsonic":  OpenSubsonic,
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [16]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
		`'`, "&apos;",
	)
	return replacer.Replace(value)
}
