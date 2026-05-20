package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liunuozhi/claude-task/internal/claude"
)

// makeFixture builds a temp ~/.claude with two sessions worth of tasks and
// transcripts, returning the base dir.
func makeFixture(t *testing.T) string {
	t.Helper()
	base := t.TempDir()

	task := func(dir, name, id, subj, status string, mtime time.Time) {
		body := `{"id":"` + id + `","subject":"` + subj + `","description":"Body of ` + subj +
			`","activeForm":"Working on ` + subj + `","status":"` + status + `","blocks":[],"blockedBy":[]}`
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}

	now := time.Now()
	// Session A (newer): two pending + one in_progress + one completed.
	dirA := filepath.Join(claude.TasksDir(base), "sessA")
	os.MkdirAll(dirA, 0o755)
	task(dirA, "1.json", "1", "Alpha", claude.StatusPending, now.Add(-1*time.Minute))
	task(dirA, "2.json", "2", "Bravo", claude.StatusInProgress, now)
	task(dirA, "3.json", "3", "Charlie", claude.StatusCompleted, now.Add(-2*time.Minute))
	task(dirA, "4.json", "4", "Echo", claude.StatusPending, now.Add(-3*time.Minute))

	// Session B (older): one in_progress task, for the live-feed cross jump.
	dirB := filepath.Join(claude.TasksDir(base), "sessB")
	os.MkdirAll(dirB, 0o755)
	task(dirB, "1.json", "1", "Delta", claude.StatusInProgress, now.Add(-1*time.Hour))

	for _, p := range []struct{ dir, sess, title, cwd string }{
		{"-Users-liu-projA", "sessA", "Session Alpha", "/Users/liu/projA"},
		{"-Users-liu-projB", "sessB", "Session Bravo", "/Users/liu/projB"},
	} {
		pd := filepath.Join(claude.ProjectsDir(base), p.dir)
		os.MkdirAll(pd, 0o755)
		os.WriteFile(filepath.Join(pd, p.sess+".jsonl"),
			[]byte(`{"type":"ai-title","aiTitle":"`+p.title+`","cwd":"`+p.cwd+`"}`), 0o644)
	}
	return base
}

func mkKey(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// send applies a message and runs the resulting command once (good enough for
// our synchronous loadData command).
func send(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	tm, cmd := m.Update(msg)
	m = tm.(Model)
	if cmd != nil {
		if out := cmd(); out != nil {
			tm, _ = m.Update(out)
			m = tm.(Model)
		}
	}
	return m
}

// booted returns a sized model with data loaded.
func booted(t *testing.T, base string, w, h int) Model {
	t.Helper()
	m := New(base, nil)
	m = send(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	m = send(t, m, loadData(base)())
	return m
}

func TestViewBootRendersPanes(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44)

	v := m.View()
	// The main view stacks only the task panes and Preview; Live and Sessions
	// are reached through overlays, not the accordion.
	for _, want := range []string{"[1]─Pending", "[2]─In Progress", "[3]─Completed", "[0]─Preview"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing pane title %q", want)
		}
	}
	for _, absent := range []string{"Live", "Sessions"} {
		if strings.Contains(v, absent) {
			t.Errorf("main view should not show %q pane", absent)
		}
	}
	if m.activeSession != "sessA" {
		t.Errorf("activeSession = %q, want sessA", m.activeSession)
	}
	// Boot lands on the first non-empty task pane (Pending), which shows its tasks.
	if m.focus != FocusTodo {
		t.Errorf("boot focus = %v, want Pending", m.focus)
	}
	if !strings.Contains(v, "Alpha") {
		t.Errorf("expanded Pending pane missing task subject:\n%s", v)
	}
	if got := len(m.todoList.Items()); got != 2 {
		t.Errorf("todo items = %d, want 2", got)
	}
	if got := len(m.doingList.Items()); got != 1 {
		t.Errorf("doing items = %d, want 1", got)
	}
}

func TestFocusCycleStackOrder(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44)
	if m.focus != FocusTodo {
		t.Fatalf("initial focus %v", m.focus)
	}
	// Tab cycles only the three task panes; Preview is excluded from the ring.
	wants := []Focus{FocusDoing, FocusDone, FocusTodo, FocusDoing}
	for _, want := range wants {
		m = send(t, m, mkKey("tab"))
		if m.focus != want {
			t.Errorf("after tab, focus = %v, want %v", m.focus, want)
		}
	}
}

func TestHLNavigatesPanes(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44) // boots on Pending
	if m.focus != FocusTodo {
		t.Fatalf("initial focus %v", m.focus)
	}
	m = send(t, m, mkKey("l"))
	if m.focus != FocusDoing {
		t.Errorf("after l, focus = %v, want In Progress", m.focus)
	}
	m = send(t, m, mkKey("h"))
	if m.focus != FocusTodo {
		t.Errorf("after h, focus = %v, want Pending", m.focus)
	}
	// h wraps from the first task pane to the last (Completed), skipping Preview.
	m = send(t, m, mkKey("h"))
	if m.focus != FocusDone {
		t.Errorf("after wrap h, focus = %v, want Completed", m.focus)
	}
}

