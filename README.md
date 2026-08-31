<h3 align="center">
  Hyprland Keys
</h3>

<p align="center">Hyprland's window-management keymap for herdr panes and tabs. No Hyprland required — or involved.</p>

---

herdr-hyprland is a herdr plugin. It ports Hyprland's window-management *semantics* into herdr, the way vscode-neovim ports vim semantics into VSCode. It works on any OS herdr runs on. There is no Hyprland process, no `hyprctl`, no window manager integration.

The mapping: a herdr **tab** is a Hyprland **workspace**, a herdr **pane** is a **window**, a herdr **workspace** is a **monitor**.

## Install

herdr 0.8.2 or later and Go are required (the install step runs `go build`).

```bash
herdr plugin install aorumbayev/herdr-hyprland
```

The install patches `~/.config/herdr/config.toml` with the keymap below and writes a timestamped backup beside it. The patch is additive: every herdr `prefix+*` default keeps working, and any key you already set in `[keys]` is left alone. To undo, delete the `# herdr-hyprland` blocks from `config.toml`, restore the backup, or run `herdr config reset-keys` (removes all custom keybindings).

After install, run `herdr server reload-config` or restart herdr.

## Keymap

Arrow keys and `hjkl` both work everywhere they appear.

| Chord | Action | Hyprland twin |
|---|---|---|
| `alt+arrows` / `alt+hjkl` | focus pane | `movefocus` |
| `alt+shift+arrows` / `alt+shift+hjkl` | swap pane | `swapwindow` |
| `alt+ctrl+arrows` | resize pane | `resizeactive` |
| `alt+1..9` | focus tab N | `workspace N` |
| `alt+shift+1..9` | move pane to tab N | `movetoworkspace N` |
| `alt+n` / `alt+p` | next / previous tab | `workspace e+1/e-1` |
| `alt+ctrl+tab` | previously focused tab | `workspace previous` |
| `alt+z` | zoom pane | `fullscreen` |
| `alt+w` | close pane | `killactive` |

Moving a pane to a tab that does not exist creates a new tab, like Hyprland. The new tab takes the next free number, which may differ from the number you pressed. Focus follows the moved pane.

`togglesplit` is not here. herdr's `layout.apply` recreates panes instead of rearranging them, which would kill whatever runs in them. It returns if herdr grows a non-destructive way to flip a split.

## Known key collisions

No chord set is free on every OS, WM, shell, and TUI at once. Every chord here is a starting point: override any of them in `config.toml`, your line wins.

- `alt+1..9` is readline's digit-argument.
- `alt+n` / `alt+p` overlap zsh's `ESC-n`/`ESC-p` history search.
- `alt+w` is emacs' copy key; the plugin steals it inside an emacs pane.
- On Linux under Hyprland/Omarchy, bare-ALT chords reach the terminal except `ALT+TAB`/`ALT+SHIFT+TAB` (which this plugin deliberately does not use).

Why `alt` and not `super`: on Linux the window manager eats `super` before any terminal sees it, and `super` only reaches herdr through the kitty keyboard protocol. `alt` reaches every terminal.

## Development

```bash
go test ./...
go build -o bin/herdr-hyprland ./cmd/herdr-hyprland
herdr plugin link .
bin/herdr-hyprland setup   # plugin link does not run build steps
```

The plugin binary has four commands: `setup` (patch config.toml), `track` (event hook on `tab.focused`, records tab focus history per workspace), `last-tab`, and `move-to-tab N`.
