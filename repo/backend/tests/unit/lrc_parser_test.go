package unit

import (
	"testing"

	"backend/internal/playback"
)

func TestParseLRCBasic(t *testing.T) {
	content := `[00:12.00]First line
[00:17.20]Second line
[00:30.50]Third line`

	lines, err := playback.ParseLRC(content)
	if err != nil {
		t.Fatalf("ParseLRC returned error: %v", err)
	}

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// [00:12.00] = 0*60000 + 12*1000 + 0 = 12000ms
	if lines[0].TimeMs != 12000 {
		t.Errorf("line 0: expected 12000ms, got %d", lines[0].TimeMs)
	}
	if lines[0].Text != "First line" {
		t.Errorf("line 0: expected 'First line', got %q", lines[0].Text)
	}

	// [00:17.20] = 17*1000 + 200 = 17200ms
	if lines[1].TimeMs != 17200 {
		t.Errorf("line 1: expected 17200ms, got %d", lines[1].TimeMs)
	}
	if lines[1].Text != "Second line" {
		t.Errorf("line 1: expected 'Second line', got %q", lines[1].Text)
	}

	// [00:30.50] = 30*1000 + 500 = 30500ms
	if lines[2].TimeMs != 30500 {
		t.Errorf("line 2: expected 30500ms, got %d", lines[2].TimeMs)
	}

	// Verify sorted order.
	for i := 1; i < len(lines); i++ {
		if lines[i].TimeMs < lines[i-1].TimeMs {
			t.Errorf("lines not sorted: line %d (%dms) < line %d (%dms)",
				i, lines[i].TimeMs, i-1, lines[i-1].TimeMs)
		}
	}
}

func TestParseLRCWordLevel(t *testing.T) {
	content := `[00:12.00]<00:12.00>Hello <00:12.50>World <00:13.00>Today`

	lines, err := playback.ParseLRC(content)
	if err != nil {
		t.Fatalf("ParseLRC returned error: %v", err)
	}

	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	line := lines[0]
	if len(line.Words) == 0 {
		t.Fatal("expected word-level timing data, got none")
	}

	if len(line.Words) != 3 {
		t.Fatalf("expected 3 words, got %d", len(line.Words))
	}

	// First word: "Hello " at 12000ms
	if line.Words[0].Text != "Hello " {
		t.Errorf("word 0: expected 'Hello ', got %q", line.Words[0].Text)
	}
	if line.Words[0].TimeMs != 12000 {
		t.Errorf("word 0: expected 12000ms, got %d", line.Words[0].TimeMs)
	}

	// Second word: "World " at 12500ms
	if line.Words[1].Text != "World " {
		t.Errorf("word 1: expected 'World ', got %q", line.Words[1].Text)
	}
	if line.Words[1].TimeMs != 12500 {
		t.Errorf("word 1: expected 12500ms, got %d", line.Words[1].TimeMs)
	}

	// Third word: "Today" at 13000ms
	if line.Words[2].Text != "Today" {
		t.Errorf("word 2: expected 'Today', got %q", line.Words[2].Text)
	}
	if line.Words[2].TimeMs != 13000 {
		t.Errorf("word 2: expected 13000ms, got %d", line.Words[2].TimeMs)
	}

	// Verify end times: first word's end = second word's start
	if line.Words[0].EndMs != 12500 {
		t.Errorf("word 0 end: expected 12500ms, got %d", line.Words[0].EndMs)
	}
}

func TestParseLRCMultipleTimestamps(t *testing.T) {
	// Multiple timestamps on the same line: shared text.
	content := `[01:00.00][02:00.00]Repeated chorus line`

	lines, err := playback.ParseLRC(content)
	if err != nil {
		t.Fatalf("ParseLRC returned error: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (one per timestamp), got %d", len(lines))
	}

	// Both lines should have the same text.
	for _, line := range lines {
		if line.Text != "Repeated chorus line" {
			t.Errorf("expected 'Repeated chorus line', got %q", line.Text)
		}
	}

	// First timestamp: 1*60000 = 60000ms
	if lines[0].TimeMs != 60000 {
		t.Errorf("line 0: expected 60000ms, got %d", lines[0].TimeMs)
	}
	// Second timestamp: 2*60000 = 120000ms
	if lines[1].TimeMs != 120000 {
		t.Errorf("line 1: expected 120000ms, got %d", lines[1].TimeMs)
	}
}

func TestParseLRCMetadata(t *testing.T) {
	content := `[ti:Song Title]
[ar:Artist Name]
[al:Album Name]
[by:Creator]
[00:05.00]First lyric line
[00:10.00]Second lyric line`

	lines, err := playback.ParseLRC(content)
	if err != nil {
		t.Fatalf("ParseLRC returned error: %v", err)
	}

	// Metadata lines should be skipped, only lyric lines returned.
	if len(lines) != 2 {
		t.Fatalf("expected 2 lyric lines (metadata skipped), got %d", len(lines))
	}

	if lines[0].Text != "First lyric line" {
		t.Errorf("line 0: expected 'First lyric line', got %q", lines[0].Text)
	}
	if lines[1].Text != "Second lyric line" {
		t.Errorf("line 1: expected 'Second lyric line', got %q", lines[1].Text)
	}
}

