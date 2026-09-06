// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package libvlc

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var errUnavailable = errors.New("libvlc not available")

var (
	loadOnce  sync.Once
	loadErr   error
	libHandle uintptr

	fnNew              func(argc int32, argv **byte) uintptr
	fnRelease          func(instance uintptr)
	fnMediaNewLocation func(instance uintptr, mrl string) uintptr
	fnMediaRelease     func(media uintptr)
	fnPlayerNew        func(instance uintptr) uintptr
	fnPlayerRelease    func(player uintptr)
	fnPlayerSetMedia   func(player, media uintptr)
	fnPlayerPlay       func(player uintptr) int32
	fnPlayerPause      func(player uintptr, doPause int32)
	fnPlayerSetTime    func(player uintptr, time int64) int32
	fnPlayerGetTime    func(player uintptr) int64
	fnPlayerGetLength  func(player uintptr) int64
	fnPlayerGetState   func(player uintptr) int32
	fnAudioSetVolume   func(player uintptr, volume int32) int32
)

const (
	statePlaying = 3
	statePaused  = 4
	stateEnded   = 6
	stateError   = 7
)

// Available reports whether libvlc can be loaded on this system.
func Available() bool {
	return ensureLoaded() == nil
}

func ensureLoaded() error {
	loadOnce.Do(func() {
		loadErr = tryLoadLibrary()
		if loadErr != nil {
			return
		}
		registerFuncs()
	})
	return loadErr
}

func registerFuncs() {
	purego.RegisterLibFunc(&fnNew, libHandle, "libvlc_new")
	purego.RegisterLibFunc(&fnRelease, libHandle, "libvlc_release")
	purego.RegisterLibFunc(&fnMediaNewLocation, libHandle, "libvlc_media_new_location")
	purego.RegisterLibFunc(&fnMediaRelease, libHandle, "libvlc_media_release")
	purego.RegisterLibFunc(&fnPlayerNew, libHandle, "libvlc_media_player_new")
	purego.RegisterLibFunc(&fnPlayerRelease, libHandle, "libvlc_media_player_release")
	purego.RegisterLibFunc(&fnPlayerSetMedia, libHandle, "libvlc_media_player_set_media")
	purego.RegisterLibFunc(&fnPlayerPlay, libHandle, "libvlc_media_player_play")
	purego.RegisterLibFunc(&fnPlayerPause, libHandle, "libvlc_media_player_pause")
	purego.RegisterLibFunc(&fnPlayerSetTime, libHandle, "libvlc_media_player_set_time")
	purego.RegisterLibFunc(&fnPlayerGetTime, libHandle, "libvlc_media_player_get_time")
	purego.RegisterLibFunc(&fnPlayerGetLength, libHandle, "libvlc_media_player_get_length")
	purego.RegisterLibFunc(&fnPlayerGetState, libHandle, "libvlc_media_player_get_state")
	purego.RegisterLibFunc(&fnAudioSetVolume, libHandle, "libvlc_audio_set_volume")
}

func vlcArgs(options []string) (**byte, [][]byte) {
	backing := make([][]byte, len(options))
	ptrs := make([]*byte, len(options)+1)
	for i, opt := range options {
		backing[i] = append([]byte(opt), 0)
		ptrs[i] = &backing[i][0]
	}
	return (**byte)(unsafe.Pointer(&ptrs[0])), backing //#nosec G103 -- required for libvlc dynamic calls
}
