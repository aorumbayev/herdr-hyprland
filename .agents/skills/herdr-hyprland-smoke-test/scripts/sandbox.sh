#!/usr/bin/env bash
# Isolated herdr instance for e2e smoke testing herdr-hyprland.
# Limitation: `up` runs `go build -o bin/herdr-hyprland ./cmd/herdr-hyprland`
# in this checkout, so bin/herdr-hyprland is shared with the user's live
# plugin link — not a private binary. Config, plugin registry, socket, state,
# and panes are isolated.
set -euo pipefail

CANONICAL_SANDBOX="/tmp/hhl-sandbox"
SENTINEL_NAME=".hhl-sandbox-owned"
FIXED_SESSION="hhl-sandbox"

SANDBOX="$CANONICAL_SANDBOX"
PLUGIN_ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"

CFG="$SANDBOX/config"
CONFIG_TOML="$CFG/herdr/config.toml"
BIN="$SANDBOX/bin"
SOCK="$SANDBOX/herdr.sock"
CLIENT_SOCK="$SANDBOX/herdr-client.sock"
HSB="$BIN/hsb"
PLUGIN_STATE="$SANDBOX/state/herdr/plugins/herdr-hyprland"
SENTINEL="$SANDBOX/$SENTINEL_NAME"

# Drop inherited herdr/config/socket/bin/plugin overrides that would leak into the live instance.
ISOLATE_UNSET=(
  HERDR_ENV HERDR_PANE_ID HERDR_TAB_ID HERDR_WORKSPACE_ID HERDR_PLUGIN_CONTEXT_JSON
  HERDR_CONFIG_PATH HERDR_BIN_PATH HERDR_SOCKET_PATH HERDR_CLIENT_SOCKET_PATH
  HERDR_PLUGIN_CONFIG_DIR HERDR_PLUGIN_STATE_DIR HERDR_PLUGIN_DIR
)

assert_owned_sentinel() {
  local root="${1:-$SANDBOX}"
  local sentinel="$root/$SENTINEL_NAME"
  if [ ! -f "$sentinel" ] || [ -L "$sentinel" ]; then
    echo "refusing: $root exists without ownership sentinel $sentinel" >&2
    exit 2
  fi
  if ! grep -Fqx 'hhl-sandbox' "$sentinel"; then
    echo "refusing: sentinel does not claim hhl-sandbox ownership" >&2
    exit 2
  fi
  if ! grep -Fqx "session=$FIXED_SESSION" "$sentinel"; then
    echo "refusing: sentinel session identity mismatch (want session=$FIXED_SESSION)" >&2
    exit 2
  fi
  if ! grep -Fqx "plugin_root=$PLUGIN_ROOT" "$sentinel"; then
    echo "refusing: sentinel plugin_root mismatch (want plugin_root=$PLUGIN_ROOT)" >&2
    exit 2
  fi
}

write_sentinel() {
  printf 'hhl-sandbox\nsession=%s\nplugin_root=%s\n' "$FIXED_SESSION" "$PLUGIN_ROOT" > "$SENTINEL"
}

# Exclusive mkdir claims an absent path. Existing trees must carry a complete
# sentinel. Never claim or overwrite unknown contents.
claim_or_reuse_sandbox() {
  if mkdir "$SANDBOX" 2>/dev/null; then
    write_sentinel
  elif [ -e "$SANDBOX" ]; then
    assert_owned_sentinel
  else
    echo "refusing: cannot create sandbox directory $SANDBOX" >&2
    exit 2
  fi
}

kill_sandbox_tmux() {
  if ! tmux has-session -t "$FIXED_SESSION" 2>/dev/null; then
    return 0
  fi
  # Kill only when the session's HERDR_SOCKET_PATH exactly matches this
  # sandbox socket. A same-named session may belong to something else.
  local sock_env
  sock_env="$(tmux show-environment -t "$FIXED_SESSION" HERDR_SOCKET_PATH 2>/dev/null || true)"
  case "$sock_env" in
    "HERDR_SOCKET_PATH=$SOCK")
      tmux kill-session -t "$FIXED_SESSION" 2>/dev/null || true
      ;;
    *)
      echo "refusing tmux kill-session: $FIXED_SESSION HERDR_SOCKET_PATH does not match $SOCK" >&2
      exit 2
      ;;
  esac
}

isolated() {
  local -a unset_flags=()
  local v
  for v in "${ISOLATE_UNSET[@]}"; do
    unset_flags+=(-u "$v")
  done
  env "${unset_flags[@]}" \
      HERDR_PLUGIN_STATE_DIR="$PLUGIN_STATE" \
      XDG_CONFIG_HOME="$CFG" XDG_STATE_HOME="$SANDBOX/state" XDG_BIN_HOME="$BIN" \
      HERDR_CONFIG_PATH="$CONFIG_TOML" \
      HERDR_SOCKET_PATH="$SOCK" HERDR_CLIENT_SOCKET_PATH="$CLIENT_SOCK" \
      PATH="$BIN:$PATH" "$@"
}

server_running() {
  local out
  out="$(isolated herdr status 2>/dev/null)" || return 1
  case "$out" in *'status: running'*) return 0 ;; *) return 1 ;; esac
}

