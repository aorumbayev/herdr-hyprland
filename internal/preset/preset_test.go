package preset

import (
	"strings"
	"testing"
)

// The patcher edits ~/.config/herdr/config.toml text. It must be additive
// (herdr defaults stay), respect keys the user already set, and be a no-op
// on a config it already patched.

func TestPatchingAnEmptyConfigAddsKeysAndCommands(t *testing.T) {
	got, changed := Patch("")
	if !changed {
		t.Fatal("Patch reported no change on an empty config")
	}
	for _, want := range []string{
		"[keys]",
		`focus_pane_left = ["prefix+h", "alt+left"]`,
		`switch_tab = ["prefix+1..9", "alt+1..9"]`,
		`switch_workspace = ["prefix+shift+1..9", "alt+ctrl+1..9"]`,
		"[[keys.command]]",
		`key = "alt+shift+3"`,
		`command = "herdr-hyprland.move-to-tab-3"`,
		`type = "plugin_action"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("patched config is missing %q", want)
		}
	}
	for _, gone := range []string{
		"alt+h",
		"next_tab",
		"previous_tab",
		"next_workspace",
		"previous_workspace",
		"swap_pane",
		"resize_pane",
		"alt+z",
		"alt+w",
		"last-tab",
		"alt+ctrl+tab",
	} {
		if strings.Contains(got, gone) {
			t.Errorf("patched config should not contain %q", gone)
		}
	}
}

func TestPatchingTwiceChangesNothing(t *testing.T) {
	once, _ := Patch("")
	twice, changed := Patch(once)
	if changed || twice != once {
		t.Fatal("Patch changed an already patched config")
	}
}

func TestUserContentSurvivesUntouched(t *testing.T) {
	in := "# my config\n[theme]\nname = \"dark\"\n"
	got, _ := Patch(in)
	if !strings.HasPrefix(got, in) {
		t.Fatalf("patched config does not start with the original content:\n%s", got)
	}
}

// TOML rejects a second [keys] table, so the patcher must merge into an
// existing one instead of appending its own.
func TestMergesIntoAnExistingKeysTable(t *testing.T) {
	in := "[keys]\ngoto = \"prefix+g\"\n\n[theme]\nname = \"dark\"\n"
	got, _ := Patch(in)
	if strings.Count(got, "[keys]\n") != 1 {
		t.Fatalf("patched config defines [keys] more than once:\n%s", got)
	}
	if !strings.Contains(got, "goto = \"prefix+g\"") {
		t.Fatal("patched config lost the user's own [keys] entry")
	}
	if !strings.Contains(got, `switch_tab = ["prefix+1..9", "alt+1..9"]`) {
		t.Fatal("patched config did not gain the preset keys")
	}
}

func TestAKeyTheUserAlreadySetWins(t *testing.T) {
	in := "[keys]\nswitch_tab = \"prefix+t\"\n"
	got, _ := Patch(in)
	if !strings.Contains(got, "switch_tab = \"prefix+t\"") {
		t.Fatal("patched config lost the user's switch_tab binding")
	}
	if strings.Contains(got, `"alt+1..9"`) {
		t.Fatal("patched config overrode the user's switch_tab binding with the preset")
	}
}

// Explicit [keys] entries outrank [[keys.command]] bindings in herdr, so
// re-stating a default chord like prefix+k would disable a user command the
// default had been ceding it to. A chord the user's [[keys.command]] uses
// must not appear in the preset.
func TestADefaultChordUsedByAUserCommandIsNotRestated(t *testing.T) {
	in := "[[keys.command]]\nkey = \"prefix+k\"\ntype = \"plugin_action\"\ncommand = \"x.y\"\n"
	got, _ := Patch(in)
	if !strings.Contains(got, `focus_pane_up = "alt+up"`) {
		t.Fatalf("focus_pane_up should drop prefix+k and keep the alt chord:\n%s", got)
	}
	if strings.Count(got, `"prefix+k"`) != 1 {
		t.Fatal("the preset re-stated prefix+k even though a user command binds it")
	}
}
