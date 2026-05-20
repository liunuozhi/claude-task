package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// Task status values, exactly as written by Claude Code.
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
)

// Task is a single TodoWrite item, read from
// ~/.claude/tasks/{session-id}/{N}.json.
type Task struct {
	ID          string   `json:"id"`
	Subject     string   `json:"subject"`
	Description string   `json:"description"`
	ActiveForm  string   `json:"activeForm"`
	Status      string   `json:"status"` // pending | in_progress | completed
	Blocks      []string `json:"blocks"`
	BlockedBy   []string `json:"blockedBy"`

	SessionID string    `json:"-"` // derived from the parent directory name
	UpdatedAt time.Time `json:"-"` // file mtime
}

// LoadTasks reads every {N}.json task file for one session. It is tolerant: a
// missing session directory yields (nil, nil), and a single malformed file is
// skipped rather than failing the whole scan. Only an unreadable directory is a
// hard error. Tasks are returned sorted by numeric ID ("2" before "11").
func LoadTasks(base, sessionID string) ([]Task, error) {
	dir := filepath.Join(TasksDir(base), sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tasks []Task
	for _, e := range entries {
		// Task dirs also hold .lock / .highwatermark — filter strictly to *.json.
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // skip unreadable file
		}
		var t Task
		if err := json.Unmarshal(data, &t); err != nil {
			continue // skip malformed file
		}
		t.SessionID = sessionID
		if info, err := e.Info(); err == nil {
			t.UpdatedAt = info.ModTime()
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return taskIDLess(tasks[i].ID, tasks[j].ID)
	})
	return tasks, nil
}

// taskIDLess orders task IDs numerically when both parse as integers, falling
// back to lexical comparison otherwise.
func taskIDLess(a, b string) bool {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	if aerr == nil && berr == nil {
		return ai < bi
	}
	return a < b
}
