# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`lazycurl` is a keyboard-driven terminal UI (Bubble Tea) for building, storing, and executing HTTP requests. It does not implement its own HTTP client — every request is shelled out to a `curl` subprocess (curl >= 7.70 required for `-w '%{json}'` response metadata), and the TUI parses curl's output back into structured responses.

## Commands

```sh
go build -o lazycurl ./cmd/lazycurl   # build
./lazycurl                            # run
go test ./...                         # run all tests
go test ./internal/tui/...            # run a package's tests
go test ./internal/httpfile/ -run TestParse -v   # run a single test
go fmt ./...                          # format
go mod tidy                           # sync go.sum
```

There is no CI config, Makefile, or lint config in this repo — `go build`, `go test`, and `go fmt` are the only gates. There is no `golangci-lint` config file despite it being mentioned in `docs/claude-code-remote-setup.md`; don't assume lint rules beyond `go vet`/`gofmt`.

### E2E tests (Docker required)

`internal/curlexec` and `internal/tui` each contain an `e2e_test.go` alongside their regular unit tests, in the same package (no build tags separate them):

```sh
go test ./internal/curlexec/...   # includes real-curl + testcontainers-go E2E tests
go test ./internal/tui/...        # includes teatest-driven E2E tests
```

These tests require a running Docker daemon: each package's `TestMain` uses `testcontainers-go` to build `testing/mockserver/Dockerfile` and start one shared mockserver container for the whole package (a fresh container per package, not per test case). Without Docker available, `TestMain` fails and **every test in that package** fails to run, including the pre-existing non-E2E tests in the same package — there is no build-tag isolation (see `openspec/changes/archive/` for the accepted trade-off).

`internal/curlexec`'s E2E tests drive `curlexec.NewExecutor()` (real curl subprocesses) against the mockserver container to verify argv flags (`-L`, `--max-time`, Basic/Bearer auth, `-w '%{json}'` parsing) and `@stream` chunked delivery/cancellation end-to-end. `internal/tui`'s E2E tests drive `tui.App` as a real `tea.Program` via `teatest`, sending key input to navigate Collections, select a request, and send it, then assert on the rendered terminal output.

### Mock HTTP server for manual testing

