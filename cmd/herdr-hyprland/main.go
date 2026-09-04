// herdr-hyprland is a herdr plugin binary. herdr invokes it through the
// manifest: `setup` as an install step and `move-to-tab N` as a keybound action.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"herdr-hyprland/internal/movetab"
	"herdr-hyprland/internal/preset"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-hyprland:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: herdr-hyprland <setup|move-to-tab N>")
	}
	switch args[0] {
	case "setup":
		return setup()
	case "move-to-tab":
		if len(args) != 2 {
			return errors.New("usage: herdr-hyprland move-to-tab <1-9>")
		}
		return moveToTab(args[1])
	}
	return fmt.Errorf("unknown command %q", args[0])
}

// setup patches the user's config.toml with the keymap preset, writing a
// timestamped backup first, the same pattern herdr config reset-keys uses.
func setup() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	current, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	patched, changed := preset.Patch(string(current))
	if !changed {
		fmt.Println("herdr-hyprland: preset already present in", path)
		return nil
	}
	if len(current) > 0 {
		backup := path + ".bak-" + time.Now().Format("20060102-150405")
		if err := os.WriteFile(backup, current, 0o644); err != nil {
			return err
		}
		fmt.Println("herdr-hyprland: backup written to", backup)
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return err
	}
	fmt.Println("herdr-hyprland: keymap preset written to", path)
	fmt.Println("herdr-hyprland: run `herdr server reload-config` or restart herdr to activate")
	return nil
}

func configPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "herdr", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "herdr", "config.toml"), nil
}

// moveToTab moves the focused pane to the workspace's tab number n,
// creating the tab when it does not exist, like Hyprland's movetoworkspace.
func moveToTab(n string) error {
	pane := focusedPaneID()
	if pane == "" {
		return errors.New("no focused pane id in environment")
	}
	workspace := os.Getenv("HERDR_WORKSPACE_ID")

	tabs, err := listTabs()
	if err != nil {
		return err
	}
	plan := movetab.Resolve(tabs, workspace, n)
	if plan.TabID != "" && tabExists(plan.TabID) {
		err := movePaneToTab(pane, plan.TabID)
		if err == nil || !isTabNotFound(err) {
			return err
		}
	}
	label := plan.NewTabLabel
	if label == "" {
		label = n
	}
	return movePaneToNewTab(pane, workspace, label)
}

func focusedPaneID() string {
	if pane := os.Getenv("HERDR_PANE_ID"); pane != "" {
		return pane
	}
	if pane := os.Getenv("HERDR_ACTIVE_PANE_ID"); pane != "" {
		return pane
	}
	raw := os.Getenv("HERDR_PLUGIN_CONTEXT_JSON")
	if raw == "" {
		return ""
	}
	var ctx struct {
		FocusedPaneID string `json:"focused_pane_id"`
	}
	if json.Unmarshal([]byte(raw), &ctx) == nil {
		return ctx.FocusedPaneID
	}
	return ""
}

func listTabs() ([]movetab.Tab, error) {
	out, err := herdr("tab", "list")
	if err != nil {
		return nil, err
	}
	var env struct {
		Result struct {
			Tabs []struct {
				TabID       string `json:"tab_id"`
				WorkspaceID string `json:"workspace_id"`
				Number      int    `json:"number"`
			} `json:"tabs"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("parse tab list: %w", err)
	}
	tabs := make([]movetab.Tab, len(env.Result.Tabs))
	for i, tab := range env.Result.Tabs {
		tabs[i] = movetab.Tab{
			ID:          tab.TabID,
			WorkspaceID: tab.WorkspaceID,
			Number:      tab.Number,
		}
	}
	return tabs, nil
}

func tabExists(tabID string) bool {
	_, err := herdr("tab", "get", tabID)
	return err == nil
}

func movePaneToTab(pane, tabID string) error {
	_, err := herdr("pane", "move", pane, "--tab", tabID, "--split", "right", "--focus")
	return err
}

func movePaneToNewTab(pane, workspace, label string) error {
	args := []string{"pane", "move", pane, "--new-tab", "--label", label, "--focus"}
	if workspace != "" {
		args = append(args, "--workspace", workspace)
	}
	_, err := herdr(args...)
	return err
}

func isTabNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "tab_not_found")
}

// herdr calls the herdr CLI. Plugins run with a minimal PATH, so
// HERDR_BIN_PATH wins over the bare name.
func herdr(args ...string) ([]byte, error) {
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		bin = "herdr"
	}
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("herdr %s: %s", args[0], strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("herdr %s: %w", args[0], err)
	}
	return out, nil
}
