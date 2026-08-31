# AGENTS.md

herdr-hyprland: Hyprland-style pane/tab keys as a herdr plugin. One Go module, one binary (`cmd/herdr-hyprland`).

## Commands

```sh
make setup            # installs git hooks (.githooks) — do this once
go build -o bin/herdr-hyprland ./cmd/herdr-hyprland
go test ./...
```

Verify before pushing — CI (`.github/workflows/verify.yml`) runs on linux and macos, in this order:
`gofmt -l .` (must be empty) → `go vet ./...` → `go test ./...` → `go build ./cmd/herdr-hyprland`. On linux only, CI also checks every commit message.

`bin/herdr-hyprland setup` patches `config.toml` with the alt keymap. Plugin link does not run that step by itself.

## Commits

- Conventional Commits.
- The commit-msg hook and CI **reject AI attribution trailers** (`Co-authored-by: Claude`, `Generated with ...`, etc. — see `scripts/check-commit-message.sh`). Never add them.

## Architecture

- `internal/preset` — additive `config.toml` patcher.
- `internal/lasttab` — per-workspace previous-tab history.
- `cmd/herdr-hyprland` — `setup`, `track`, `last-tab`, `move-to-tab N`.
- Native herdr actions are config only. Go is only for last-tab and move-to-tab.

## Style

- README.md is written in short declarative sentences. Match that register when editing docs.
- Release builds are not a thing yet: install is `go build` in the plugin manifest. Keep the code pure Go, no cgo.