write_wrapper() {
  mkdir -p "$BIN"
  local unset_args=""
  local v
  for v in "${ISOLATE_UNSET[@]}"; do
    unset_args+=" -u $v"
  done
  cat > "$HSB" <<EOF
#!/usr/bin/env bash
# Runs any command against the sandbox herdr instead of the user's live one.
exec env$unset_args \\
  HERDR_PLUGIN_STATE_DIR="$PLUGIN_STATE" \\
  XDG_CONFIG_HOME="$CFG" XDG_STATE_HOME="$SANDBOX/state" XDG_BIN_HOME="$BIN" \\
  HERDR_CONFIG_PATH="$CONFIG_TOML" \\
  HERDR_SOCKET_PATH="$SOCK" HERDR_CLIENT_SOCKET_PATH="$CLIENT_SOCK" \\
  PATH="$BIN:\$PATH" "\$@"
EOF
  chmod +x "$HSB"
}

seed_config() {
  mkdir -p "$CFG/herdr"
  # allow_nested: the sandbox herdr runs inside a herdr-managed pane, which
  # herdr refuses by default. Written once so setup's patch survives reruns.
  [ -f "$CFG/herdr/config.toml" ] ||
    printf '[experimental]\nallow_nested = true\n' > "$CFG/herdr/config.toml"
}

start_server() {
  if server_running; then return 0; fi
  kill_sandbox_tmux
  # Empty HERDR_CONFIG_PATH= is not unset: herdr then skips the sandbox
  # config.toml and the TUI keeps compiled default keys. Point at the file.
  # env -u drops live-pane HERDR_* inherited from the outer herdr session.
  tmux new-session -d -s "$FIXED_SESSION" -x 200 -y 50 -c "$SANDBOX" \
    -e XDG_CONFIG_HOME="$CFG" -e XDG_STATE_HOME="$SANDBOX/state" -e XDG_BIN_HOME="$BIN" \
    -e HERDR_PLUGIN_STATE_DIR="$PLUGIN_STATE" \
    -e HERDR_SOCKET_PATH="$SOCK" -e HERDR_CLIENT_SOCKET_PATH="$CLIENT_SOCK" \
    "env -u HERDR_PANE_ID -u HERDR_TAB_ID -u HERDR_WORKSPACE_ID -u HERDR_ENV \
      -u HERDR_PLUGIN_CONTEXT_JSON -u HERDR_BIN_PATH -u HERDR_PLUGIN_CONFIG_DIR \
      -u HERDR_PLUGIN_DIR \
      HERDR_CONFIG_PATH=$CONFIG_TOML \
      PATH=$BIN:\$PATH herdr; echo '[sandbox herdr exited]'; sleep 86400"
  for _ in $(seq 1 40); do
    server_running && return 0
    sleep 0.5
  done
  echo "sandbox herdr did not come up; tmux capture-pane -p -t $FIXED_SESSION" >&2
  tmux capture-pane -p -t "$FIXED_SESSION" 2>/dev/null | head -20 >&2
  return 1
}

verify() {
  isolated herdr config check
  local plugins
  plugins="$(isolated herdr plugin list 2>&1)"
  case "$plugins" in
    *herdr-hyprland*) ;;
    *) echo "self-check failed: plugin not registered in sandbox" >&2; return 1 ;;
  esac
  local reload
  reload="$(isolated herdr server reload-config 2>&1)"
  case "$reload" in
    *'"diagnostics":[]'*) echo "self-check ok: config loads with no diagnostics" ;;
    *) echo "$reload" >&2; echo "self-check failed: reload diagnostics not empty" >&2; return 1 ;;
  esac
}

up() {
  claim_or_reuse_sandbox
  seed_config
  write_wrapper
  # Shared-binary limitation: rebuilds this checkout's bin/herdr-hyprland,
  # which the user's live plugin link also runs.
  (cd "$PLUGIN_ROOT" && go build -o bin/herdr-hyprland ./cmd/herdr-hyprland)
  # Register + patch before the server starts, so boot picks both up and no
  # reload ordering matters. Both work with no server running.
  isolated herdr plugin link "$PLUGIN_ROOT" >/dev/null
  (cd "$PLUGIN_ROOT" && isolated bin/herdr-hyprland setup)
  start_server
  verify
  status
}

down() {
  assert_owned_sentinel
  if server_running; then isolated herdr server stop >/dev/null 2>&1 || true; fi
  kill_sandbox_tmux
  rm -rf -- "$SANDBOX"
  echo "sandbox removed: $SANDBOX"
}

status() {
  echo "sandbox:     $SANDBOX"
  echo "wrapper:     $HSB <any command>"
  echo "tmux:        tmux attach -t $FIXED_SESSION"
  echo "limitation:  shared checkout binary (go build mutates bin/herdr-hyprland)"
  printf 'tmux alive:  '; tmux has-session -t "$FIXED_SESSION" 2>/dev/null && echo yes || echo no
  printf 'server:      '; server_running && echo running || echo "not running"
  echo "plugins:"; isolated herdr plugin list 2>&1 | head -c 400 | sed 's/^/  /'; echo
}

case "${1:-up}" in
  up) up ;;
  down) down ;;
  status) status ;;
  *) echo "usage: sandbox.sh [up|down|status]" >&2; exit 2 ;;
esac
