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

## Architecture

### Data flow: file -> parse -> expand -> exec -> response

1. **`internal/httpfile`** — parser/serializer for the on-disk collection format: `.http` files (IntelliJ HTTP Client / VS Code REST Client compatible) with `###`-delimited request blocks, extended with `# @pragma` comment lines (`@insecure`, `@timeout <duration>`, `@no-redirect`) for curl-only behavior that has no plain-HTTP representation. `Request.Auth` is derived from/serialized back into an `Authorization` header — it is never a literal header in `Request.Headers`. Unknown pragmas are silently ignored for forward/backward compatibility with other tools.
2. **`internal/collection`** — `Store` maps collection names to `.http` files on disk (one file = one collection, one collection = many requests). All read/modify/write cycles go through `LoadRequests`/`SaveRequests`; there is no incremental/streaming edit of a `.http` file.
3. **`internal/environment`** — `Store` manages per-collection `<name>.env.json` variable files and which environment is "active" per collection (persisted in `state.json`). `ExpandRequest` substitutes `{{variable}}` placeholders against the active environment's variables; requests with any undefined variable are rejected *before* being sent (see `shell.sendCurrent`).
4. **`internal/curlexec`** — turns an already-variable-expanded `httpfile.Request` into a curl argv (`buildArgs` in `argv.go`) and runs it via the `Runner` interface (`NewExecutor` uses the real subprocess; `NewExecutorWithRunner` injects a fake for tests). Response body/headers are captured through temp files (`-D`/`-o`) rather than parsed from stdout; `-w '%{json}'` on stdout supplies status code and timing. `BuildArgv` is reused by the "yank as curl command" feature, so argv construction must stay side-effect-free and deterministic.
5. **`internal/config`** — resolves `~/.config/lazycurl/` (global, not project-local) and ensures the directory tree exists on startup.

### TUI structure: App -> Shell -> panels, Editor as a modal overlay

- **`cmd/lazycurl/main.go`** wires the three stores/executor together, runs a startup `curl --version` check (hard-fails if `curl` is missing, warns if too old), and starts the Bubble Tea program.
- **`internal/tui.App`** (`app.go`) is the top-level `tea.Model`. It owns two mutually-exclusive modes: `modeShell` (browsing) and `modeEditor` (editing a single request). It does not contain panel logic itself — it just switches between `shell.Shell` and `form.Editor`, and persists `form.Editor` saves back through `collection.Store` on `ctrl+s`.
- **`internal/tui/shell.Shell`** (`model.go`/`update.go`/`view.go`) is the four-panel browser: Collections, Requests, Response, History, navigated with lazygit-style keys (`tab`/`1`-`4` to switch panel, `j`/`k` to move, `n`/`e`/`c`/`d`/`x` to create/edit/duplicate/delete, `E` for environment select). Modal overlays (help, env-select, new-collection, confirm-delete) are handled via an `overlay` enum inside `Shell`, not as separate `tea.Model`s. Sending a request (`enter` on the Requests panel) runs asynchronously via a `tea.Cmd` that returns `sendResultMsg`; `ctrl+c` while sending cancels via a stored `context.CancelFunc` instead of quitting.
- **`internal/tui/form.Editor`** is the request-editing form (Name/Method/URL, then Params/Headers/Auth/Body tabs). It communicates with `Shell`/`App` only through `OpenEditorMsg` (Shell -> App, "start editing") and `ToRequest()`/`FromRequest()` conversions — it has no direct reference to the collection store. Body editing has an external-`$EDITOR` escape hatch (`ctrl-e`) that shells out and reloads the file on process exit.
- Cross-cutting: `Shell.ReloadCurrentCollection()` is called by `App` after every editor save, since the `.http` file was rewritten out from under `Shell`'s in-memory `requests` slice.

## Spec-driven workflow (openspec)

This project develops features through `openspec` (installed separately via `npm install -g openspec`), not ad hoc changes. Specs live in `openspec/specs/<capability>/spec.md`; in-flight and archived work lives in `openspec/changes/`. The slash commands `/opsx:explore`, `/opsx:propose`, `/opsx:apply`, `/opsx:archive` (backed by skills in `.claude/skills/`) drive this: explore -> propose (generates `proposal.md`/`design.md`/`tasks.md`) -> apply (implements `tasks.md` in order) -> archive (moves the change into `openspec/changes/archive/`). All openspec artifacts (proposals, tasks, specs) are written in Japanese per `openspec/config.yaml`; code, identifiers, file paths, and technical terms (API, REST, HTTP, TUI, etc.) stay in English. There is no commit message convention (free-form).

## Storage layout (runtime, not repo)

Collections are stored globally, not per-project:

```
~/.config/lazycurl/
├── state.json                       # active environment per collection
└── collections/
    ├── <collection-name>.http       # one file per collection, ### per request
    └── env/<collection-name>/<env-name>.env.json
```
