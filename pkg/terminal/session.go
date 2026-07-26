// Package terminal provides a transport-neutral, model-oriented terminal session.
package terminal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	DefaultBufferSize = 256 * 1024
	DefaultMaxBytes   = 16 * 1024
)

type WaitKind string

const (
	WaitNone         WaitKind = "none"
	WaitPrompt       WaitKind = "prompt"
	WaitPattern      WaitKind = "pattern"
	WaitQuiet        WaitKind = "quiet"
	WaitScreenStable WaitKind = "screen_stable"
)

type Wait struct {
	Kind    WaitKind
	Pattern string
	Quiet   time.Duration
}

// Screen is optional. It is a projection of the raw byte stream, never the
// source of truth for output delivery.
type Screen interface {
	Feed([]byte)
	Snapshot() string
	Cursor() (int, int)
	Size() (int, int)
	Resize(width, height int)
}

type Config struct {
	BufferSize int
	Screen     Screen
	OnData     func([]byte)
}

type InteractRequest struct {
	Input      []byte
	FromOffset *uint64
	Wait       Wait
	MaxBytes   int
}

type Result struct {
	State             string `json:"state"`
	StopReason        string `json:"stop_reason"`
	Data              string `json:"data"`
	Encoding          string `json:"encoding"`
	Matched           bool   `json:"matched"`
	FromOffset        uint64 `json:"from_offset"`
	NextOffset        uint64 `json:"next_offset"`
	StartOffset       uint64 `json:"start_offset"`
	EndOffset         uint64 `json:"end_offset"`
	BytesRead         int    `json:"bytes_read"`
	BytesLost         uint64 `json:"bytes_lost"`
	BufferedRemaining uint64 `json:"buffered_remaining"`
	Truncated         bool   `json:"truncated"`
	ElapsedMS         int64  `json:"elapsed_ms"`
}

type Status struct {
	StartOffset uint64 `json:"start_offset"`
	EndOffset   uint64 `json:"end_offset"`
	Closed      bool   `json:"closed"`
	ReadError   string `json:"read_error,omitempty"`
}

type Session struct {
	reader io.Reader
	writer io.Writer
	ring   *byteRing
	screen Screen
	onData func([]byte)

	interactMu sync.Mutex
	writeMu    sync.Mutex
	stateMu    sync.RWMutex
	readErr    error
	done       chan struct{}
	closeOnce  sync.Once
}

func New(reader io.Reader, writer io.Writer, config Config) *Session {
	bufferSize := config.BufferSize
	if bufferSize <= 0 {
		bufferSize = DefaultBufferSize
	}
	s := &Session{
		reader: reader,
		writer: writer,
		ring:   newByteRing(bufferSize),
		screen: config.Screen,
		onData: config.OnData,
		done:   make(chan struct{}),
	}
	go s.readLoop()
	return s
}

func (s *Session) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.reader.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			s.ring.append(data)
			if s.screen != nil {
				s.screen.Feed(data)
			}
			if s.onData != nil {
				s.onData(data)
			}
		}
		if err != nil {
			s.stateMu.Lock()
			s.readErr = err
			s.stateMu.Unlock()
			s.ring.signal()
			return
		}
		select {
		case <-s.done:
			return
		default:
		}
	}
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		close(s.done)
		s.ring.signal()
	})
}

func (s *Session) Status() Status {
	start, end := s.ring.bounds()
	s.stateMu.RLock()
	err := s.readErr
	s.stateMu.RUnlock()
	status := Status{StartOffset: start, EndOffset: end, Closed: s.isClosed() || err != nil}
	if err != nil && err != io.EOF {
		status.ReadError = err.Error()
	}
	return status
}

func (s *Session) Screen() Screen { return s.screen }

func (s *Session) Interact(ctx context.Context, request InteractRequest) (Result, error) {
	s.interactMu.Lock()
	defer s.interactMu.Unlock()

	if s.isClosed() || s.hasReadError() {
		return Result{}, fmt.Errorf("terminal session is closed")
	}

	start := s.ring.endOffset()
	if request.FromOffset != nil {
		start = *request.FromOffset
	}
	if len(request.Input) > 0 {
		// Capture the offset before writing so immediate responses belong to this turn.
		start = s.ring.endOffset()
		s.writeMu.Lock()
		_, err := s.writer.Write(request.Input)
		s.writeMu.Unlock()
		if err != nil {
			return Result{}, fmt.Errorf("write terminal input: %w", err)
		}
	}
	return s.collect(ctx, start, request.Wait, request.MaxBytes)
}

