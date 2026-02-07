# Checkpoint Package - Agent Notes

## Scope
- Parse Apache Flink `_metadata` checkpoint files.
- Provide both full parsing and summary parsing.

## Key Files
- `metadata.go`: entry points (`Parse`, `ParseSummary`, `ParseFile`, `ParseFileSummary`).
- `reader.go`: binary reader helpers (big-endian, modified UTF-8).
- `stream_handle.go`, `operator_state.go`, `keyed_state.go`, `channel_state.go`: state handle parsing.
- `checkpoint_properties.go`: lightweight parsing of trailing Java-serialization bytes.
- `summary_scan.go`: scan inline strings and state file paths.

## Behavior Notes
- Summary parsing avoids deep keyed-state and channel-state decoding when `ParseOptions.ParseFull` is false.
- Channel state is only present in metadata version >= 3.
- Checkpoint properties appear in version >= 4.

## Tests
- `go test ./checkpoint` from `backend/`.
