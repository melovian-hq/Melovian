// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package libmpv

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var errUnavailable = errors.New("libmpv not available")

var (
	loadOnce  sync.Once
	loadErr   error
	libHandle uintptr

	fnCreate           func() uintptr
	fnInitialize       func(handle uintptr) int32
	fnTerminateDestroy func(handle uintptr)
	fnSetOptionString  func(handle uintptr, name, value string) int32
	fnCommand          func(handle uintptr, cmd **byte) int32
	fnSetProperty      func(handle uintptr, name string, format int32, data unsafe.Pointer) int32
	fnGetProperty      func(handle uintptr, name string, format int32, data unsafe.Pointer) int32
	fnWaitEvent        func(handle uintptr, timeout float64) *cEvent
	fnErrorString      func(code int32) string
)

const (
	formatFlag   int32 = 3
	formatDouble int32 = 5

	eventEnd     uint32 = 7
	endFileEOF   uint32 = 0
	endFileError uint32 = 4
)

type cEvent struct {
	EventID       uint32
	Error         int32
	ReplyUserdata uint64
	Data          unsafe.Pointer
}

type eventEndFile struct {
	Reason          uint32
	Error           int32
	EntryID         uint64
	Bytes           uint64
	PlaylistEntryID uint64
}

// Available reports whether libmpv can be loaded on this system.
func Available() bool {
	return ensureLoaded() == nil
}

// LoadError returns the reason libmpv could not be loaded, if any.
func LoadError() error {
	return ensureLoaded()
}

func ensureLoaded() error {
	loadOnce.Do(func() {
		loadErr = tryLoadLibrary()
		if loadErr != nil {
			return
		}
		registerFuncs()
		if loadErr != nil {
			return
		}
		if fnCreate == nil {
			loadErr = errors.New("libmpv: mpv_create symbol not found")
		}
	})
	return loadErr
}

func registerFuncs() {
	purego.RegisterLibFunc(&fnCreate, libHandle, "mpv_create")
	purego.RegisterLibFunc(&fnInitialize, libHandle, "mpv_initialize")
	purego.RegisterLibFunc(&fnTerminateDestroy, libHandle, "mpv_terminate_destroy")
	purego.RegisterLibFunc(&fnSetOptionString, libHandle, "mpv_set_option_string")
	purego.RegisterLibFunc(&fnCommand, libHandle, "mpv_command")
	purego.RegisterLibFunc(&fnSetProperty, libHandle, "mpv_set_property")
	purego.RegisterLibFunc(&fnGetProperty, libHandle, "mpv_get_property")
	purego.RegisterLibFunc(&fnWaitEvent, libHandle, "mpv_wait_event")
	purego.RegisterLibFunc(&fnErrorString, libHandle, "mpv_error_string")
}

func mpvError(code int32) error {
	if code >= 0 {
		return nil
	}
	if fnErrorString != nil {
		if msg := fnErrorString(code); msg != "" {
			return fmt.Errorf("libmpv: %s (code %d)", msg, code)
		}
	}
	return fmt.Errorf("libmpv: error code %d", code)
}

func runCommand(handle uintptr, cmd ...string) error {
	args := make([]*byte, len(cmd)+1)
	for i, part := range cmd {
		args[i] = cStr(part)
	}
	return mpvError(fnCommand(handle, unsafe.SliceData(args))) //#nosec G103 -- required for libmpv dynamic calls
}

func cStr(str string) *byte {
	bs := append([]byte(str), 0)
	return &bs[0]
}
