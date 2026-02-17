package checkpoint

import "bytes"

// scanInlineStrings extracts printable ASCII runs from raw bytes.
func scanInlineStrings(data []byte) []string {
	strings := make([]string, 0)
	current := make([]byte, 0, 128)
	for _, b := range data {
		if b >= 32 && b <= 126 {
			current = append(current, b)
			continue
		}
		if len(current) >= 6 {
			strings = append(strings, string(current))
		}
		current = current[:0]
	}
	if len(current) >= 6 {
		strings = append(strings, string(current))
	}

	return uniqueStrings(strings)
}

// extractStateFilePaths filters scanned strings by known state path prefixes.
func extractStateFilePaths(data []byte) []string {
	strings := scanInlineStrings(data)
	paths := make([]string, 0)
	for _, s := range strings {
		if hasStatePathPrefix(s) {
			paths = append(paths, s)
		}
	}
	return uniqueStrings(paths)
}

// hasStatePathPrefix reports whether a string looks like a state file path.
func hasStatePathPrefix(value string) bool {
	if len(value) < 5 {
		return false
	}
	return bytes.HasPrefix([]byte(value), []byte("s3://")) ||
		bytes.HasPrefix([]byte(value), []byte("hdfs://")) ||
		bytes.HasPrefix([]byte(value), []byte("file:/")) ||
		bytes.HasPrefix([]byte(value), []byte("gs://"))
}

// uniqueStrings preserves first occurrence order for unique values.
func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

// copyBytes returns a shallow copy of the input.
func copyBytes(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	cpy := make([]byte, len(data))
	copy(cpy, data)
	return cpy
}