func TestSearchLyrics(t *testing.T) {
	lines := []playback.LRCLine{
		{TimeMs: 5000, Text: "Hello world"},
		{TimeMs: 10000, Text: "Goodbye World"},
		{TimeMs: 15000, Text: "Testing lyrics search"},
		{TimeMs: 20000, Text: "Another WORLD line"},
	}

	// Case-insensitive search for "world"
	results := playback.SearchLyrics(lines, "world")
	if len(results) != 3 {
		t.Fatalf("expected 3 matches for 'world', got %d", len(results))
	}

	// Search for "testing"
	results2 := playback.SearchLyrics(lines, "TESTING")
	if len(results2) != 1 {
		t.Fatalf("expected 1 match for 'TESTING', got %d", len(results2))
	}
	if results2[0].TimeMs != 15000 {
		t.Errorf("expected match at 15000ms, got %d", results2[0].TimeMs)
	}

	// Search for empty string
	resultsEmpty := playback.SearchLyrics(lines, "")
	if resultsEmpty != nil {
		t.Errorf("expected nil for empty query, got %d results", len(resultsEmpty))
	}

	// Search for non-existent text
	resultsNone := playback.SearchLyrics(lines, "nonexistent")
	if len(resultsNone) != 0 {
		t.Errorf("expected 0 results for nonexistent query, got %d", len(resultsNone))
	}
}

func TestFindNearestLine(t *testing.T) {
	lines := []playback.LRCLine{
		{TimeMs: 5000, Text: "Line 1"},
		{TimeMs: 10000, Text: "Line 2"},
		{TimeMs: 20000, Text: "Line 3"},
		{TimeMs: 30000, Text: "Line 4"},
	}

	tests := []struct {
		name     string
		timeMs   int64
		expected string
	}{
		{"exact match", 10000, "Line 2"},
		{"between lines", 15000, "Line 2"},
		{"before first line", 1000, "Line 1"},
		{"after last line", 50000, "Line 4"},
		{"exact first", 5000, "Line 1"},
		{"exact last", 30000, "Line 4"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := playback.FindNearestLine(lines, tc.timeMs)
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Text != tc.expected {
				t.Errorf("at %dms: expected %q, got %q", tc.timeMs, tc.expected, result.Text)
			}
		})
	}

	// Test with empty slice.
	nilResult := playback.FindNearestLine([]playback.LRCLine{}, 5000)
	if nilResult != nil {
		t.Error("expected nil for empty lines slice")
	}
}

func TestInvalidLRC(t *testing.T) {
	// Content with no timestamps at all.
	invalidContent := `This is just plain text
without any LRC timestamps
at all`

	err := playback.ValidateLRCFormat(invalidContent)
	if err == nil {
		t.Fatal("expected error for content without timestamps, got nil")
	}

	// ParseLRC should handle it gracefully (return empty, no crash).
	lines, err := playback.ParseLRC(invalidContent)
	if err != nil {
		t.Fatalf("ParseLRC should not return error for invalid content, got: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines from invalid LRC, got %d", len(lines))
	}
}

func TestEmptyLRC(t *testing.T) {
	// Empty content.
	lines, err := playback.ParseLRC("")
	if err != nil {
		t.Fatalf("ParseLRC should not error on empty content, got: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines from empty LRC, got %d", len(lines))
	}

	// Whitespace-only content.
	lines2, err := playback.ParseLRC("   \n\n   \n")
	if err != nil {
		t.Fatalf("ParseLRC should not error on whitespace content, got: %v", err)
	}
	if len(lines2) != 0 {
		t.Errorf("expected 0 lines from whitespace LRC, got %d", len(lines2))
	}

	// Validate format on empty content.
	err = playback.ValidateLRCFormat("")
	if err == nil {
		t.Error("ValidateLRCFormat should return error for empty content")
	}
}

// TestParseLRCWithUTF16LE exercises the UTF-16 LE BOM detection path.
func TestParseLRCWithUTF16LE(t *testing.T) {
	// UTF-16 LE BOM = 0xFF 0xFE, followed by UTF-16LE encoded LRC content.
	lrc := "[00:01.00]Hello"
	var utf16le []byte
	utf16le = append(utf16le, 0xFF, 0xFE) // BOM
	for _, r := range lrc {
		utf16le = append(utf16le, byte(r), 0x00) // LE: low byte first
	}

	lines, err := playback.ParseLRC(string(utf16le))
	if err != nil {
		t.Fatalf("ParseLRC with UTF-16LE BOM failed: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "Hello" {
		t.Errorf("expected text 'Hello', got %q", lines[0].Text)
	}
}

// TestParseLRCWithUTF16BE exercises the UTF-16 BE BOM detection path.
func TestParseLRCWithUTF16BE(t *testing.T) {
	// UTF-16 BE BOM = 0xFE 0xFF, followed by UTF-16BE encoded LRC content.
	lrc := "[00:02.00]World"
	var utf16be []byte
	utf16be = append(utf16be, 0xFE, 0xFF) // BOM
	for _, r := range lrc {
		utf16be = append(utf16be, 0x00, byte(r)) // BE: high byte first
	}

	lines, err := playback.ParseLRC(string(utf16be))
	if err != nil {
		t.Fatalf("ParseLRC with UTF-16BE BOM failed: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Text != "World" {
		t.Errorf("expected text 'World', got %q", lines[0].Text)
	}
}
