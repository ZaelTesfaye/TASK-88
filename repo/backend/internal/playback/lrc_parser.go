package playback

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// LRCLine represents a single lyrics line with timestamp.
type LRCLine struct {
	TimeMs int64     `json:"time_ms"`
	Text   string    `json:"text"`
	Words  []LRCWord `json:"words,omitempty"`
}

// LRCWord represents a word with timing.
type LRCWord struct {
	TimeMs int64  `json:"time_ms"`
	EndMs  int64  `json:"end_ms"`
	Text   string `json:"text"`
}

// Regex patterns for LRC parsing.
var (
	// Matches a timestamp: [mm:ss.xx] or [mm:ss.xxx]
	timestampRe = regexp.MustCompile(`\[(\d{1,3}):(\d{2})(?:\.(\d{2,3}))?\]`)

	// Matches metadata tags like [ti:Title], [ar:Artist], etc.
	metadataRe = regexp.MustCompile(`^\[([a-zA-Z#]+):([^\]]*)\]$`)

	// Matches word-level timing: <mm:ss.xx>word
	wordTimingRe = regexp.MustCompile(`<(\d{1,3}):(\d{2})(?:\.(\d{2,3}))?>([^<]*)`)
)

// Known metadata tag prefixes.
var metadataTags = map[string]bool{
	"ti": true, "ar": true, "al": true, "au": true,
	"length": true, "by": true, "offset": true,
	"re": true, "ve": true, "#": true,
}

// ParseLRC parses an LRC file content.
// Handles: [mm:ss.xx] line format (centisecond precision)
// Handles: [mm:ss.xx]<mm:ss.xx>word<mm:ss.xx>word format for word-level
// Handles: multiple timestamps per line [mm:ss.xx][mm:ss.xx] text
// Handles: metadata tags [ti:], [ar:], [al:], etc.
// Returns sorted by timestamp.
func ParseLRC(content string) ([]LRCLine, error) {
	// Convert UTF-16 to UTF-8 if needed.
	content = ensureUTF8(content)

	lines := strings.Split(content, "\n")
	var result []LRCLine

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip metadata lines.
		if isMetadataLine(line) {
			continue
		}

		// Extract all timestamps from the line.
		timestamps := extractTimestamps(line)
		if len(timestamps) == 0 {
			continue
		}

		// Get the text after all timestamps.
		text := removeTimestamps(line)
		text = strings.TrimSpace(text)

		// Parse word-level timing if present.
		var words []LRCWord
		if strings.Contains(text, "<") {
			words = parseWordTimings(text)
			if len(words) > 0 {
				// Reconstruct plain text from words.
				var plainParts []string
				for _, w := range words {
					plainParts = append(plainParts, w.Text)
				}
				text = strings.Join(plainParts, "")
			}
		}

		// Create a line entry for each timestamp (handles multi-timestamp lines).
		for _, ts := range timestamps {
			lrcLine := LRCLine{
				TimeMs: ts,
				Text:   text,
			}
			if len(words) > 0 {
				lrcLine.Words = words
			}
			result = append(result, lrcLine)
		}
	}

	// Sort by timestamp.
	sort.Slice(result, func(i, j int) bool {
		return result[i].TimeMs < result[j].TimeMs
	})

	return result, nil
}

// SearchLyrics searches lyrics text and returns matching lines with timestamps.
// Case-insensitive substring match.
func SearchLyrics(lines []LRCLine, query string) []LRCLine {
	if query == "" {
		return nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []LRCLine

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line.Text), lowerQuery) {
			matches = append(matches, line)
		}
	}

	return matches
}

// FindNearestLine finds the line closest to a given timestamp.
func FindNearestLine(lines []LRCLine, timeMs int64) *LRCLine {
	if len(lines) == 0 {
		return nil
	}

	// Binary search for the closest line at or before the timestamp.
	idx := sort.Search(len(lines), func(i int) bool {
		return lines[i].TimeMs > timeMs
	})

	// idx is the first line with TimeMs > timeMs.
	// The line we want is idx-1 (the last line at or before the timestamp).
	if idx > 0 {
		return &lines[idx-1]
	}
	// If no line is at or before timeMs, return the first line.
	return &lines[0]
}

// ValidateLRCFormat checks if content is valid LRC.
func ValidateLRCFormat(content string) error {
	content = ensureUTF8(content)

	lines := strings.Split(content, "\n")
	hasTimestamp := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if isMetadataLine(line) {
			continue
		}
		if timestampRe.MatchString(line) {
			hasTimestamp = true
			break
		}
	}

	if !hasTimestamp {
		return fmt.Errorf("no valid LRC timestamps found in content")
	}

	return nil
}