func TestNumberJumpAndPreview(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44)

	// 1 -> Pending: preview shows the first pending task (Alpha).
	m = send(t, m, mkKey("1"))
	if m.focus != FocusTodo {
		t.Fatalf("focus = %v, want Todo", m.focus)
	}
	if !m.preview.hasTask || m.preview.task.Subject != "Alpha" {
		t.Errorf("preview task = %+v, want Alpha", m.preview.task)
	}
	// j moves to the second pending task (Echo).
	m = send(t, m, mkKey("j"))
	if m.preview.task.Subject != "Echo" {
		t.Errorf("after j, preview = %q, want Echo", m.preview.task.Subject)
	}
	// 2 -> In Progress: Bravo. 3 -> Completed: Charlie.
	m = send(t, m, mkKey("2"))
	if m.preview.task.Subject != "Bravo" {
		t.Errorf("Doing preview = %q, want Bravo", m.preview.task.Subject)
	}
	m = send(t, m, mkKey("3"))
	if m.preview.task.Subject != "Charlie" {
		t.Errorf("Done preview = %q, want Charlie", m.preview.task.Subject)
	}
	// Enter dives into the Preview pane.
	m = send(t, m, mkKey("enter"))
	if m.focus != FocusPreview {
		t.Errorf("after enter, focus = %v, want Preview", m.focus)
	}
	// Esc returns to the pane matching the previewed task (Done/completed).
	m = send(t, m, mkKey("esc"))
	if m.focus != FocusDone {
		t.Errorf("after esc, focus = %v, want Done", m.focus)
	}
	// 0 jumps straight to the Preview pane.
	m = send(t, m, mkKey("0"))
	if m.focus != FocusPreview {
		t.Errorf("after 0, focus = %v, want Preview", m.focus)
	}
}

func TestLiveOverlayJumpAcrossSessions(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44)
	if got := len(m.liveList.Items()); got != 2 {
		t.Fatalf("live items = %d, want 2 (Bravo, Delta)", got)
	}
	m = send(t, m, mkKey("a")) // open the Active overlay
	if m.overlay != overlayLive {
		t.Fatalf("overlay = %v, want overlayLive", m.overlay)
	}
	m = send(t, m, mkKey("j")) // Bravo (now) then Delta (1h ago) -> Delta
	it, ok := m.liveList.SelectedItem().(liveItem)
	if !ok || it.task.SessionID != "sessB" {
		t.Fatalf("selected live item = %+v", it)
	}
	m = send(t, m, mkKey("space")) // select -> jump and close overlay
	if m.overlay != overlayNone {
		t.Errorf("overlay = %v, want overlayNone after select", m.overlay)
	}
	if m.activeSession != "sessB" {
		t.Errorf("after jump, activeSession = %q, want sessB", m.activeSession)
	}
	if m.focus != FocusDoing { // Delta is in_progress
		t.Errorf("after jump, focus = %v, want Doing", m.focus)
	}
	if !m.preview.hasTask || m.preview.task.Subject != "Delta" {
		t.Errorf("after jump, preview = %+v, want Delta", m.preview.task)
	}
}

func TestSessionOverlaySelectLoadsTasks(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44)
	m = send(t, m, mkKey("s")) // open the Sessions overlay
	if m.overlay != overlaySessions {
		t.Fatalf("overlay = %v, want overlaySessions", m.overlay)
	}
	m = send(t, m, mkKey("j"))     // sessA -> sessB
	m = send(t, m, mkKey("space")) // select -> load and close overlay
	if m.overlay != overlayNone {
		t.Errorf("overlay = %v, want overlayNone after select", m.overlay)
	}
	if m.activeSession != "sessB" {
		t.Errorf("activeSession = %q, want sessB", m.activeSession)
	}
	// Session B has one in_progress task -> focus jumps to its first non-empty
	// pane (In Progress).
	if m.focus != FocusDoing {
		t.Errorf("focus = %v, want Doing", m.focus)
	}
	if len(m.doingList.Items()) != 1 {
		t.Errorf("doing items = %d, want 1", len(m.doingList.Items()))
	}
}

func TestOverlayEscReturnsToMain(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44)
	m = send(t, m, mkKey("a"))
	if m.overlay != overlayLive {
		t.Fatalf("overlay = %v, want overlayLive", m.overlay)
	}
	m = send(t, m, mkKey("esc"))
	if m.overlay != overlayNone {
		t.Errorf("overlay = %v, want overlayNone after esc", m.overlay)
	}
}

func TestFiltersAndTheme(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44)

	// The session filters live inside the Sessions overlay now.
	m = send(t, m, mkKey("s"))
	m = send(t, m, mkKey("a"))
	if !m.activeOnly || len(m.sessions) != 2 {
		t.Errorf("activeOnly=%v sessions=%d", m.activeOnly, len(m.sessions))
	}
	m = send(t, m, mkKey("a"))
	if m.activeOnly {
		t.Error("activeOnly should be off")
	}

	m = send(t, m, mkKey("p"))
	if m.projectFilter == "" {
		t.Error("project filter should be set after p")
	}
	if len(m.sessions) != 1 {
		t.Errorf("project filter should narrow to 1 session, got %d", len(m.sessions))
	}
	m = send(t, m, mkKey("esc"))

	// Theme toggles from the main view.
	before := m.theme.Name
	m = send(t, m, mkKey("t"))
	if m.theme.Name == before {
		t.Errorf("theme did not toggle from %s", before)
	}
}

func TestSizesNoPanic(t *testing.T) {
	base := makeFixture(t)
	for _, dim := range [][2]int{{40, 16}, {80, 30}, {130, 50}} {
		m := booted(t, base, dim[0], dim[1])
		for i := 0; i < paneCount; i++ {
			if v := m.View(); v == "" {
				t.Errorf("%dx%d: empty view", dim[0], dim[1])
			}
			m = send(t, m, mkKey("tab"))
		}
	}
}

func TestHelpOverlay(t *testing.T) {
	m := booted(t, makeFixture(t), 100, 44)
	m = send(t, m, mkKey("?"))
	if m.overlay != overlayHelp {
		t.Fatal("help should be open")
	}
	if !strings.Contains(m.View(), "keys") {
		t.Error("help overlay missing title")
	}
	m = send(t, m, mkKey("esc"))
	if m.overlay != overlayNone {
		t.Error("help should be closed")
	}
}
