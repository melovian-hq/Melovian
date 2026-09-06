# PCM audio output

Melovian sends raw PCM from mpv into a Go `pcmsink` pipeline. The pipeline can route to the system audio device, a process, a FIFO, or a network socket.

## Stream format

Fixed little-endian signed 16-bit stereo at 48 kHz. `BytesPerSec` is 192000.

## Key files

| File | Role |
|---|---|
| `internal/pcmsink/sink.go` | `Sink` interface and `New(spec)` factory. Lists all supported specs. |
| `internal/pcmsink/device.go` | System audio via `oto/v3`. |
| `internal/pcmsink/sinks.go` | `dialSink` (reconnecting FIFO, TCP, or Unix) and `listenSink` (TCP or Unix listener). |
| `internal/pcmsink/fanout.go` | Multicasts PCM chunks to every registered sink with bounded queues. |
| `internal/pcmsink/router.go` | Reads from per-player FIFOs, mixes concurrent streams, feeds the fanout. |
| `internal/pcmsink/fifo_unix.go` | Unix-only FIFO helpers. Build tag `//go:build unix`. |

## Supported sink specs

- `device`
- `stdout`
- `fifo:<path>`
- `tcp:<host:port>`
- `tcp-listen:<addr>`
- `unix:<path>`
- `unix-listen:<path>`

## Runtime behavior

- `Router` creates one FIFO per source. Concurrent players do not interleave bytes.
- `Fanout` gives each sink a 256 KB ring buffer. It counts dropped chunks when a sink falls behind.
- `dialSink` reconnects 500 ms after a peer goes away.
- `listenSink` drops clients that stall on a write.
- `mixS16` clamps mixed samples to `int16` and passes extra frames from the longest chunk through unchanged.
