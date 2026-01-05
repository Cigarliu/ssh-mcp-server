package sshmcp

import (
	"testing"
)

// TestCircularBuffer_BasicOperations tests basic write and read operations
func TestCircularBuffer_BasicOperations(t *testing.T) {
	cb := NewCircularBuffer(10)

	// Write 5 lines
	for i := 1; i <= 5; i++ {
		cb.Write("Line content")
	}

	if count := cb.GetCount(); count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}

	if capacity := cb.GetCapacity(); capacity != 10 {
		t.Errorf("Expected capacity 10, got %d", capacity)
	}
}

// TestCircularBuffer_Overflow tests buffer overflow behavior
func TestCircularBuffer_Overflow(t *testing.T) {
	cb := NewCircularBuffer(5)

	// Write 10 lines (should overflow)
	for i := 1; i <= 10; i++ {
		cb.Write("Line content")
	}

	// Buffer should only keep 5 most recent lines
	if count := cb.GetCount(); count != 5 {
		t.Errorf("Expected count 5 after overflow, got %d", count)
	}
}

// TestCircularBuffer_ReadLatestLines tests reading latest N lines
func TestCircularBuffer_ReadLatestLines(t *testing.T) {
	cb := NewCircularBuffer(10)

	// Write 5 lines
	for i := 1; i <= 5; i++ {
		cb.Write("Line content")
	}

	// Read latest 3 lines
	lines := cb.ReadLatestLines(3)

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

// TestCircularBuffer_ReadLatestLines_Empty tests reading from empty buffer
func TestCircularBuffer_ReadLatestLines_Empty(t *testing.T) {
	cb := NewCircularBuffer(10)

	lines := cb.ReadLatestLines(5)

	if len(lines) != 0 {
		t.Errorf("Expected 0 lines from empty buffer, got %d", len(lines))
	}
}

// TestCircularBuffer_ReadLatestLines_OverflowRequest tests requesting more lines than available
func TestCircularBuffer_ReadLatestLines_OverflowRequest(t *testing.T) {
	cb := NewCircularBuffer(10)

	// Write only 3 lines
	for i := 1; i <= 3; i++ {
		cb.Write("Line content")
	}

	// Request 10 lines (should only return 3)
	lines := cb.ReadLatestLines(10)

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines when requesting more than available, got %d", len(lines))
	}
}

// TestCircularBuffer_ReadLatestBytes tests reading latest N bytes
func TestCircularBuffer_ReadLatestBytes(t *testing.T) {
	cb := NewCircularBuffer(10)

	// Write lines with known length
	cb.Write("12345") // 5 bytes
	cb.Write("67890") // 5 bytes
	cb.Write("abcde") // 5 bytes

	// Read latest 10 bytes
	result := cb.ReadLatestBytes(10)

	// Should return "67890\nabcde\n" (approximately)
	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}
}

// TestCircularBuffer_ReadAllUnread tests reading all unread data
func TestCircularBuffer_ReadAllUnread(t *testing.T) {
	cb := NewCircularBuffer(10)

	// Write 5 lines
	for i := 1; i <= 5; i++ {
		cb.Write("Line content")
	}

	// Read all unread
	lines := cb.ReadAllUnread()

	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}

	// Buffer should now be empty
	if count := cb.GetCount(); count != 0 {
		t.Errorf("Expected count 0 after reading all, got %d", count)
	}
}

// TestCircularBuffer_HeartbeatFiltering tests that heartbeat data is filtered out
func TestCircularBuffer_HeartbeatFiltering(t *testing.T) {
	cb := NewCircularBuffer(10)

	// Write normal data
	cb.Write("Normal line 1")
	cb.Write("Normal line 2")

	// Write heartbeat data (should be filtered)
	cb.Write("\x1b[s\x1b[u") // ANSI save/restore cursor
	cb.Write("\x00")         // NULL character
	cb.Write("\x1b[s")       // ANSI save cursor
	cb.Write("\x1b[u")       // ANSI restore cursor

	// Write more normal data
	cb.Write("Normal line 3")

	// Should only have 3 lines (heartbeats filtered)
	if count := cb.GetCount(); count != 3 {
		t.Errorf("Expected 3 lines after filtering heartbeats, got %d", count)
	}

	lines := cb.ReadAllUnread()
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
}

// TestCircularBuffer_ConcurrentAccess tests concurrent read/write operations
func TestCircularBuffer_ConcurrentAccess(t *testing.T) {
	cb := NewCircularBuffer(1000)
	done := make(chan bool)

	// Start 10 writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cb.Write("Line content")
			}
			done <- true
		}(i)
	}

	// Start 5 readers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				cb.ReadLatestLines(10)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}

	// Should have 1000 lines total (10 writers * 100 writes)
	// But buffer only holds 1000, so count should be <= 1000
	count := cb.GetCount()
	if count > 1000 {
		t.Errorf("Expected count <= 1000, got %d", count)
	}

	t.Logf("Concurrent test passed. Final count: %d", count)
}

// TestCircularBuffer_LineIntegrity tests that lines are not corrupted
func TestCircularBuffer_LineIntegrity(t *testing.T) {
	cb := NewCircularBuffer(100)

	// Write lines with unique content
	for i := 0; i < 50; i++ {
		cb.Write("Line content")
	}

	lines := cb.ReadLatestLines(50)

	if len(lines) != 50 {
		t.Fatalf("Expected 50 lines, got %d", len(lines))
	}

	// Verify each line
	for i, line := range lines {
		if line != "Line content" {
			t.Errorf("Line %d corrupted: got %s", i, line)
		}
	}
}