// Write sends raw bytes without waiting for a result. It exists only for
// backwards-compatible callers; model-facing callers should use Interact.
func (s *Session) Write(input []byte) error {
	if s.isClosed() || s.hasReadError() {
		return fmt.Errorf("terminal session is closed")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.writer.Write(input)
	if err != nil {
		return fmt.Errorf("write terminal input: %w", err)
	}
	return nil
}

func (s *Session) collect(ctx context.Context, from uint64, wait Wait, maxBytes int) (Result, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	started := time.Now()
	if wait.Kind == "" {
		wait.Kind = WaitNone
	}
	if (wait.Kind == WaitPrompt || wait.Kind == WaitPattern) && wait.Pattern == "" {
		return Result{}, fmt.Errorf("wait pattern is required")
	}
	if wait.Kind == WaitScreenStable && s.screen == nil {
		return Result{}, fmt.Errorf("screen_stable requires a terminal screen projection")
	}
	if wait.Kind == WaitQuiet || wait.Kind == WaitScreenStable {
		if wait.Quiet <= 0 {
			wait.Quiet = 150 * time.Millisecond
		}
	}

	lastEnd := uint64(0)
	lastChange := time.Now()
	sawData := false
	for {
		slice := s.ring.snapshot(from, maxBytes)
		if slice.end != lastEnd {
			lastEnd = slice.end
			lastChange = time.Now()
		}
		if len(slice.data) > 0 {
			sawData = true
		}

		result := makeResult(slice, started)
		if wait.Kind == WaitNone {
			result.State = "complete"
			result.StopReason = "data_available"
			return result, nil
		}
		if wait.Kind == WaitPrompt || wait.Kind == WaitPattern {
			if bytes.Contains(slice.data, []byte(wait.Pattern)) {
				result.State = "matched"
				result.StopReason = "pattern_matched"
				result.Matched = true
				return result, nil
			}
		}
		if (wait.Kind == WaitQuiet || wait.Kind == WaitScreenStable) && sawData && time.Since(lastChange) >= wait.Quiet {
			result.State = "stable"
			result.StopReason = map[WaitKind]string{WaitQuiet: "quiet", WaitScreenStable: "screen_stable"}[wait.Kind]
			return result, nil
		}
		if slice.truncated {
			result.State = "limit_reached"
			result.StopReason = "max_bytes"
			return result, nil
		}
		if s.hasReadError() || s.isClosed() {
			result.State = "closed"
			result.StopReason = "connection_closed"
			return result, nil
		}

		waitFor := s.ring.changed()
		var quiet <-chan time.Time
		if (wait.Kind == WaitQuiet || wait.Kind == WaitScreenStable) && sawData {
			quiet = time.After(time.Until(lastChange.Add(wait.Quiet)))
		}
		select {
		case <-ctx.Done():
			result = makeResult(s.ring.snapshot(from, maxBytes), started)
			result.State = "timeout"
			result.StopReason = "timeout"
			return result, nil
		case <-quiet:
		case <-waitFor:
		case <-s.done:
		}
	}
}

func (s *Session) isClosed() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

func (s *Session) hasReadError() bool {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.readErr != nil
}

func makeResult(slice ringSlice, started time.Time) Result {
	data, encoding := encode(slice.data)
	return Result{
		Data:              data,
		Encoding:          encoding,
		FromOffset:        slice.from,
		NextOffset:        slice.next,
		StartOffset:       slice.start,
		EndOffset:         slice.end,
		BytesRead:         len(slice.data),
		BytesLost:         slice.lost,
		BufferedRemaining: slice.end - slice.next,
		Truncated:         slice.truncated,
		ElapsedMS:         time.Since(started).Milliseconds(),
	}
}

func encode(data []byte) (string, string) {
	if utf8.Valid(data) {
		return string(data), "utf8"
	}
	return base64.StdEncoding.EncodeToString(data), "base64"
}

func ScreenHash(screen Screen) string {
	if screen == nil {
		return ""
	}
	sum := sha256.Sum256([]byte(screen.Snapshot()))
	return base64.RawURLEncoding.EncodeToString(sum[:12])
}

type byteRing struct {
	mu        sync.RWMutex
	data      []byte
	start     uint64
	end       uint64
	changedCh chan struct{}
}

type ringSlice struct {
	data      []byte
	from      uint64
	next      uint64
	start     uint64
	end       uint64
	lost      uint64
	truncated bool
}

func newByteRing(capacity int) *byteRing {
	return &byteRing{data: make([]byte, 0, capacity), changedCh: make(chan struct{})}
}

func (r *byteRing) append(data []byte) {
	r.mu.Lock()
	if len(data) >= cap(r.data) {
		if len(data) > cap(r.data) {
			data = data[len(data)-cap(r.data):]
		}
		r.start = r.end + uint64(len(data)) - uint64(cap(r.data))
		r.data = append(r.data[:0], data...)
		r.end += uint64(len(data))
	} else {
		overflow := len(r.data) + len(data) - cap(r.data)
		if overflow > 0 {
			r.data = append(r.data[:0], r.data[overflow:]...)
			r.start += uint64(overflow)
		}
		r.data = append(r.data, data...)
		r.end += uint64(len(data))
	}
	r.notifyLocked()
	r.mu.Unlock()
}

func (r *byteRing) snapshot(from uint64, max int) ringSlice {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lost := uint64(0)
	if from < r.start {
		lost = r.start - from
		from = r.start
	}
	if from > r.end {
		from = r.end
	}
	available := int(r.end - from)
	take := available
	if take > max {
		take = max
	}
	startIndex := int(from - r.start)
	data := append([]byte(nil), r.data[startIndex:startIndex+take]...)
	return ringSlice{
		data: data, from: from, next: from + uint64(take), start: r.start, end: r.end,
		lost: lost, truncated: take < available,
	}
}

func (r *byteRing) bounds() (uint64, uint64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.start, r.end
}

func (r *byteRing) endOffset() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.end
}

func (r *byteRing) changed() <-chan struct{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.changedCh
}

func (r *byteRing) signal() {
	r.mu.Lock()
	r.notifyLocked()
	r.mu.Unlock()
}

func (r *byteRing) notifyLocked() {
	close(r.changedCh)
	r.changedCh = make(chan struct{})
}
