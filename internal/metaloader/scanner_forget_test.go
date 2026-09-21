// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"sync"
	"testing"
)

func TestScannerForgetLibraryClearsMaps(t *testing.T) {
	s := &Scanner{}
	mu := s.libraryScanMu("lib_1")
	s.setProgress("lib_1", "scan", 3)

	s.ForgetLibrary("lib_1")

	if _, ok := s.Progress("lib_1"); ok {
		t.Fatal("expected progress cleared")
	}
	// The mutex entry is intentionally retained: deleting it while a scan
	// holds the lock would let a fresh mutex overlap the running scan.
	if got := s.libraryScanMu("lib_1"); got != mu {
		t.Fatal("expected the same scan mutex to be reused after forget")
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
