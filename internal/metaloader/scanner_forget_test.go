// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"sync"
	"testing"
)

func TestScannerForgetLibraryClearsMaps(t *testing.T) {
	s := &Scanner{}
	_ = s.libraryScanMu("lib_1")
	s.setProgress("lib_1", "scan", 3)

	s.ForgetLibrary("lib_1")

	if _, ok := s.Progress("lib_1"); ok {
		t.Fatal("expected progress cleared")
	}
	if _, ok := s.scanMu.Load("lib_1"); ok {
		t.Fatal("expected scan mutex cleared")
	}
}

func TestScannerForgetLibraryConcurrentSafe(t *testing.T) {
	s := &Scanner{}
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_ = s.libraryScanMu("lib")
			s.setProgress("lib", "scan", 1)
			s.ForgetLibrary("lib")
		})
	}
	wg.Wait()
}
