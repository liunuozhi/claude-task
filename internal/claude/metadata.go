package claude

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

// peekSize is how much of a session JSONL we scan for metadata. The fields we
// want (titles, slug, cwd) appear near the top, so reading the whole multi-MB
// transcript would be wasteful.
const peekSize = 64 * 1024

// sessionMeta holds the metadata extracted from a session's JSONL transcript.
type sessionMeta struct {
	CustomTitle string
	AITitle     string
	Slug        string
	CWD         string
}

// metaLine is one decoded JSONL record. Only the fields we care about are
// listed; everything else is ignored by encoding/json.
type metaLine struct {
	Type        string `json:"type"`
	CustomTitle string `json:"customTitle"`
	AITitle     string `json:"aiTitle"`
	Slug        string `json:"slug"`
	CWD         string `json:"cwd"`
}

// peekSessionMeta scans the first peekSize bytes of a session JSONL file for the
// custom title, ai title, first slug and first cwd. It is tolerant: a missing
// file yields a zero value with no error, and malformed lines are skipped. It
// stops early once every field has been found.
func peekSessionMeta(path string) (sessionMeta, error) {
	var m sessionMeta
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return m, err
	}
	defer f.Close()

	r := bufio.NewScanner(io.LimitReader(f, peekSize))
	// Lines can be long (tool results); allow a generous token size so a long
	// early line doesn't abort the scan.
	r.Buffer(make([]byte, 0, 64*1024), peekSize)

	for r.Scan() {
		var line metaLine
		if err := json.Unmarshal(r.Bytes(), &line); err != nil {
			continue // skip malformed line
		}
		switch line.Type {
		case "custom-title":
			if line.CustomTitle != "" {
				m.CustomTitle = line.CustomTitle
			}
		case "ai-title":
			if line.AITitle != "" {
				m.AITitle = line.AITitle
			}
		}
		if m.Slug == "" && line.Slug != "" {
			m.Slug = line.Slug
		}
		if m.CWD == "" && line.CWD != "" {
			m.CWD = line.CWD
		}
		if m.CustomTitle != "" && m.AITitle != "" && m.Slug != "" && m.CWD != "" {
			break // everything found
		}
	}
	// A scan error past the limit (e.g. a truncated final line) is non-fatal:
	// whatever we collected so far is still useful.
	return m, nil
}