`testing/mockserver/` is a standalone Go module (its own `go.mod`, no relation to lazycurl's dependency graph) providing endpoints for exercising curl argv behavior by hand: `/echo`, `/status/{code}`, `/redirect/{n}`, `/delay/{sec}`, `/stream?chunks=&interval=`, `/auth/basic`, `/auth/bearer`. Run it locally with:

```sh
docker compose up   # builds testing/mockserver/Dockerfile, exposes it on localhost:8089
```

Point a `.http` request's URL at `http://localhost:8089/...` (e.g. with the `@stream` pragma against `/stream?chunks=5&interval=500`) to watch lazycurl's TUI handle real chunked responses, redirects, timeouts, etc.

## Architecture

### Data flow: file -> parse -> expand -> exec -> response

1. **`internal/httpfile`** — parser/serializer for the on-disk collection format: `.http` files (IntelliJ HTTP Client / VS Code REST Client compatible) with `###`-delimited request blocks, extended with `# @pragma` comment lines (`@insecure`, `@timeout <duration>`, `@no-redirect`) for curl-only behavior that has no plain-HTTP representation. `Request.Auth` is derived from/serialized back into an `Authorization` header — it is never a literal header in `Request.Headers`. Unknown pragmas are silently ignored for forward/backward compatibility with other tools.
2. **`internal/collection`** — `Store` maps collection names to `.http` files on disk (one file = one collection, one collection = many requests). All read/modify/write cycles go through `LoadRequests`/`SaveRequests`; there is no incremental/streaming edit of a `.http` file.
3. **`internal/environment`** — `Store` manages per-collection `<name>.env.json` variable files and which environment is "active" per collection (persisted in `state.json`). `ExpandRequest` substitutes `{{variable}}` placeholders against the active environment's variables; requests with any undefined variable are rejected *before* being sent (see `shell.sendCurrent`).
4. **`internal/curlexec`** — turns an already-variable-expanded `httpfile.Request` into a curl argv (`buildArgs` in `argv.go`) and runs it via the `Runner` interface (`NewExecutor` uses the real subprocess; `NewExecutorWithRunner` injects a fake for tests). Response body/headers are captured through temp files (`-D`/`-o`) rather than parsed from stdout; `-w '%{json}'` on stdout supplies status code and timing. `BuildArgv` is reused by the "yank as curl command" feature, so argv construction must stay side-effect-free and deterministic.
5. **`internal/config`** — resolves `~/.config/lazycurl/` (global, not project-local) and ensures the directory tree exists on startup.

### TUI structure: App -> Shell -> panels, request editing inline within panels

- **`cmd/lazycurl/main.go`** wires the three stores/executor together, runs a startup `curl --version` check (hard-fails if `curl` is missing, warns if too old), and starts the Bubble Tea program.
- **`internal/tui.App`** (`app.go`) is the top-level `tea.Model`. It is a thin wrapper: it owns no editing state and just forwards Bubble Tea messages (window resize, quit) to `shell.Shell`. Request editing happens entirely inside `Shell`, not as a separate top-level App mode.
- **`internal/tui/shell.Shell`** (`model.go`/`update.go`/`view.go`) is the four-panel browser: Collections, Requests, Response, History, navigated with lazygit-style keys (`tab`/`1`-`4` to switch panel, `j`/`k` to move, `n`/`c`/`d`/`x` to create/duplicate/delete, `E` for environment select). Modal overlays (help, env-select, new-collection, confirm-delete) are handled via an `overlay` enum inside `Shell`, not as separate `tea.Model`s. Sending a request (`enter` on the Requests panel's list, or `ctrl+r` while editing) runs asynchronously via a `tea.Cmd` that returns `sendResultMsg`; `ctrl+c` while sending cancels via a stored `context.CancelFunc` instead of quitting.
- **Inline request editing**: `Shell` owns a single embedded `form.Editor` instance and renders it directly inside the Requests panel (Collections mode) or the Editor panel (Adhoc mode) — there is no separate full-screen editing mode. The Requests panel has two internal sub-focus zones: `zoneList` (the browsable request list, `j`/`k`/`enter`/`n`/`c`/`d`/`x`/`E`) and `zoneForm` (the embedded editor, entered via `tab` from the list). Adhoc's Editor panel has no list zone — it is always in `zoneForm`, since there is only ever one scratch request. `Shell.inFormZone()` reports whether the currently focused panel/zone is the embedded form; while true, `Shell.handleKey` routes almost all keys directly to the form (bypassing global shortcuts like `q`/`[`/`]`/`1`-`4` that would otherwise steal characters meant for text fields) — only `ctrl+c` stays global. Within the form zone, `ctrl+s` saves (writes `requests` to the collection's `.http` file, or opens the save-to-collection overlay in Adhoc) and `ctrl+r` sends; `[`/`]` switch the form's Params/Headers/Auth/Body sub-tab. Every keystroke inside the form is synced back into `Shell`'s in-memory `requests[idx]`/`adhocRequest` via `syncEditorToTarget()`, so list summaries update live; nothing is written to disk until `ctrl+s`.
- **`internal/tui/form.Editor`** is the request-editing form itself (Name/Method/URL, then Params/Headers/Auth/Body tabs), converted to/from `httpfile.Request` via `ToRequest()`/`FromRequest()` — it has no direct reference to the collection store and no awareness of being embedded in a panel. Body editing has an external-`$EDITOR` escape hatch (`ctrl-e`) that shells out and reloads the file on process exit.

## Spec-driven workflow (openspec)

This project develops features through `@fission-ai/openspec` (installed separately via `npm install -g openspec`), not ad hoc changes. Specs live in `openspec/specs/<capability>/spec.md`; in-flight and archived work lives in `openspec/changes/`. The slash commands `/opsx:explore`, `/opsx:propose`, `/opsx:apply`, `/opsx:archive` (backed by skills in `.claude/skills/`) drive this: explore -> propose (generates `proposal.md`/`design.md`/`tasks.md`) -> apply (implements `tasks.md` in order) -> archive (moves the change into `openspec/changes/archive/`). All openspec artifacts (proposals, tasks, specs) are written in Japanese per `openspec/config.yaml`; code, identifiers, file paths, and technical terms (API, REST, HTTP, TUI, etc.) stay in English. There is no commit message convention (free-form).

## Storage layout (runtime, not repo)

Collections are stored globally, not per-project:

```
~/.config/lazycurl/
├── state.json                       # active environment per collection
└── collections/
    ├── <collection-name>.http       # one file per collection, ### per request
    └── env/<collection-name>/<env-name>.env.json
```
