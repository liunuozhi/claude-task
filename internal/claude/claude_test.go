package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveClaudeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	t.Run("flag wins over env", func(t *testing.T) {
		t.Setenv("CLAUDE_DIR", "/from/env")
		if got := ResolveClaudeDir("/from/flag"); got != "/from/flag" {
			t.Errorf("got %q, want /from/flag", got)
		}
	})

	t.Run("env wins over default", func(t *testing.T) {
		t.Setenv("CLAUDE_DIR", "/from/env")
		if got := ResolveClaudeDir(""); got != "/from/env" {
			t.Errorf("got %q, want /from/env", got)
		}
	})

	t.Run("default is ~/.claude", func(t *testing.T) {
		t.Setenv("CLAUDE_DIR", "")
		want := filepath.Join(home, ".claude")
		if got := ResolveClaudeDir(""); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("tilde expansion in flag", func(t *testing.T) {
		want := filepath.Join(home, "custom")
		if got := ResolveClaudeDir("~/custom"); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("bare tilde in env", func(t *testing.T) {
		t.Setenv("CLAUDE_DIR", "~")
		if got := ResolveClaudeDir(""); got != home {
			t.Errorf("got %q, want %q", got, home)
		}
	})
}

// writeTask writes a task JSON file and back-dates its mtime.
func writeTask(t *testing.T, dir, name, body string, mtime time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	if !mtime.IsZero() {
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}
}

func taskJSON(id, status string) string {
	return `{"id":"` + id + `","subject":"s` + id + `","description":"d","activeForm":"a","status":"` + status + `","blocks":[],"blockedBy":[]}`
}

func TestLoadTasks(t *testing.T) {
	base := t.TempDir()
	sid := "sess1"
	dir := filepath.Join(TasksDir(base), sid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Numeric IDs out of lexical order, plus noise files and a malformed task.
	writeTask(t, dir, "2.json", taskJSON("2", StatusInProgress), time.Time{})
	writeTask(t, dir, "11.json", taskJSON("11", StatusPending), time.Time{})
	writeTask(t, dir, "1.json", taskJSON("1", StatusCompleted), time.Time{})
	writeTask(t, dir, ".lock", "", time.Time{})
	writeTask(t, dir, ".highwatermark", "2", time.Time{})
	writeTask(t, dir, "bad.json", "{not json", time.Time{})

	tasks, err := LoadTasks(base, sid)
	if err != nil {
		t.Fatalf("LoadTasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks, want 3 (noise + malformed skipped)", len(tasks))
	}
	wantOrder := []string{"1", "2", "11"} // numeric, not lexical
	for i, w := range wantOrder {
		if tasks[i].ID != w {
			t.Errorf("position %d: got id %q, want %q", i, tasks[i].ID, w)
		}
	}
	if tasks[0].SessionID != sid {
		t.Errorf("SessionID not derived: got %q", tasks[0].SessionID)
	}
}

func TestLoadTasksMissingDir(t *testing.T) {
	tasks, err := LoadTasks(t.TempDir(), "does-not-exist")
	if err != nil {
		t.Fatalf("missing dir should be (nil, nil), got err %v", err)
	}
	if tasks != nil {
		t.Errorf("got %v, want nil", tasks)
	}
}

func TestPeekSessionMeta(t *testing.T) {
	dir := t.TempDir()

	t.Run("extracts fields, ignores malformed lines", func(t *testing.T) {
		path := filepath.Join(dir, "a.jsonl")
		lines := strings.Join([]string{
			`{not valid json`,
			`{"type":"summary","cwd":"/Users/liu/proj"}`,
			`{"type":"ai-title","aiTitle":"Auto Title"}`,
			`{"slug":"witty-slug"}`,
			`{"type":"custom-title","customTitle":"My Title"}`,
		}, "\n")
		if err := os.WriteFile(path, []byte(lines), 0o644); err != nil {
			t.Fatal(err)
		}
		m, err := peekSessionMeta(path)
		if err != nil {
			t.Fatal(err)
		}
		if m.CustomTitle != "My Title" {
			t.Errorf("CustomTitle = %q", m.CustomTitle)
		}
		if m.AITitle != "Auto Title" {
			t.Errorf("AITitle = %q", m.AITitle)
		}
		if m.Slug != "witty-slug" {
			t.Errorf("Slug = %q", m.Slug)
		}
		if m.CWD != "/Users/liu/proj" {
			t.Errorf("CWD = %q", m.CWD)
		}
	})

	t.Run("missing file is zero value, no error", func(t *testing.T) {
		m, err := peekSessionMeta(filepath.Join(dir, "nope.jsonl"))
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if (m != sessionMeta{}) {
			t.Errorf("got %+v, want zero value", m)
		}
	})

	t.Run("respects 64KB cap", func(t *testing.T) {
		path := filepath.Join(dir, "big.jsonl")
		var b strings.Builder
		// Pad past the cap with valid-but-irrelevant lines, then place the
		// title far beyond peekSize; it must NOT be found.
		filler := `{"type":"text","cwd":"/early"}` + "\n"
		for b.Len() < peekSize+8192 {
			b.WriteString(filler)
		}
		b.WriteString(`{"type":"custom-title","customTitle":"TooLate"}` + "\n")
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		m, err := peekSessionMeta(path)
		if err != nil {
			t.Fatal(err)
		}
		if m.CustomTitle == "TooLate" {
			t.Error("read past the 64KB cap")
		}
		if m.CWD != "/early" {
			t.Errorf("CWD = %q, want /early", m.CWD)
		}
	})
}

func TestDisplayNamePrecedence(t *testing.T) {
	cases := []struct {
		name string
		s    Session
		want string
	}{
		{"custom wins", Session{ID: "id", CustomTitle: "C", AITitle: "A", Slug: "s"}, "C"},
		{"ai over slug", Session{ID: "id", AITitle: "A", Slug: "s"}, "A"},
		{"slug over id", Session{ID: "id", Slug: "s"}, "s"},
		{"id fallback", Session{ID: "id"}, "id"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.s.DisplayName(); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestScanSessions(t *testing.T) {
	base := t.TempDir()

	mkSession := func(id string, mtimes map[string]time.Time, statuses map[string]string) {
		dir := filepath.Join(TasksDir(base), id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for tid, st := range statuses {
			writeTask(t, dir, tid+".json", taskJSON(tid, st), mtimes[tid])
		}
	}

	old := time.Now().Add(-2 * time.Hour)
	recent := time.Now().Add(-1 * time.Minute)

	// older session
	mkSession("older",
		map[string]time.Time{"1": old},
		map[string]string{"1": StatusCompleted})
	// newer session with mixed statuses
	mkSession("newer",
		map[string]time.Time{"1": old, "2": recent, "3": old},
		map[string]string{"1": StatusPending, "2": StatusInProgress, "3": StatusCompleted})
	// empty stub session: only noise, no JSON → must be skipped
	stub := filepath.Join(TasksDir(base), "stub")
	if err := os.MkdirAll(stub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTask(t, stub, ".lock", "", time.Time{})

	// transcript metadata for "newer"
	projDir := filepath.Join(ProjectsDir(base), "-Users-liu-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "newer.jsonl"),
		[]byte(`{"type":"ai-title","aiTitle":"Newer Session","cwd":"/Users/liu/proj"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	sessions, err := ScanSessions(base)
	if err != nil {
		t.Fatalf("ScanSessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2 (stub skipped)", len(sessions))
	}
	// newest mtime first
	if sessions[0].ID != "newer" {
		t.Errorf("sort: got %q first, want newer", sessions[0].ID)
	}
	n := sessions[0]
	if n.TaskCount() != 3 || n.Pending != 1 || n.InProgress != 1 || n.Completed != 1 {
		t.Errorf("counts: %+v", n)
	}
	if n.DisplayName() != "Newer Session" {
		t.Errorf("DisplayName = %q", n.DisplayName())
	}
	if n.Project != "/Users/liu/proj" {
		t.Errorf("Project = %q", n.Project)
	}
}

func TestScanSessionsMissingDir(t *testing.T) {
	sessions, err := ScanSessions(t.TempDir())
	if err != nil {
		t.Fatalf("missing tasks dir should be (nil,nil), got %v", err)
	}
	if sessions != nil {
		t.Errorf("got %v, want nil", sessions)
	}
}

func TestFilters(t *testing.T) {
	sessions := []Session{
		{ID: "a", Project: "/p1", InProgress: 1},
		{ID: "b", Project: "/p2", InProgress: 0},
		{ID: "c", Project: "/p1", InProgress: 0},
	}

	t.Run("ProjectList distinct sorted", func(t *testing.T) {
		got := ProjectList(sessions)
		if len(got) != 2 || got[0] != "/p1" || got[1] != "/p2" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("project filter", func(t *testing.T) {
		got := ApplyFilters(sessions, "/p1", false)
		if len(got) != 2 {
			t.Errorf("got %d, want 2", len(got))
		}
	})

	t.Run("active only", func(t *testing.T) {
		got := ApplyFilters(sessions, "", true)
		if len(got) != 1 || got[0].ID != "a" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("empty project matches all", func(t *testing.T) {
		if got := ApplyFilters(sessions, "", false); len(got) != 3 {
			t.Errorf("got %d, want 3", len(got))
		}
	})
}
