// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"testing"
	"time"
)

// Repeated startCheck calls must not stack concurrent checks: a second call
// while one is running is a no-op.
func TestUpdateServiceStartCheckDedup(t *testing.T) {
	svc := NewUpdateService()
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	check := func(ctx context.Context) {
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
	}

	svc.startCheck(check)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("check did not start")
	}

	svc.startCheck(check)
	svc.startCheck(check)
	select {
	case <-started:
		t.Fatal("duplicate check spawned while one was running")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	svc.Shutdown()
}

// Shutdown cancels the in-flight check and waits for the goroutine to exit.
func TestUpdateServiceShutdownCancelsCheck(t *testing.T) {
	svc := NewUpdateService()
	started := make(chan struct{})
	exited := make(chan struct{})
	svc.startCheck(func(ctx context.Context) {
		close(started)
		<-ctx.Done()
		close(exited)
	})
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("check did not start")
	}

	svc.Shutdown()
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("check goroutine was not cancelled by Shutdown")
	}
}

// After Shutdown, startCheck must not spawn new work.
func TestUpdateServiceStartCheckAfterShutdown(t *testing.T) {
	svc := NewUpdateService()
	svc.Shutdown()

	ran := make(chan struct{}, 1)
	svc.startCheck(func(context.Context) { ran <- struct{}{} })
	select {
	case <-ran:
		t.Fatal("check ran after Shutdown")
	case <-time.After(100 * time.Millisecond):
	}
}
