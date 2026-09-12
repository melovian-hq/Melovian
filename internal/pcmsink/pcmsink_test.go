// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package pcmsink

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type bufferSink struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	spec string
}

func (b *bufferSink) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *bufferSink) Close() error   { return nil }
func (b *bufferSink) String() string { return b.spec }
func (b *bufferSink) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

type blockingSink struct{ release chan struct{} }

func (b *blockingSink) Write(p []byte) (int, error) {
	<-b.release
	return len(p), nil
}
func (b *blockingSink) Close() error   { return nil }
func (b *blockingSink) String() string { return "blocking" }

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

func TestFanoutBroadcast(t *testing.T) {
	buf := &bufferSink{spec: "test"}
	f := &Fanout{workers: make(map[string]*sinkWorker)}
	f.add("test", buf)
	defer f.Close()

	payload := bytes.Repeat([]byte{1, 0, 2, 0}, 256)
	f.Write(payload)
	waitFor(t, func() bool { return buf.Len() >= len(payload) })
}

func TestFanoutDropsWhenSinkStalls(t *testing.T) {
	sink := &blockingSink{release: make(chan struct{})}
	f := &Fanout{workers: make(map[string]*sinkWorker)}
	f.add("blocking", sink)
	defer func() {
		close(sink.release)
		f.Close()
	}()

	chunk := make([]byte, chunkBytes)
	for range sinkQueueChunks + 10 {
		f.Write(chunk)
	}
	waitFor(t, func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.workers["blocking"].dropped.Load() > 0
	})
}

func TestTCPSinkDial(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(io.Discard, conn)
	}()

	sink, err := New("tcp:" + ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	if _, err := sink.Write([]byte{1, 2, 3, 4}); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestUnixListenSink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.sock")
	sink, err := New("unix-listen:" + path)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	payload := []byte{9, 8, 7, 6}
	waitFor(t, func() bool {
		if _, err := sink.Write(payload); err != nil {
			t.Fatalf("write: %v", err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		got := make([]byte, len(payload))
		_, err := io.ReadFull(conn, got)
		return err == nil && bytes.Equal(got, payload)
	})
}

func TestFIFOSink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.fifo")
	sink, err := New("fifo:" + path)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatal("expected a named pipe")
	}

	r, err := openFIFOReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	payload := []byte{1, 2, 3, 4}
	waitFor(t, func() bool {
		if _, err := sink.Write(payload); err != nil {
			return false // no reader attached yet per nonblocking semantics
		}
		got := make([]byte, len(payload))
		n, _ := r.Read(got)
		return n == len(payload) && bytes.Equal(got, payload)
	})
}

func TestNewSourceAndRoute(t *testing.T) {
	buf := &bufferSink{spec: "test"}
	f := &Fanout{workers: make(map[string]*sinkWorker)}
	f.add("test", buf)
	defer f.Close()

	dir := t.TempDir()
	router, err := NewRouter(filepath.Join(dir, "fifos"), f)
	if err != nil {
		t.Fatal(err)
	}
	defer router.Close()

	src, err := router.NewSource()
	if err != nil {
		t.Fatal(err)
	}

	wfd, err := unix.Open(src.Path(), unix.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(wfd)

	frame := []byte{5, 0, 5, 0}
	payload := bytes.Repeat(frame, 512)
	if _, err := unix.Write(wfd, payload); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return buf.Len() >= len(payload) })
}

func TestMixS16(t *testing.T) {
	a := []byte{1, 0, 2, 0} // samples 1, 2
	b := []byte{3, 0, 4, 0} // samples 3, 4
	out := mixS16([][]byte{a, b})
	got := []int16{
		int16(binary.LittleEndian.Uint16(out[0:])),
		int16(binary.LittleEndian.Uint16(out[2:])),
	}
	if got[0] != 4 || got[1] != 6 {
		t.Fatalf("mixS16 = %v, want [4 6]", got)
	}

	loud := []byte{0xFF, 0x7F, 0xFF, 0x7F} // +32767, +32767
	clamped := mixS16([][]byte{loud, loud})
	if got := int16(binary.LittleEndian.Uint16(clamped)); got != 32767 {
		t.Fatalf("clamp = %d, want 32767", got)
	}
}

func TestNewRejectsUnknownSpec(t *testing.T) {
	if _, err := New("bogus:thing"); err == nil {
		t.Fatal("expected error for unknown target")
	}
	if _, err := New("fifo:"); err == nil {
		t.Fatal("expected error for empty fifo path")
	}
}
