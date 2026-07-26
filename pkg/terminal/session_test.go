package terminal

import (
	"context"
	"encoding/base64"
	"io"
	"sync"
	"testing"
	"time"
)

type hookWriter struct {
	mu      sync.Mutex
	writes  [][]byte
	onWrite func([]byte)
}

func (w *hookWriter) Write(data []byte) (int, error) {
	copyData := append([]byte(nil), data...)
	w.mu.Lock()
	w.writes = append(w.writes, copyData)
	w.mu.Unlock()
	if w.onWrite != nil {
		w.onWrite(copyData)
	}
	return len(data), nil
}

func newPipeSession(t *testing.T, capacity int, writer io.Writer) (*Session, *io.PipeWriter) {
	t.Helper()
	reader, pipeWriter := io.Pipe()
	session := New(reader, writer, Config{BufferSize: capacity})
	t.Cleanup(func() {
		_ = pipeWriter.Close()
		session.Close()
	})
	return session, pipeWriter
}

func waitForEnd(t *testing.T, session *Session, end uint64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if session.Status().EndOffset >= end {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("terminal stream did not reach offset %d; got %+v", end, session.Status())
}

func TestInteractCapturesImmediateResponseBeforeWrite(t *testing.T) {
	writer := &hookWriter{}
	session, pipeWriter := newPipeSession(t, 64, writer)
	writer.onWrite = func([]byte) {
		_, _ = pipeWriter.Write([]byte("ready> "))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := session.Interact(ctx, InteractRequest{
		Input:    []byte("status\n"),
		Wait:     Wait{Kind: WaitUntil, Until: "ready> "},
		MaxBytes: 64,
	})
	if err != nil {
		t.Fatalf("interact: %v", err)
	}
	if !result.Matched || result.State != "matched" || result.Data != "ready> " {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSessionPreservesChunkOrderingAndOffsets(t *testing.T) {
	session, pipeWriter := newPipeSession(t, 4, io.Discard)
	if _, err := pipeWriter.Write([]byte("foo")); err != nil {
		t.Fatalf("write first chunk: %v", err)
	}
	if _, err := pipeWriter.Write([]byte("bar\n")); err != nil {
		t.Fatalf("write second chunk: %v", err)
	}
	waitForEnd(t, session, 7)

	offset := uint64(0)
	result, err := session.Interact(context.Background(), InteractRequest{
		FromOffset: &offset,
		Wait:       Wait{Kind: WaitNone},
		MaxBytes:   64,
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if result.Data != "bar\n" || result.BytesLost != 3 || result.NextOffset != 7 || result.State != "complete" || result.StopReason != "no_wait" {
		t.Fatalf("unexpected bounded replay: %+v", result)
	}
}

func TestSessionEncodesBinaryAndReportsTimeout(t *testing.T) {
	session, pipeWriter := newPipeSession(t, 32, io.Discard)
	if _, err := pipeWriter.Write([]byte{0xff, 0x00, 0x41}); err != nil {
		t.Fatalf("write binary data: %v", err)
	}
	waitForEnd(t, session, 3)

	offset := uint64(0)
	result, err := session.Interact(context.Background(), InteractRequest{
		FromOffset: &offset,
		Wait:       Wait{Kind: WaitNone},
		MaxBytes:   32,
	})
	if err != nil {
		t.Fatalf("binary replay: %v", err)
	}
	if result.Encoding != "base64" || result.Data != base64.StdEncoding.EncodeToString([]byte{0xff, 0x00, 0x41}) {
		t.Fatalf("binary response was not base64 encoded: %+v", result)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err = session.Interact(ctx, InteractRequest{Wait: Wait{Kind: WaitQuiet, Quiet: time.Millisecond}})
	if err != nil {
		t.Fatalf("timeout read: %v", err)
	}
	if result.State != "timeout" || result.StopReason != "timeout" {
		t.Fatalf("expected timeout state, got %+v", result)
	}
}
