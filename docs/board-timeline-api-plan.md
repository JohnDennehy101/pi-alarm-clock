# Board timeline CRUD API — plan spec

Approved plan for managing board-timeline configuration at runtime: full CRUD
for presets and per-view custom duration/transition (refresh) settings via the
authenticated API. Changes apply to the next cast or alarm fire without a
server restart.

## Storage: yml stays the single source of truth

`data/board-timeline.yml` is rewritten in canonical form on every mutation.
The first API write replaces any hand-written file (structure and comments are
lost from that point — accepted tradeoff). DB tables were considered and
rejected: a migration plus repository plus merge layer above the yml builtins
would add three config layers for no benefit, and the preview tool keeps
working unchanged since it already reads this file.

Built-in presets live in code and are never written to disk; the file only
persists user deltas.

## API surface (all under the existing bearer-token auth)

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/board/timeline` | Views (duration/refresh), presets (with `builtIn` flag), default order, available view names |
| PUT | `/api/v1/board/timeline/views/{view}` | Partial update: `{durationSeconds?, refreshSeconds?}` — custom transition per view |
| PUT | `/api/v1/board/timeline/order` | Set global default order: preset name or view list |
| GET | `/api/v1/board/timeline/presets` | List presets |
| POST | `/api/v1/board/timeline/presets` | Create a preset — `{ "name": "morning", "order": ["bus.map", "weather"] }`; 201 created, 409 if the name exists |
| GET | `/api/v1/board/timeline/presets/{name}` | Read one preset; 404 unknown |
| PUT | `/api/v1/board/timeline/presets/{name}` | Replace preset (builtin name = override) |
| DELETE | `/api/v1/board/timeline/presets/{name}` | Delete preset; 204, 404 unknown, 400 for builtins |

Create example:

```bash
curl -sk -X POST https://localhost:8080/api/v1/board/timeline/presets \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"morning","order":["bus.map","weather","bus.progress"]}'
```

Alarms reference presets through the existing `boardTimeline` chromecast
setting: `"boardTimeline": "morning"`.

## Domain changes

- `internal/app/board_timeline.go`: `clone()`, `setViewSettings`,
  `putPreset`, `deletePreset`, `setOrder`, plus `serializeCatalog` producing
  canonical yml (grouped keys — `bus: {map: {duration: 3m}}` — which the
  loader accepts, so write and read are a verified round-trip).
- `BoardService` gains settings methods on top of `ResolveTimeline`: each
  does validate → clone → mutate → write file (tmp + rename) → atomically
  swap the `b.catalog` pointer. No server restart; the next cast or alarm
  fire resolves through the new catalog. In-flight casts keep their
  already-resolved timeline for the session.
- Startup-invalid file: the catalog is nil today (server runs on built-in
  defaults); the first API mutation rebuilds from defaults and overwrites
  the broken file, so the API doubles as config recovery.

## Validation

- View names from the known set (`weather`, `bus`/`schedule`, `bus.map`,
  `bus.progress`, and aliases); unknown names return 400 listing what is
  available.
- Duration 1s–60min; refresh has a 10s floor and is capped at that view's
  duration (existing rules).
- Preset names normalized lowercase, `^[a-z0-9][a-z0-9-]{0,31}$`; orders
  of 1–20 known views.
- Builtins (`default`, `interleaved`, `bus-focus`): PUT overrides (existing
  yml semantics), DELETE rejected.

## Tests

App layer: clone/mutate/serialize round-trip, persist-then-reload via temp
dir, validation tables, builtin-delete rejection. API layer: plain
`httptest` against a real `BoardService` over `t.TempDir()` — happy paths
plus 400/404/409.

## Out of scope

- Transition animations (fades etc.) — "transition" here is the refresh
  cadence.
- Flutter frontend.

## Files touched

- `server/internal/app/board_timeline.go` (+ likely a new
  `server/internal/app/board_settings.go`)
- `server/internal/app/board.go`
- `server/internal/app/board_timeline_test.go` (+ new test file)
- `server/internal/api/board_timeline.go` (new)
- `server/internal/api/router.go`
- `README.md`

## Implementation phases

1. Catalog mutation + serialization with tests.
2. `BoardService` settings methods.
3. API handlers + router.
4. README + curl smoke against the local API.
