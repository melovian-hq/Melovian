// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package pcmsink

import (
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// nopCloser wraps a writer we must not close, such as os.Stdout.
type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// writerSink forwards PCM to a plain io.WriteCloser.
type writerSink struct {
	w    io.WriteCloser
	spec string
}

func (s writerSink) Write(p []byte) (int, error) { return s.w.Write(p) }
func (s writerSink) Close() error                { return s.w.Close() }
func (s writerSink) String() string              { return s.spec }

// dialSink writes to a connection-oriented target (FIFO file, TCP dial, Unix
// dial) and reconnects when the peer goes away. Writes during an outage are
// dropped after the retry floor elapses.
type dialSink struct {
	spec     string
	open     func() (io.WriteCloser, error)
	mu       sync.Mutex
	wc       io.WriteCloser
	nextTry  time.Time
	retryMin time.Duration
	closed   bool
	lastErr  atomic.Value
}

func newDialSink(spec, network, address string) *dialSink {
	return &dialSink{
		spec: spec,
		open: func() (io.WriteCloser, error) {
			conn, err := net.DialTimeout(network, address, 2*time.Second)
			if err != nil {
				return nil, err
			}
			return &deadlineWriter{conn: conn, timeout: 500 * time.Millisecond}, nil
		},
		retryMin: 500 * time.Millisecond,
	}
}

// deadlineWriter bounds each write so a stalled peer cannot block a sink
// worker indefinitely.
type deadlineWriter struct {
	conn    net.Conn
	timeout time.Duration
}

func (w *deadlineWriter) Write(p []byte) (int, error) {
	_ = w.conn.SetWriteDeadline(time.Now().Add(w.timeout))
	return w.conn.Write(p)
}

func (w *deadlineWriter) Close() error { return w.conn.Close() }

func (s *dialSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, errors.New("pcmsink: sink closed")
	}
	if s.wc == nil {
		if time.Now().Before(s.nextTry) {
			return 0, errSinkDown
		}
		wc, err := s.open()
		if err != nil {
			s.nextTry = time.Now().Add(s.retryMin)
			s.lastErr.Store(err.Error())
			return 0, err
		}
		s.wc = wc
		s.lastErr.Store("")
	}
	n, err := s.wc.Write(p)
	if err != nil {
		s.lastErr.Store(err.Error())
		_ = s.wc.Close()
		s.wc = nil
		s.nextTry = time.Now().Add(s.retryMin)
	}
	return n, err
}

func (s *dialSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.wc != nil {
		err := s.wc.Close()
		s.wc = nil
		return err
	}
	return nil
}

func (s *dialSink) String() string { return s.spec }

func (s *dialSink) statusErr() string {
	if v, ok := s.lastErr.Load().(string); ok {
		return v
	}
	return ""
}

var errSinkDown = errors.New("pcmsink: waiting for peer")

// listenSink accepts connections on a TCP or Unix listener and streams PCM to
// every connected client. A stalled client is dropped after the write deadline
// instead of blocking the whole fanout.
type listenSink struct {
	spec     string
	ln       net.Listener
	mu       sync.Mutex
	conns    map[net.Conn]struct{}
	deadline time.Duration
	closed   chan struct{}
	wg       sync.WaitGroup
	lastErr  atomic.Value
}

func newListenSink(spec, network, address string) (*listenSink, error) {
	if network == "unix" {
		_ = removeSocketFile(address)
	}
	ln, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	s := &listenSink{
		spec:     spec,
		ln:       ln,
		conns:    make(map[net.Conn]struct{}),
		deadline: 250 * time.Millisecond,
		closed:   make(chan struct{}),
	}
	s.wg.Add(1)
	go s.acceptLoop()
	return s, nil
}

func (s *listenSink) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
			}
			s.lastErr.Store(err.Error())
			return
		}
		s.mu.Lock()
		if s.conns != nil {
			s.conns[conn] = struct{}{}
		} else {
			_ = conn.Close()
		}
		s.mu.Unlock()
	}
}

func (s *listenSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.conns {
		_ = conn.SetWriteDeadline(time.Now().Add(s.deadline))
		if _, err := conn.Write(p); err != nil {
			delete(s.conns, conn)
			_ = conn.Close()
			s.lastErr.Store(err.Error())
		}
	}
	return len(p), nil
}

func (s *listenSink) Close() error {
	close(s.closed)
	err := s.ln.Close()
	s.wg.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.conns = nil
	return err
}

func (s *listenSink) String() string { return s.spec }

func (s *listenSink) statusErr() string {
	if v, ok := s.lastErr.Load().(string); ok {
		return v
	}
	return ""
}
