// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var jsonBufPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 512))
	},
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	buf := jsonBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer jsonBufPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func QueryInt(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

const MaxJSONBody = 1 << 20

func QueryIntClamped(r *http.Request, key string, fallback, min, max int) int {
	n := QueryInt(r, key, fallback)
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func ReadLimited(r io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("body too large")
	}
	return data, nil
}

func ReadJSONBytes(r *http.Request) ([]byte, error) {
	defer func() { _ = r.Body.Close() }()
	return ReadLimited(r.Body, MaxJSONBody)
}

func DecodeJSONBody[T any](r *http.Request, dst *T) error {
	data, err := ReadJSONBytes(r)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("empty request body")
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

func CopyHeaders(dst, src http.Header) {
	for key, values := range src {
		if IsHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func IsHopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailers", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}
