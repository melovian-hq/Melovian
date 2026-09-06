// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"melovian/internal/extensions"
	"melovian/internal/httputil"
	"melovian/internal/lyrics"
)

// lyricsWhisperExtensionID is the bundled extension that gates whisper
// transcription. The endpoint refuses to run while it is disabled.
const lyricsWhisperExtensionID = "lyrics-whisper"

const whisperMaxAudioBytes = 128 << 20

type whisperSegmentsRequest struct {
	Artist      string                  `json:"artist"`
	Title       string                  `json:"title"`
	Album       string                  `json:"album"`
	DurationSec int                     `json:"durationSec"`
	Language    string                  `json:"language"`
	Segments    []lyrics.WhisperSegment `json:"segments"`
}

// handleWhisperTrackLyrics generates synced lyrics for a track. Two modes:
//
//   - application/json with a "segments" array: a client-side whisper engine
//     (WASM) already transcribed the audio and posts timed lines to store.
//   - multipart form or raw audio body: the server forwards the audio to the
//     whisper.cpp-compatible server from lyrics settings (whisperUrl).
func (s *Server) handleWhisperTrackLyrics(w http.ResponseWriter, r *http.Request) {
	if !extensions.IsEnabled(s.cfg.DataDir, lyricsWhisperExtensionID) {
		writeLyricsError(w, http.StatusForbidden, "lyrics-whisper extension is not enabled")
		return
	}
	trackID := strings.TrimSpace(r.PathValue("trackId"))
	if trackID == "" {
		writeLyricsError(w, http.StatusBadRequest, "track id required")
		return
	}
	userID := ResolveProgressUserID(r.Context())
	settings, err := s.loadLyricsSettings(userID)
	if err != nil {
		writeLyricsError(w, http.StatusInternalServerError, "failed to load lyrics settings")
		return
	}
	in := lyricsFetchInputFromQuery(r, trackID)

	var doc *lyrics.Document
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "application/json") {
		doc, err = s.whisperFromSegments(r, in)
	} else {
		doc, err = s.whisperFromAudio(w, r, in, settings)
	}
	if err != nil {
		writeLyricsError(w, http.StatusBadGateway, err.Error())
		return
	}

	root := s.lyricsRoot(settings)
	instanceID := InstanceIDFromContext(r.Context())
	store := lyrics.NewStore(root)
	if saveErr := store.Save(root, instanceID, in.TrackID, doc); saveErr != nil {
		writeLyricsError(w, http.StatusInternalServerError, "save lyrics: "+saveErr.Error())
		return
	}
	writeLyricsDocument(w, doc)
}

func (s *Server) whisperFromSegments(r *http.Request, in lyrics.FetchInput) (*lyrics.Document, error) {
	var req whisperSegmentsRequest
	if err := httputil.DecodeJSONBody(r, &req); err != nil {
		return nil, err
	}
	if in.Artist == "" {
		in.Artist = strings.TrimSpace(req.Artist)
	}
	if in.Title == "" {
		in.Title = strings.TrimSpace(req.Title)
	}
	if in.Album == "" {
		in.Album = strings.TrimSpace(req.Album)
	}
	if in.DurationSec == 0 && req.DurationSec > 0 {
		in.DurationSec = req.DurationSec
	}
	return lyrics.DocumentFromWhisper(in, req.Segments)
}

func (s *Server) whisperFromAudio(w http.ResponseWriter, r *http.Request, in lyrics.FetchInput, settings LyricsSettings) (*lyrics.Document, error) {
	serverURL, err := lyrics.WhisperServerURL(settings.WhisperURL)
	if err != nil {
		return nil, err
	}
	r.Body = http.MaxBytesReader(w, r.Body, whisperMaxAudioBytes)

	var audio []byte
	var filename string
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(contentType, "multipart/") {
		if err := r.ParseMultipartForm(whisperMaxAudioBytes); err != nil {
			return nil, err
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			file, header, err = r.FormFile("audio")
		}
		if err != nil {
			return nil, err
		}
		defer file.Close()
		filename = header.Filename
		if in.Artist == "" {
			in.Artist = strings.TrimSpace(r.FormValue("artist"))
		}
		if in.Title == "" {
			in.Title = strings.TrimSpace(r.FormValue("title"))
		}
		if in.Album == "" {
			in.Album = strings.TrimSpace(r.FormValue("album"))
		}
		audio, err = io.ReadAll(io.LimitReader(file, whisperMaxAudioBytes))
		if err != nil {
			return nil, err
		}
	} else {
		audio, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		filename = "audio.wav"
	}
	if len(audio) == 0 {
		return nil, errors.New("whisper: no audio received")
	}

	language := strings.TrimSpace(r.URL.Query().Get("language"))
	ctx, cancel := context.WithTimeout(r.Context(), lyrics.WhisperRequestTimeout)
	defer cancel()
	client := &http.Client{Timeout: lyrics.WhisperRequestTimeout, Transport: httputil.APITransport()}
	segments, err := lyrics.TranscribeWithWhisperServer(ctx, client, serverURL, filename, audio, language)
	if err != nil {
		return nil, err
	}
	return lyrics.DocumentFromWhisper(in, segments)
}
