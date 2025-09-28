package submodels

import "strings"

var excludedMetadataKeys = map[string]struct{}{
	"created": {},
	"title":   {},
}

// ShouldExcludeMetadataKey reports whether the provided metadata key should be
// hidden from filter inventory and options.
func ShouldExcludeMetadataKey(key string) bool {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return false
	}

	_, excluded := excludedMetadataKeys[strings.ToLower(trimmed)]
	return excluded
}
