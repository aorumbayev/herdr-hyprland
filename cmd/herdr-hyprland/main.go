// herdr-hyprland is a herdr plugin binary. herdr invokes it through the
// manifest: `setup` as an install step, `track` on every tab_focused event,
// and `last-tab` / `move-to-tab N` as keybound actions.
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

	"herdr-hyprland/internal/lasttab"
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
		return errors.New("usage: herdr-hyprland <setup|track|last-tab|move-to-tab N>")
	}
	switch args[0] {
	case "setup":
		return setup()
	case "track":
		return track()
	case "last-tab":
		return focusLastTab()
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

// track handles one tab_focused event: fold it into the state file.
func track() error {
	var envelope struct {
		Data struct {
			WorkspaceID string `json:"workspace_id"`
			TabID       string `json:"tab_id"`
		} `json:"data"`
	}
	raw := os.Getenv("HERDR_PLUGIN_EVENT_JSON")
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return fmt.Errorf("parse HERDR_PLUGIN_EVENT_JSON: %w", err)
	}
	event := envelope.Data
	if event.WorkspaceID == "" || event.TabID == "" {
		return fmt.Errorf("event carries no workspace/tab id: %s", raw)
	}
	state, err := loadState()
	if err != nil {
		return err
	}
	return saveState(lasttab.Observe(state, event.WorkspaceID, event.TabID))
}

func statePath() (string, error) {
	dir := os.Getenv("HERDR_PLUGIN_STATE_DIR")
	if dir == "" {
		return "", errors.New("HERDR_PLUGIN_STATE_DIR is not set; run through herdr")
	}
	return filepath.Join(dir, "lasttab.json"), nil
}

func loadState() (lasttab.State, error) {
	path, err := statePath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return lasttab.State{}, nil
	}
	if err != nil {
		return nil, err
	}
	var state lasttab.State
	if err := json.Unmarshal(raw, &state); err != nil {
		// A corrupt state file heals on the next tab switch.
		return lasttab.State{}, nil
	}
	return state, nil
}

func saveState(state lasttab.State) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// focusLastTab jumps to the workspace's previously focused tab.
func focusLastTab() error {
	workspace := os.Getenv("HERDR_WORKSPACE_ID")
	if workspace == "" {
		return errors.New("HERDR_WORKSPACE_ID is not set")
	}
	state, err := loadState()
	if err != nil {
		return err
	}
	tab, ok := lasttab.Last(state, workspace)
	if !ok {
		return nil // nowhere to go yet; not an error
	}
	_, err = herdr("tab", "focus", tab)
	return err
}

// moveToTab moves the focused pane to the workspace's tab number n,
// creating the tab when it does not exist, like Hyprland's movetoworkspace.
func moveToTab(n string) error {
	pane := os.Getenv("HERDR_PANE_ID")
	if pane == "" {
		pane = os.Getenv("HERDR_ACTIVE_PANE_ID")
	}
	if pane == "" {
		return errors.New("no focused pane id in environment")
	}
	workspace := os.Getenv("HERDR_WORKSPACE_ID")

	out, err := herdr("tab", "list")
	if err != nil {
		return err
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
		return fmt.Errorf("parse tab list: %w", err)
	}
	for _, tab := range env.Result.Tabs {
		if fmt.Sprint(tab.Number) == n && (workspace == "" || tab.WorkspaceID == workspace) {
			_, err := herdr("pane", "move", pane, "--tab", tab.TabID, "--split", "right", "--focus")
			return err
		}
	}
	_, err = herdr("pane", "move", pane, "--new-tab", "--focus")
	return err
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
