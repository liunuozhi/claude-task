package claude

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session is one Claude Code session that has tasks, enriched with metadata from
// its JSONL transcript.
type Session struct {
	ID          string
	CustomTitle string
	AITitle     string
	Slug        string
	Project     string // real cwd from the transcript, falls back to encoded dir
	Description string // optional, from sessions-index.json
	GitBranch   string // optional, from sessions-index.json

	ModifiedAt time.Time // newest task mtime — the sort key

	Pending    int
	InProgress int
	Completed  int
}

// DisplayName resolves the human-facing name with the precedence
// customTitle > aiTitle > slug > sessionID.
func (s Session) DisplayName() string {
	switch {
	case s.CustomTitle != "":
		return s.CustomTitle
	case s.AITitle != "":
		return s.AITitle
	case s.Slug != "":
		return s.Slug
	default:
		return s.ID
	}
}

// TaskCount is the total number of tasks across all statuses.
func (s Session) TaskCount() int { return s.Pending + s.InProgress + s.Completed }

// ScanSessions discovers every session that has at least one task, loads its
// tasks for counts and the newest mtime, and enriches it with transcript
// metadata. Sessions are returned sorted by newest task mtime, descending. It is
// tolerant: per-session read errors are skipped; only an unreadable tasks
// directory is a hard error. A missing tasks directory yields (nil, nil).
func ScanSessions(base string) ([]Session, error) {
	sessions, _, _, err := ScanWithLive(base)
	return sessions, err
}

// ScanWithLive is ScanSessions plus, from the same single pass over task files:
// the flat list of all in-progress tasks across every session (the global "Live
// feed", newest first), and every session's full task slice keyed by ID. The UI
// keeps tasksByID so selecting a session needs no further disk reads.
func ScanWithLive(base string) (sessions []Session, live []Task, tasksByID map[string][]Task, err error) {
	entries, err := os.ReadDir(TasksDir(base))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, err
	}

	jsonlIndex := indexSessionTranscripts(base)
	idx := loadIndexEnrichment(base)
	tasksByID = make(map[string][]Task)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		tasks, err := LoadTasks(base, id)
		if err != nil || len(tasks) == 0 {
			continue // skip unreadable or task-less session directories
		}
		tasksByID[id] = tasks

		s := Session{ID: id}
		for _, t := range tasks {
			switch t.Status {
			case StatusPending:
				s.Pending++
			case StatusInProgress:
				s.InProgress++
				live = append(live, t)
			case StatusCompleted:
				s.Completed++
			}
			if t.UpdatedAt.After(s.ModifiedAt) {
				s.ModifiedAt = t.UpdatedAt
			}
		}

		if path, ok := jsonlIndex[id]; ok {
			if meta, err := peekSessionMeta(path); err == nil {
				s.CustomTitle = meta.CustomTitle
				s.AITitle = meta.AITitle
				s.Slug = meta.Slug
				s.Project = meta.CWD
			}
		}
		if s.Project == "" {
			s.Project = projectFromTranscriptPath(jsonlIndex[id])
		}
		if enr, ok := idx[id]; ok {
			if s.Description == "" {
				s.Description = enr.Description
			}
			if s.GitBranch == "" {
				s.GitBranch = enr.GitBranch
			}
		}

		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModifiedAt.After(sessions[j].ModifiedAt)
	})
	// Most-recently-updated in-progress tasks first.
	sort.Slice(live, func(i, j int) bool {
		return live[i].UpdatedAt.After(live[j].UpdatedAt)
	})
	return sessions, live, tasksByID, nil
}

// indexSessionTranscripts walks projects/{encoded}/{session-id}.jsonl once and
// maps each session ID to its transcript path. Tolerant of an absent dir.
func indexSessionTranscripts(base string) map[string]string {
	index := make(map[string]string)
	projects, err := os.ReadDir(ProjectsDir(base))
	if err != nil {
		return index
	}
	for _, p := range projects {
		if !p.IsDir() {
			continue
		}
		projDir := filepath.Join(ProjectsDir(base), p.Name())
		files, err := os.ReadDir(projDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".jsonl" {
				continue
			}
			id := strings.TrimSuffix(f.Name(), ".jsonl")
			// Keep the first match; duplicates across projects are unexpected.
			if _, ok := index[id]; !ok {
				index[id] = filepath.Join(projDir, f.Name())
			}
		}
	}
	return index
}

// projectFromTranscriptPath derives a readable project name from a transcript
// path when cwd is unavailable, by un-encoding the parent directory name.
func projectFromTranscriptPath(path string) string {
	if path == "" {
		return ""
	}
	dir := filepath.Base(filepath.Dir(path))
	// Encoded dirs look like "-Users-liu-Developer-projects-lecorb2".
	if strings.HasPrefix(dir, "-") {
		return "/" + strings.ReplaceAll(strings.TrimPrefix(dir, "-"), "-", "/")
	}
	return dir
}
