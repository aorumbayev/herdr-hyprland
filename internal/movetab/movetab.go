// Package movetab resolves Hyprland-style movetoworkspace targets against
// herdr tab numbers. herdr tab list can briefly list tabs that are already
// closed after the last pane leaves, so callers must verify a tab still exists.
package movetab

import "strconv"

type Tab struct {
	ID          string
	WorkspaceID string
	Number      int
}

// Plan names an existing tab id or asks for a new tab labeled n.
type Plan struct {
	TabID       string
	NewTabLabel string
}

// Resolve finds tab number n in workspace. When no live tab matches, the pane
// should move to a new tab labeled n.
func Resolve(tabs []Tab, workspace, n string) Plan {
	for _, tab := range tabs {
		if strconv.Itoa(tab.Number) != n {
			continue
		}
		if workspace != "" && tab.WorkspaceID != workspace {
			continue
		}
		return Plan{TabID: tab.ID}
	}
	return Plan{NewTabLabel: n}
}
