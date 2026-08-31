// Package lasttab tracks, per herdr workspace, which tab was focused before
// the current one. It is the state behind the plugin's last-tab action,
// herdr's answer to Hyprland's `workspace previous`.
package lasttab

// State maps a workspace id to its tab focus history. The zero value is
// ready to use. It marshals to JSON as-is for the plugin state file.
type State map[string]History

// History is the current and previous focused tab of one workspace.
type History struct {
	Current  string `json:"current"`
	Previous string `json:"previous"`
}

// Observe records that tabID is now focused in workspaceID.
func Observe(s State, workspaceID, tabID string) State {
	if s == nil {
		s = State{}
	}
	h := s[workspaceID]
	if h.Current == tabID {
		return s
	}
	s[workspaceID] = History{Current: tabID, Previous: h.Current}
	return s
}

// Last returns the tab focused before the current one in workspaceID.
func Last(s State, workspaceID string) (string, bool) {
	h := s[workspaceID]
	return h.Previous, h.Previous != ""
}
