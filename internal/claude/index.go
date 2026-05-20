package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// indexEntry is the optional per-session enrichment from sessions-index.json.
// This file is absent on many machines, so it is treated as purely optional:
// any failure to read or parse it simply yields no enrichment.
type indexEntry struct {
	Description string `json:"description"`
	GitBranch   string `json:"gitBranch"`
}

// loadIndexEnrichment reads ~/.claude/sessions-index.json if present, returning
// a map keyed by session ID. A missing or malformed file yields an empty map.
func loadIndexEnrichment(base string) map[string]indexEntry {
	out := make(map[string]indexEntry)
	data, err := os.ReadFile(filepath.Join(base, "sessions-index.json"))
	if err != nil {
		return out
	}
	// Tolerate either {"sessions": {id: {...}}} or a flat {id: {...}} shape.
	var wrapped struct {
		Sessions map[string]indexEntry `json:"sessions"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Sessions) > 0 {
		return wrapped.Sessions
	}
	var flat map[string]indexEntry
	if err := json.Unmarshal(data, &flat); err == nil {
		return flat
	}
	return out
}
