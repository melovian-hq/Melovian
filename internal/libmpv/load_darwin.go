// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package libmpv

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

// lcNumeric is the Darwin category index for LC_NUMERIC.
const lcNumeric int32 = 4

var (
	setlocaleOnce sync.Once
	fnSetlocale   func(category int32, locale string) string
)

// prepareCreateEnv forces LC_NUMERIC to "C" before mpv_create runs, matching
// libmpv's requirement for the C numeric locale.
func prepareCreateEnv() {
	setlocaleOnce.Do(func() {
		handle, err := purego.Dlopen("libc.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			return
		}
		purego.RegisterLibFunc(&fnSetlocale, uintptr(handle), "setlocale")
	})
	if fnSetlocale != nil {
		fnSetlocale(lcNumeric, "C")
	}
}

func tryLoadLibrary() error {
	names := []string{"libmpv.dylib", "libmpv.2.dylib", "libmpv.1.dylib"}
	var lastErr error
	for _, name := range names {
		handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			lastErr = err
			continue
		}
		libHandle = uintptr(handle)
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("libmpv: %w", lastErr)
	}
	return errUnavailable
}
