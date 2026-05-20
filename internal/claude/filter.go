package claude

import "sort"

// ProjectList returns the distinct, sorted project names present across the
// given sessions. Sessions with no project are ignored.
func ProjectList(sessions []Session) []string {
	seen := make(map[string]struct{})
	for _, s := range sessions {
		if s.Project != "" {
			seen[s.Project] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// ApplyFilters returns the sessions matching the given filters, preserving input
// order. An empty project matches all projects. When activeOnly is set, only
// sessions with at least one in-progress task are kept.
func ApplyFilters(sessions []Session, project string, activeOnly bool) []Session {
	out := make([]Session, 0, len(sessions))
	for _, s := range sessions {
		if project != "" && s.Project != project {
			continue
		}
		if activeOnly && s.InProgress == 0 {
			continue
		}
		out = append(out, s)
	}
	return out
}
