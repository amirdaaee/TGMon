package filesystem

import (
	"strings"
	"unicode"
)

// sanitizeFilename makes a filename safe for use in filesystems by replacing
// unsafe characters with underscores. This handles slashes, colons, and other
// characters that are problematic in filenames.
func sanitizeFilename(filename string) string {
	var builder strings.Builder
	builder.Grow(len(filename) * 2) // Pre-allocate space for potential encoding

	for _, r := range filename {
		// Replace unsafe characters with underscore
		// Unsafe characters: / \ : * ? " < > | and control characters
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|' ||
			unicode.IsControl(r) {
			builder.WriteRune('_')
		} else {
			builder.WriteRune(r)
		}
	}

	result := builder.String()
	// Remove leading/trailing spaces and dots (problematic on Windows)
	result = strings.Trim(result, " .")
	// If the result is empty after trimming, use a default name
	if result == "" {
		result = "file"
	}
	return result
}
