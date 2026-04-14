# Playback Format Compatibility

## Supported Audio Formats

**Implementation**: `backend/internal/playback/playback_service.go:13`

| Format | Extension | MIME Types Accepted | Notes |
|---|---|---|---|
| MP3 | `.mp3` | `audio/mpeg`, `audio/mp3` | Most common compressed format |
| WAV | `.wav` | `audio/wav`, `audio/x-wav` | Uncompressed PCM audio |
| FLAC | `.flac` | `audio/flac`, `audio/x-flac` | Lossless compressed |
| M4A | `.m4a` | `audio/mp4`, `audio/m4a`, `audio/x-m4a`, `audio/aac` | AAC container |

Source: `backend/internal/playback/playback_service.go:219-233` (MIME validation map)

### Format Validation

Extension-based check:
```go
func IsSupportedAudioFormat(ext string) bool
```

MIME-type-based check:
```go
func isValidAudioMimeType(mimeType string) bool
```

Both checks are case-insensitive. Extension check strips the leading dot.

### Discovery Endpoint

`GET /api/v1/media/formats/supported`

```json
{
  "audio": ["mp3", "wav", "flac", "m4a"],
  "lyrics": ["lrc"]
}
```

Source: `backend/internal/handlers/playback_handler.go:313-316`

## LRC Lyrics Format

**Implementation**: `backend/internal/playback/lrc_parser.go`

### Supported LRC Features

| Feature | Syntax | Example | Supported |
|---|---|---|---|
| Line-level timestamps | `[mm:ss.xx]` | `[01:23.45] Hello world` | Yes |
| Millisecond precision | `[mm:ss.xxx]` | `[01:23.456] Hello world` | Yes |
| Multi-timestamp lines | `[mm:ss.xx][mm:ss.xx]` | `[01:23.45][02:34.56] Chorus line` | Yes |
| Word-level timing | `<mm:ss.xx>word` | `[01:23.45]<01:23.45>Hello <01:24.00>world` | Yes |
| Metadata tags | `[ti:Title]`, `[ar:Artist]` | `[ti:Song Name]` | Parsed and skipped |
| Empty lines | (blank) | | Skipped |

### Recognized Metadata Tags

Tags that are identified and excluded from lyrics output:

| Tag | Description |
|---|---|
| `ti` | Title |
| `ar` | Artist |
| `al` | Album |
| `au` | Author |
| `length` | Duration |
| `by` | LRC creator |
| `offset` | Timestamp offset |
| `re` | Player |
| `ve` | Version |
| `#` | Comment |

Source: `backend/internal/playback/lrc_parser.go:40-44`

### Timestamp Parsing

Regex: `\[(\d{1,3}):(\d{2})(?:\.(\d{2,3}))?\]`

Supports:
- 1 to 3 digit minutes (handles files > 99 minutes)
- 2-digit seconds
- Optional fractional part: 2 digits (centiseconds) or 3 digits (milliseconds)
- Centiseconds are converted to milliseconds by multiplying by 10

Source: `backend/internal/playback/lrc_parser.go:278-301` (parseTimestamp)

### Word-Level Timing

Regex: `<(\d{1,3}):(\d{2})(?:\.(\d{2,3}))?>([^<]*)`

Each word has:
- `time_ms`: Start time in milliseconds
- `end_ms`: End time (start of next word, or +500ms if last word)
- `text`: The word text

Source: `backend/internal/playback/lrc_parser.go:309-352`

### Output Schema

```json
{
  "status": "success",
  "line_count": 42,
  "lines": [
    {
      "time_ms": 15300,
      "text": "Hello world",
      "words": [
        {"time_ms": 15300, "end_ms": 15800, "text": "Hello "},
        {"time_ms": 15800, "end_ms": 16300, "text": "world"}
      ]
    }
  ]
}
```

## Encoding Fallbacks

**Implementation**: `backend/internal/playback/lrc_parser.go:183-248`

### Detection Order

1. **UTF-16 LE BOM** (bytes `FF FE`): Decode as UTF-16 Little Endian
2. **UTF-16 BE BOM** (bytes `FE FF`): Decode as UTF-16 Big Endian
3. **UTF-8 BOM** (bytes `EF BB BF`): Strip BOM, treat as UTF-8
4. **Heuristic detection**: If content is not valid UTF-8 and contains interleaved null bytes in the first 100 bytes, treat as UTF-16 LE without BOM
5. **Default**: Treat as UTF-8

### UTF-16 Conversion

Both LE and BE decoders:
- Handle surrogate pairs via `unicode/utf16.Decode()`
- Trim odd trailing bytes
- Return valid UTF-8 string

Source: `backend/internal/playback/lrc_parser.go:221-248`

## Parsing Fallbacks

### Parse Error Handling

When LRC parsing encounters issues, the API returns a success response with parse status:

```json
{
  "status": "parse_error",
  "message": "no valid LRC timestamps found in content",
  "lines": []
}
```

Source: `backend/internal/handlers/playback_handler.go:256-261`

### LRC Content Sources (Priority)

1. **Request body**: Raw LRC text posted directly
2. **Asset file path**: Read from `media_asset.lyrics_lrc_path` on disk
3. **Neither available**: Returns 400 error

Source: `backend/internal/handlers/playback_handler.go:238-252`

### Format Validation

```go
func ValidateLRCFormat(content string) error
```

Returns error if no valid `[mm:ss.xx]` timestamps are found in the content. Metadata-only files are considered invalid.

Source: `backend/internal/playback/lrc_parser.go:155-180`

## UX Behavior

### Audio Streaming

`GET /api/v1/media/:id/stream`

1. Looks up the media asset by ID
2. Checks that `audio_path` is non-empty
3. Verifies the file exists on disk (`os.Stat`)
4. Serves the file using `c.File(asset.AudioPath)` (Gin handles Content-Type and range requests)

Source: `backend/internal/handlers/playback_handler.go:170-194`

### Cover Art

`GET /api/v1/media/:id/cover`

Same flow as audio streaming but uses `cover_art_path`.

Source: `backend/internal/handlers/playback_handler.go:196-220`

### Lyrics Search

`GET /api/v1/media/:id/lyrics/search?q=hello`

1. Reads the LRC file from disk
2. Parses it into structured lines
3. Performs case-insensitive substring search across all line texts
4. Returns matching lines with their timestamps

```json
{
  "matches": [
    {"time_ms": 15300, "text": "Hello world", "words": [...]},
    {"time_ms": 45600, "text": "Say hello again", "words": [...]}
  ],
  "total": 2
}
```

Source: `backend/internal/playback/lrc_parser.go:117-132`

### Nearest Line Lookup

The `FindNearestLine(lines, timeMs)` function uses binary search (`sort.Search`) to find the lyrics line active at a given playback position. Returns the last line with `time_ms <= timeMs`.

Source: `backend/internal/playback/lrc_parser.go:136-152`

## Media Asset Model

| Field | Type | Description |
|---|---|---|
| `title` | VARCHAR(500) | Display title |
| `audio_path` | VARCHAR(1000) | Path to audio file on disk |
| `cover_art_path` | VARCHAR(1000) | Path to cover art image |
| `lyrics_lrc_path` | VARCHAR(1000) | Path to LRC lyrics file |
| `theme_json` | JSON | Custom theme/styling data |
| `duration` | INT | Duration in seconds |
| `mime_type` | VARCHAR(100) | Audio MIME type |
| `file_size_bytes` | BIGINT | File size |
| `uploaded_by` | BIGINT | Uploader user ID |
| `status` | VARCHAR(50) | active, deleted |

Source: `backend/internal/models/media.go`
