# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Running the game

```sh
go run cmd/dungeon.go
```

Then open http://localhost:9001. The server serves static files from `public/` by default.

Flags: `--addr` (default `127.0.0.1:9001`), `--env` (local/production), `--serveFiles` (default true).

## Docker (local)

```sh
make up    # build and start via docker-compose
make stop  # stop
```

## Build

```sh
go build ./cmd/dungeon.go
```

No test suite exists in this project.

## Architecture

**Go backend + Phaser 3 frontend, communicating over WebSocket.**

### Backend layers (`internal/`)

| Package | Role |
|---|---|
| `transport` | WebSocket upgrade, read/write loops per client |
| `lobby` | Client registry, rooms, matchmaking, command routing |
| `game` | Game logic: players, monsters, objects, traps, map, AI |

**Request flow**: WebSocket message → `transport.ServeWebSocketRequest` → `lobby.HandleClientCommand` → routed by `cc.Type` (lobby / room / game) → `game.DispatchGameCommand`.

**Lobby** runs a single goroutine (`Lobby.Run()`) that processes all client register/unregister/command events via channels — no locks needed at the lobby level. The `Game` struct uses its own `sync.Mutex` for all state mutations.

**Game loop** (`Game.StartMainLoop`): two tickers — 60 fps for position updates, ~3 fps for full stats. Monster movement happens server-side in the position tick.

**Map**: Tiled `.tmj` format. Loaded once at startup from `public/assets/dungeon1.tmj`. Processed map is written to `dungeon1_.tmj` for client debug. Spawn positions, objects (chests, triggers, traps), and collision data are all embedded in the map's named layers (`spawns`, `objects`, `walls`).

**Traps** (`game/traps.go`): FSM with states `armed → active → cooldown → armed`. Supports `timer` activators (periodic) and `link` activators (triggered by a named trigger object). Timing is configured as percentages of the cycle period, set via Tiled object properties.

**Classes**: `mage` (150 HP, half damage from fire/explosion/firespot), `knight` (250 HP, half damage from spikes/arrows), `rogue` (200 HP, half damage from bullets).

**Monsters**: `archer`, `skeleton` (spawned from map), `demon` (spawned when all keys collected).

**Events vs. Commands**: Server → client messages are "events" (JSON with a `type` field set by `eventToJSON` in `transport/events.go`). Client → server messages are "commands" with `type` (lobby/room/game) and `subType` fields.

### Frontend (`public/js/`)

Plain JavaScript Phaser 3 (no build step). Files are loaded via `<script>` tags in `public/index.html`:

- `game.js` — Phaser config, scene list
- `game_event_handler.js` — handles all incoming server events
- `players.js`, `monsters.js`, `bullets.js`, `objects.js` — sprite/animation management per entity type

The frontend handles hit detection for projectiles (fireballs, arrows) and sends `HitPlayerCommand` / `HitMonsterCommand` back to the server. The server validates and applies damage.

### Key interfaces (`internal/lobby/`)

- `ClientSender` — transport-layer interface (send, ID, close)
- `ClientPlayer` — lobby/game-layer interface (nickname, additional properties)
- `GameEventsDispatcher` — what `lobby` calls into `game`
- `MatchMaker` — pluggable matchmaking; current implementation (`game.MatchMaker`) assigns players to named rooms

### Deployment

Production Docker image patches `index.html` to use `wss://dungeon.getid.org/ws`. The `version` file content is injected into `index.html` at startup (replaces `%APP_VERSION%`).
