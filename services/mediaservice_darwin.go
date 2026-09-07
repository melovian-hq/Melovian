// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build darwin && !ios

package services

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework MediaPlayer -framework Foundation -framework AppKit
#include "mediakeys_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	darwinMediaSvc *MediaService
	darwinMediaMu  sync.Mutex
)

//export goMediaKeyPlay
func goMediaKeyPlay() {
	darwinMediaMu.Lock()
	svc := darwinMediaSvc
	darwinMediaMu.Unlock()
	if svc != nil {
		svc.onPlay()
	}
}

//export goMediaKeyPause
func goMediaKeyPause() {
	darwinMediaMu.Lock()
	svc := darwinMediaSvc
	darwinMediaMu.Unlock()
	if svc != nil {
		svc.onPause()
	}
}

//export goMediaKeyToggle
func goMediaKeyToggle() {
	darwinMediaMu.Lock()
	svc := darwinMediaSvc
	darwinMediaMu.Unlock()
	if svc != nil {
		svc.TogglePlayback()
	}
}

//export goMediaKeyNext
func goMediaKeyNext() {
	darwinMediaMu.Lock()
	svc := darwinMediaSvc
	darwinMediaMu.Unlock()
	if svc != nil {
		svc.onNext()
	}
}

//export goMediaKeyPrev
func goMediaKeyPrev() {
	darwinMediaMu.Lock()
	svc := darwinMediaSvc
	darwinMediaMu.Unlock()
	if svc != nil {
		svc.onPrevious()
	}
}

//export goMediaKeySeekTo
func goMediaKeySeekTo(positionSec C.double) {
	darwinMediaMu.Lock()
	svc := darwinMediaSvc
	darwinMediaMu.Unlock()
	if svc != nil {
		svc.queueSeek(int64(float64(positionSec) * 1000))
	}
}

type nowPlayingController struct {
	svc         *MediaService
	mu          sync.Mutex
	lastState   PlaybackState
	initialized bool
}

func newPlatformMediaController(svc *MediaService) (MediaController, error) {
	darwinMediaMu.Lock()
	darwinMediaSvc = svc
	darwinMediaMu.Unlock()

	ctrl := &nowPlayingController{svc: svc}
	application.InvokeSync(func() {
		C.MediaKeysInit()
		ctrl.initialized = true
	})
	return ctrl, nil
}

func (c *nowPlayingController) metadataChanged(state PlaybackState) bool {
	return state.Title != c.lastState.Title ||
		state.Artist != c.lastState.Artist ||
		state.Album != c.lastState.Album ||
		state.CoverArtURL != c.lastState.CoverArtURL ||
		state.DurationMs != c.lastState.DurationMs ||
		state.Playing != c.lastState.Playing ||
		state.CanGoNext != c.lastState.CanGoNext ||
		state.CanGoPrevious != c.lastState.CanGoPrevious ||
		state.TrackID != c.lastState.TrackID
}

func (c *nowPlayingController) Update(state PlaybackState) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil
	}

	positionOnly := state.TrackID != "" &&
		state.TrackID == c.lastState.TrackID &&
		!c.metadataChanged(state)

	if state.TrackID == "" {
		application.InvokeSync(func() {
			C.MediaKeysClose()
			C.MediaKeysInit()
		})
		c.lastState = state
		return nil
	}

	application.InvokeSync(func() {
		if positionOnly {
			C.MediaKeysUpdatePosition(
				C.double(float64(state.PositionMs)/1000),
				C.bool(state.Playing),
			)
			return
		}

		var cTitle, cArtist, cAlbum, cArtwork *C.char
		if state.Title != "" {
			cTitle = C.CString(state.Title)
			defer C.free(unsafe.Pointer(cTitle))
		}
		if state.Artist != "" {
			cArtist = C.CString(state.Artist)
			defer C.free(unsafe.Pointer(cArtist))
		}
		if state.Album != "" {
			cAlbum = C.CString(state.Album)
			defer C.free(unsafe.Pointer(cAlbum))
		}
		if state.CoverArtURL != "" {
			cArtwork = C.CString(state.CoverArtURL)
			defer C.free(unsafe.Pointer(cArtwork))
		}

		C.MediaKeysUpdateNowPlaying(
			cTitle,
			cArtist,
			cAlbum,
			cArtwork,
			C.double(float64(state.DurationMs)/1000),
			C.double(float64(state.PositionMs)/1000),
			C.bool(state.Playing),
			C.bool(state.CanGoNext),
			C.bool(state.CanGoPrevious),
		)
	})

	c.lastState = state
	return nil
}

func (c *nowPlayingController) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return nil
	}

	application.InvokeSync(func() {
		C.MediaKeysClose()
	})
	c.initialized = false
	c.lastState = PlaybackState{}

	darwinMediaMu.Lock()
	if darwinMediaSvc == c.svc {
		darwinMediaSvc = nil
	}
	darwinMediaMu.Unlock()

	return nil
}
