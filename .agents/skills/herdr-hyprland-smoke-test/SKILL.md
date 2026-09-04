---
name: herdr-hyprland-smoke-test
description: Throwaway herdr instance in tmux for end-to-end smoke tests of the herdr-hyprland plugin, isolated from the user's live herdr panes and config. Use when the user asks to smoke-test the plugin, run e2e tests against a real herdr, probe the keymap, or tear the sandbox down. For development of this herdr-hyprland repository.
---

# herdr-hyprland smoke-test sandbox

Run the plugin against a second herdr instance. A broken build or a bad
keybinding must not damage the herdr panes or the config.toml the user is
using.

**Shared binary limit:** `up` runs `go build -o bin/herdr-hyprland
./cmd/herdr-hyprland` in this checkout. The user's live herdr picks up the
same binary on its next plugin action. Config, plugin registry, socket,
state, and panes are isolated. The binary is not.

**Platforms:** POSIX/tmux-only (Linux and macOS).

## Bring it up

```bash
repo_root=$(git rev-parse --show-toplevel)
bash "$repo_root/.agents/skills/herdr-hyprland-smoke-test/scripts/sandbox.sh" up
```

`up` is idempotent only when `/tmp/hhl-sandbox` is absent or already owned
(complete `.hhl-sandbox-owned` sentinel: marker, session, plugin_root). It:

1. Claims `/tmp/hhl-sandbox` with an exclusive `mkdir`, or validates the
   sentinel and reuses. Refuses unknown contents.
2. Seeds a sandbox `config.toml` with `allow_nested = true` (herdr refuses
   to launch inside a herdr-managed pane otherwise).
3. Builds the plugin binary (**shared binary mutation**), links the plugin
   and runs `setup` against the sandbox config — both before the server
   starts, so boot picks them up and no reload ordering matters.
4. Starts herdr in fixed tmux session `hhl-sandbox` (200x50) on its own
   socket. Kill only when that session's `HERDR_SOCKET_PATH` exactly matches
   the sandbox socket; a sentinel alone is never enough.
5. Verifies: `config check` passes, the plugin is registered, and
   `reload-config` reports zero diagnostics.

Other actions: `sandbox.sh status`, `sandbox.sh down` (stops the server,
kills the tmux session only on socket match, deletes `/tmp/hhl-sandbox` only
after the sentinel validator passes).

## Run every command through `hsb`

`hsb` is generated at `/tmp/hhl-sandbox/bin/hsb`. It runs any command with
the sandbox env and cleared inherited HERDR overrides.

```bash
S=/tmp/hhl-sandbox
$S/bin/hsb herdr pane list
$S/bin/hsb herdr plugin log list --plugin herdr-hyprland
```

A bare `herdr` talks to the user's live instance. There is no warning and no
undo — a `pane close` lands in their real session. Never omit `hsb`.

## Driving the keymap

The whole point of this plugin is keybindings, so drive the real TUI:

- `tmux send-keys -t hhl-sandbox M-Left` presses `alt+left`. Alt chords are
  `M-<key>`; `alt+shift+3` arrives as the shifted character (`M-#` on a US
  layout); `alt+ctrl+1` is `C-M-1`.
- Observe results headlessly between keypresses:
  `hsb herdr pane list`, `hsb herdr tab list`, `hsb herdr pane layout --pane <id>`.
- Read the screen: `tmux capture-pane -p -t hhl-sandbox`.
- Watch live: `tmux attach -t hhl-sandbox` (detach with `Ctrl-b d`).
- Give herdr ~0.5s after each keypress before reading state.

## What is and is not isolated

| Isolated                                                  | Shared with the user's herdr        |
| --------------------------------------------------------- | ----------------------------------- |
| config.toml, keybindings, plugin registry, plugin config  | the compiled `bin/herdr-hyprland`   |
| socket, server process, panes, workspaces, plugin state   | this checkout's working tree        |

Say so in the report if a test leaves the shared binary in a broken state.

## After `up`, stop and ask

Report the sandbox status and wait for the user to name what to smoke test,
unless they already did in the same request. Capture real output; a binding
that fired but did the wrong thing is a finding, not a retry.

## Teardown after testing

```bash
repo_root=$(git rev-parse --show-toplevel)
bash "$repo_root/.agents/skills/herdr-hyprland-smoke-test/scripts/sandbox.sh" down
```

Do not leave the sandbox running. A lingering server or tmux session is
stale state that blocks or confuses the next run.

## Gotchas seen for real

- herdr refuses to launch inside a herdr-managed pane; the sandbox config
  sets `[experimental] allow_nested = true`. That is why `up` must never
  overwrite an existing sandbox `config.toml`.
- Do not start the TUI with `HERDR_CONFIG_PATH` empty. herdr treats that as
  “no config file” and the keybind overlay stays on compiled defaults. The
  wrapper sets it to `/tmp/hhl-sandbox/config/herdr/config.toml`.
- `plugin link` and `setup` both work with no server running; doing them
  before `start_server` avoids the reload-config ordering problem entirely.
- The tmux pane must be big enough to split (`-x 200 -y 50`); herdr answers
  `ghostty error -2` to a split in a too-small pane.
