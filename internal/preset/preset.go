// Package preset patches herdr's config.toml with the Hyprland-style
// keymap. Native herdr actions get alt chords in [keys]; the plugin's own
// actions get [[keys.command]] bindings. The patch is additive: every entry
// keeps herdr's default prefix binding, and a key the user already set in
// [keys] is left alone.
package preset

import "strings"

const marker = "# herdr-hyprland"

// key is one native [keys] entry with its chords.
type key struct {
	name   string
	chords []string
}

// Chord lists keep herdr's default binding first, so the preset never
// removes a prefix+* default.
var nativeKeys = []key{
	{"focus_pane_left", []string{"prefix+h", "alt+left"}},
	{"focus_pane_down", []string{"prefix+j", "alt+down"}},
	{"focus_pane_up", []string{"prefix+k", "alt+up"}},
	{"focus_pane_right", []string{"prefix+l", "alt+right"}},
	{"switch_tab", []string{"prefix+1..9", "alt+1..9"}},
	{"switch_workspace", []string{"prefix+shift+1..9", "alt+ctrl+1..9"}},
}

// Patch returns the config with the preset applied and whether it changed
// anything. A config that already carries the preset marker comes back
// unchanged.
func Patch(config string) (string, bool) {
	if strings.Contains(config, marker) {
		return config, false
	}
	out := insertNativeKeys(config)
	out = appendCommands(out)
	return out, true
}

// insertNativeKeys puts the preset's [keys] entries into the existing [keys]
// table, or appends one. TOML rejects a duplicate [keys] header, so merging
// is the only safe move. Keys the user already set are skipped.
func insertNativeKeys(config string) string {
	var b strings.Builder
	b.WriteString(marker + " keys (user lines above win; edit or delete freely)\n")
	taken := userSetKeys(config)
	claimed := userCommandChords(config)
	for _, k := range nativeKeys {
		if taken[k.name] {
			continue
		}
		// An explicit [keys] entry outranks [[keys.command]], so a default
		// chord the user's own command binds must not be re-stated here.
		var chords []string
		for _, chord := range k.chords {
			if !claimed[chord] {
				chords = append(chords, chord)
			}
		}
		if len(chords) > 0 {
			b.WriteString(k.name + " = " + tomlChords(chords) + "\n")
		}
	}
	block := b.String()

	lines := strings.SplitAfter(config, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "[keys]" {
			// Insert at the end of the [keys] table, before the next header.
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if isTableHeader(lines[j]) {
					end = j
					break
				}
			}
			return strings.Join(lines[:end], "") + block + strings.Join(lines[end:], "")
		}
	}
	return ensureTrailingNewline(config) + "\n[keys]\n" + block
}

// userSetKeys names every key the user assigned inside the [keys] table.
// ponytail: line-level scan, no TOML parser. Dotted `keys.x = ...` outside
// the table and inline tables are not detected; herdr's own docs only ever
// write the [keys] table form.
func userSetKeys(config string) map[string]bool {
	taken := map[string]bool{}
	in := false
	for _, line := range strings.Split(config, "\n") {
		trimmed := strings.TrimSpace(line)
		if isTableHeader(line) {
			in = trimmed == "[keys]"
			continue
		}
		if !in {
			continue
		}
		if name, _, found := strings.Cut(trimmed, "="); found {
			taken[strings.TrimSpace(name)] = true
		}
	}
	return taken
}

// userCommandChords names every chord the user bound in a [[keys.command]]
// entry. `key = "..."` assignments only appear in those tables.
func userCommandChords(config string) map[string]bool {
	claimed := map[string]bool{}
	for _, line := range strings.Split(config, "\n") {
		trimmed := strings.TrimSpace(line)
		if rest, found := strings.CutPrefix(trimmed, "key ="); found {
			claimed[strings.Trim(strings.TrimSpace(rest), `"`)] = true
		}
	}
	return claimed
}

func tomlChords(chords []string) string {
	if len(chords) == 1 {
		return `"` + chords[0] + `"`
	}
	return `["` + strings.Join(chords, `", "`) + `"]`
}

func isTableHeader(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "[")
}

func appendCommands(config string) string {
	var b strings.Builder
	b.WriteString(ensureTrailingNewline(config))
	b.WriteString("\n" + marker + " commands (edit or delete freely)\n")
	for n := 1; n <= 9; n++ {
		digit := string(rune('0' + n))
		b.WriteString("[[keys.command]]\n")
		b.WriteString(`key = "alt+shift+` + digit + `"` + "\n")
		b.WriteString(`type = "plugin_action"` + "\n")
		b.WriteString(`command = "herdr-hyprland.move-to-tab-` + digit + `"` + "\n")
		b.WriteString(`description = "move pane to tab ` + digit + `"` + "\n\n")
	}
	return b.String()
}

func ensureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
