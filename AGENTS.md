# Repository Agent Instructions
This file provides guidance to AI coding agents when working with code in this repository.

## Project

Steam CLI is a Go command-line tool that queries public Steam Store, Steam Web API, Steam Community, Steam CDN, and Steamworks event sources. The default data path is intentionally public and API-key-free.

## Common Commands

```bash
# Build the binary at repo root (the README assumes ./steam-cli)
go build -o steam-cli .

# Static analysis
go vet ./...

# Full test suite (race detector recommended; the HTTP layer uses goroutines)
go test -race ./...

# Run tests for a single package
go test ./cmd
go test ./internal/steam
go test ./internal/i18n

# Run a single test by name
go test ./cmd -run TestParseAppIDValid
go test ./internal/steam -run TestRetryOn429HonorsRetryAfter

# Smoke checks against live Steam endpoints
./steam-cli doctor
./steam-cli price 264710 --cc US --lang english
```

CI runs `go vet`, `go test -race`, and `go build` on every push (`.github/workflows/ci.yml`).

## Architecture

### Layering

- `main.go` → `cmd.Execute()` is the only entry point.
- `cmd/` holds Cobra commands and all terminal rendering. Per-command files (`app.go`, `price.go`, …) plus shared infrastructure split across `root.go` (Cobra startup, `runCommand`), `json.go` (envelope + source registry), `errors.go` (typed-error classification), `format.go` (rendering helpers + `parseAppID`), and `i18n_glue.go` (Cobra-side localization).
- `internal/steam/` is the Steam data layer: HTTP `Client`, `Cache`, retry policy, typed errors, JSON/XML decoding, and result types. Commands talk to Steam **only** through this package.
- `internal/ui/` wraps Charmbracelet Lip Gloss styles. All ANSI styling routes through here so `--no-color` (`ui.DisableColor()`) can disable it globally.
- `internal/i18n/` provides bilingual (en / zh-CN) UI strings, with `sync.Once`-cached auto-detection from `LC_ALL` / `LC_MESSAGES` / `LANG` and OS-level fallbacks. Tests touching env-vars must call `i18n.ResetDetect()` to clear the cache.

### Typed errors (important)

`internal/steam/errors.go` defines:

- `steam.Code` — stable enum (`CodeRateLimited`, `CodeNotFound`, `CodeInvalidInput`, `CodeNetworkTimeout`, `CodeSourceChanged`, …) used as the JSON envelope's `error.type`.
- `steam.Error{Code, Message, HintKey, Cause}` — the typed error returned from every public function in the package.
- `steam.HTTPError{Status, URL, RetryAfter}` — a non-2xx response, returned standalone for unwrapped HTTP failures and used as `Cause` inside an `*Error` for typed mapping.
- `steam.CodeOf(err)`, `steam.HintKeyOf(err)`, `steam.HTTPErrorFromAny(err)` — `errors.As` walkers used by the CLI.

`cmd/errors.go` `classifyError` is the **only** error-classification site. It uses `errors.As` against these types — never substring-matching error messages. When wrapping or producing errors in `internal/steam`, prefer the helpers (`newInvalidInput`, `newNotFound`, `newPrivacyRestricted`, `newSourceChanged`, `wrapNetwork`, `wrapHTTPStatus`) so the `Code` field is set correctly.

`cmd/format.go` `parseAppID(arg)` is the single source of truth for parsing APPID arguments. **Never** copy `strconv.Atoi(args[0])` + ad-hoc error wrapping into a new command — that's how the old classifier got brittle.

### HTTP plumbing (`internal/steam/client.go`)

- `Client.Endpoints` is injectable. Tests construct an `httptest.NewServer` and rewrite all four hosts (`Store`, `API`, `Community`, `CDN`) at once via `newTestClient`. Production uses `DefaultEndpoints()`.
- `Client.Cache` is a first-class `*Cache` (not a package-level singleton). `cmd.client()` shares a single `sharedCache` across all per-command Clients so cross-region price comparison still hits the same map.
- `Client.MinInterval` self-throttles consecutive requests from one Client. Surfaced via the global `--rate-ms` flag.
- `Client.getText` retry loop:
  - Single retry policy: `429`, `502`, `503`, `504` are retriable; max 3 attempts.
  - Picks **exactly one** delay per attempt: `Retry-After` (capped at 30 s) or exponential backoff. The earlier double-sleep bug (sleep `Retry-After` then sleep the index-based default) is gone — there's a regression test (`TestRetryOn429HonorsRetryAfter`) that fails if it returns.
  - Retry-After > 30 s aborts immediately rather than blocking the CLI.
  - Final attempt surfaces `*HTTPError` so callers (and `classifyError`) get structured status + retry-after.