// ensureUTF8 converts content to UTF-8, detecting and converting UTF-16 if needed.
func ensureUTF8(content string) string {
	data := []byte(content)

	// Check for UTF-16 BOM.
	if len(data) >= 2 {
		// UTF-16 LE BOM: FF FE
		if data[0] == 0xFF && data[1] == 0xFE {
			return decodeUTF16LE(data[2:])
		}
		// UTF-16 BE BOM: FE FF
		if data[0] == 0xFE && data[1] == 0xFF {
			return decodeUTF16BE(data[2:])
		}
	}

	// Check for UTF-8 BOM and strip it.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:])
	}

	// Heuristic: if the data contains many null bytes interleaved with ASCII,
	// it is likely UTF-16 LE without BOM.
	if len(data) >= 4 && !utf8.Valid(data) {
		nullCount := 0
		for i := 1; i < len(data) && i < 100; i += 2 {
			if data[i] == 0 {
				nullCount++
			}
		}
		if nullCount > len(data)/6 {
			return decodeUTF16LE(data)
		}
	}

	return content
}

// decodeUTF16LE decodes UTF-16 Little Endian bytes to a UTF-8 string.
func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}

	u16s := make([]uint16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		u16s[i/2] = uint16(data[i]) | uint16(data[i+1])<<8
	}

	runes := utf16.Decode(u16s)
	return string(runes)
}

// decodeUTF16BE decodes UTF-16 Big Endian bytes to a UTF-8 string.
func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}

	u16s := make([]uint16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		u16s[i/2] = uint16(data[i])<<8 | uint16(data[i+1])
	}

	runes := utf16.Decode(u16s)
	return string(runes)
}

// isMetadataLine checks if a line is a metadata tag.
func isMetadataLine(line string) bool {
	matches := metadataRe.FindStringSubmatch(line)
	if len(matches) < 2 {
		return false
	}
	tag := strings.ToLower(matches[1])
	return metadataTags[tag]
}

// extractTimestamps extracts all timestamp values (in ms) from a line.
func extractTimestamps(line string) []int64 {
	matches := timestampRe.FindAllStringSubmatch(line, -1)
	var timestamps []int64

	for _, match := range matches {
		ms, err := parseTimestamp(match[1], match[2], match[3])
		if err != nil {
			continue
		}
		timestamps = append(timestamps, ms)
	}

	return timestamps
}

// parseTimestamp converts mm, ss, cs/ms strings to milliseconds.
func parseTimestamp(minuteStr, secondStr, fractionStr string) (int64, error) {
	minutes, err := strconv.ParseInt(minuteStr, 10, 64)
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.ParseInt(secondStr, 10, 64)
	if err != nil {
		return 0, err
	}

	var fraction int64
	if fractionStr != "" {
		fraction, err = strconv.ParseInt(fractionStr, 10, 64)
		if err != nil {
			return 0, err
		}
		// If 2 digits: centiseconds; if 3 digits: milliseconds.
		if len(fractionStr) == 2 {
			fraction *= 10 // Convert centiseconds to milliseconds.
		}
	}

	return minutes*60*1000 + seconds*1000 + fraction, nil
}

// removeTimestamps removes all [mm:ss.xx] timestamps from a line.
func removeTimestamps(line string) string {
	return timestampRe.ReplaceAllString(line, "")
}

// parseWordTimings parses word-level timing from text containing <mm:ss.xx>word patterns.
func parseWordTimings(text string) []LRCWord {
	matches := wordTimingRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	var words []LRCWord
	for i, match := range matches {
		if len(match) < 5 {
			continue
		}

		startMs, err := parseTimestamp(match[1], match[2], match[3])
		if err != nil {
			continue
		}

		wordText := match[4]
		if wordText == "" {
			continue
		}

		// Calculate end time: use the start time of the next word, or estimate.
		var endMs int64
		if i+1 < len(matches) {
			nextMs, err := parseTimestamp(matches[i+1][1], matches[i+1][2], matches[i+1][3])
			if err == nil {
				endMs = nextMs
			} else {
				endMs = startMs + 500 // Default 500ms per word.
			}
		} else {
			endMs = startMs + 500
		}

		words = append(words, LRCWord{
			TimeMs: startMs,
			EndMs:  endMs,
			Text:   wordText,
		})
	}

	return words
}
