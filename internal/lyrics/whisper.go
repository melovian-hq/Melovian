// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"melovian/internal/httputil"
)

// WhisperSourceID is the Document.Source value for whisper-generated lyrics.
const WhisperSourceID = "lyrics-whisper"

// WhisperSegment is one transcribed span with a start offset.
type WhisperSegment struct {
	StartMs int    `json:"startMs"`
	Text    string `json:"text"`
}

// DocumentFromWhisper builds a synced Document from whisper segments. The
// raw value is emitted as LRC so export and sidecar writers keep working.
func DocumentFromWhisper(in FetchInput, segments []WhisperSegment) (*Document, error) {
	doc := &Document{
		Source: WhisperSourceID,
		Artist: in.Artist,
		Title:  in.Title,
		Synced: true,
	}
	var raw strings.Builder
	for _, segment := range segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		startMs := max(segment.StartMs, 0)
		start := startMs
		doc.Lines = append(doc.Lines, Line{Text: text, StartMs: &start})
		fmt.Fprintf(&raw, "[%02d:%02d.%02d] %s\n",
			startMs/60000, (startMs%60000)/1000, (startMs%1000)/10, text)
	}
	if len(doc.Lines) == 0 {
		return nil, fmt.Errorf("whisper: empty transcription")
	}
	doc.RawValue = strings.TrimRight(raw.String(), "\n")
	return NormalizeDocument(doc)
}

// whisperResponse is the whisper.cpp server verbose_json shape. Some builds
// return a "transcription" array with millisecond offsets instead, both are
// decoded below.
type whisperResponse struct {
	Segments []struct {
		Start float64 `json:"start"`
		Text  string  `json:"text"`
	} `json:"segments"`
	Transcription []struct {
		Text    string `json:"text"`
		Offsets struct {
			From int64 `json:"from"`
		} `json:"offsets"`
	} `json:"transcription"`
}

// ParseWhisperResponse extracts timed segments from a whisper.cpp server JSON
// response. It accepts the verbose_json "segments" array (float seconds) and
// the older "transcription" array (millisecond offsets).
func ParseWhisperResponse(data []byte) ([]WhisperSegment, error) {
	var payload whisperResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("whisper: decode response: %w", err)
	}
	segments := make([]WhisperSegment, 0, len(payload.Segments))
	for _, segment := range payload.Segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		segments = append(segments, WhisperSegment{
			StartMs: int(segment.Start * 1000),
			Text:    text,
		})
	}
	for _, segment := range payload.Transcription {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		segments = append(segments, WhisperSegment{
			StartMs: int(segment.Offsets.From),
			Text:    text,
		})
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("whisper: response has no segments")
	}
	return segments, nil
}

// WhisperServerURL validates and normalizes a whisper.cpp server base URL.
func WhisperServerURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("whisper server URL is not configured")
	}
	if _, err := httputil.ParseHTTPURL(raw); err != nil {
		if errors.Is(err, httputil.ErrURLScheme) {
			return "", fmt.Errorf("whisper server URL must be http or https")
		}
		return "", fmt.Errorf("whisper server URL is invalid")
	}
	return strings.TrimRight(raw, "/"), nil
}

// TranscribeWithWhisperServer posts audio to a whisper.cpp-compatible server
// (/inference endpoint) and returns timed segments. Language is optional;
// empty means auto-detect.
func TranscribeWithWhisperServer(ctx context.Context, client *http.Client, serverURL, filename string, audio []byte, language string) ([]WhisperSegment, error) {
	base, err := WhisperServerURL(serverURL)
	if err != nil {
		return nil, err
	}
	if len(audio) == 0 {
		return nil, fmt.Errorf("whisper: empty audio")
	}
	if filename == "" {
		filename = "audio.wav"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	header.Set("Content-Type", "application/octet-stream")
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(audio); err != nil {
		return nil, err
	}
	_ = writer.WriteField("response-format", "verbose_json")
	if language != "" {
		_ = writer.WriteField("language", language)
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	endpoint := base + "/inference"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("whisper request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("whisper read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		snippet := strings.TrimSpace(string(data))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return nil, fmt.Errorf("whisper server status %d: %s", resp.StatusCode, snippet)
	}
	return ParseWhisperResponse(data)
}

// WhisperRequestTimeout bounds a server-side transcription call. Whisper on
// CPU can take a while for long tracks, so this is generous on purpose.
const WhisperRequestTimeout = 10 * time.Minute