- `UserAgent = "steam-cli/" + Version` is reused across `getText`, `ProbeURL`, and `doctor`; no more scattered `"steam-cli/0.1"` literals.

### Cache (`internal/steam/cache.go`)

- `*steam.Cache` is goroutine-safe; methods (`GetAppDetails`, `SetAppDetails`, `GetStoreItem`, `SetStoreItem`, `GetProfile`, `SetProfile`, `GetWishlist`, `SetWishlist`) lock internally. Callers never touch the mutex.
- `DefaultCache` is the process-wide instance used by `cmd.sharedCache`. Tests construct fresh caches via `NewCache()` to avoid bleed.
- TTLs: app details 10 min, store items 10 min, profiles 10 min, wishlists 5 min. `StoreItem` is now cached (was uncached before) so `app`, `price`, `media`, and `dlc` no longer round-trip the same store item independently.

### `runCommand` wrapper

`cmd/root.go` defines `runCommand(load, render)`. **All non-trivial commands must use this wrapper for their `RunE`.** It:

1. Re-applies normalized options (`--cc`, `--lang`, `--no-color`).
2. Calls `load(...)` to fetch and assemble data via `internal/steam`.
3. If `--json` is set, prints a unified envelope (`ok`, `command`, `schema`, `generated_at`, `meta`, `data`/`error`) — `render` is skipped.
4. Otherwise calls `render(...)` for terminal output.

Per-command source attribution comes from `commandSources` in `cmd/json.go` — when adding a command, register its sources there so `meta.sources` is populated.

### Concurrency

- `Client.AppBundle` runs the four sibling lookups (store item, reviews, current players, news) concurrently via `sync.WaitGroup`. Failures are recorded in `bundle.Warnings []string` rather than aborting the bundle. The `app` command renders warnings beneath primary output; the JSON envelope includes them in `data.warnings`.
- `Client` and `Cache` are safe for concurrent use; tests run with `-race`.

### Locale/flag normalization

`normalizeOptions` in `cmd/root.go` rewrites common aliases before they reach Steam: `UK → GB` for `--cc`; `chinese|zh-cn → schinese`, `zh-tw → tchinese`, `korean → koreana`, `pt-br → brazilian` for `--lang`. New aliases should be added there rather than at call sites.

`--ui-lang` (Steam CLI's own UI) is **separate** from `--lang` (Steam content language). Don't conflate them. `i18n.Set("auto")` is `sync.Once`-cached; tests that mutate env vars must call `i18n.ResetDetect()` first.

## Conventions for new commands

1. Add a new file under `cmd/` with a `*cobra.Command` registered in the `rootCmd.AddCommand(...)` call in `init()` and in `allCommands()` in `cmd/root.go`.
2. Use `RunE: runCommand(load, render)` — never write directly to `os.Stdout` for primary output, and never print success output from `load`.
3. For commands taking APPID, call `parseAppID(args[0])`. Do not write your own `strconv.Atoi` + error-wrapping.
4. Render via `internal/ui` helpers (`ui.Title`, `ui.Section`, `ui.KeyValue`, `ui.Table`, `ui.Join`) so `--no-color` works.
5. Add the command to `localizeCommands()` in `cmd/i18n_glue.go` and add string keys to **both** `en` and `zhCN` maps in `internal/i18n/i18n.go`. `TestKeysetParity` will fail if you forget one side.
6. Register the command's data sources in `commandSources` (and an alias mapping in `commandName` if the command has aliases).
7. From `internal/steam`, prefer producing typed errors (`newNotFound`, `newInvalidInput`, etc.) so `classifyError` in `cmd/errors.go` Just Works. Don't return bare `fmt.Errorf` — `classifyError` will type it as `unknown`.

## Limitations to keep in mind

- StoreBrowse and StoreQuery are publicly reachable but not formally documented as stable APIs; assume fields can disappear and prefer `appdetails` fallbacks where reasonable.
- Authoritative discount **start** time is not exposed by Steam's current public response — only `active_discounts[].discount_end_date` is.
- Do not attempt to bypass private profiles, friends-only data, or login-gated endpoints.
