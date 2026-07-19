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
3. **`internal/environment`** — `Store` manages per-collection `<name>.env.json` variable files and which environment is "active" per collection (persisted in `state.json`). `ExpandRequest` substitutes `{{variable}}` placeholders against the active environment's variables; requests with any undefined variable are rejected *before* being sent (see `shell.sendLoadedCurrent`).
4. **`internal/curlexec`** — turns an already-variable-expanded `httpfile.Request` into a curl argv (`buildArgs` in `argv.go`) and runs it via the `Runner` interface (`NewExecutor` uses the real subprocess; `NewExecutorWithRunner` injects a fake for tests). Response body/headers are captured through temp files (`-D`/`-o`) rather than parsed from stdout; `-w '%{json}'` on stdout supplies status code and timing. `BuildArgv` is reused by the "yank as curl command" feature, so argv construction must stay side-effect-free and deterministic.
5. **`internal/config`** — resolves `~/.config/lazycurl/` (global, not project-local) and ensures the directory tree exists on startup.

### TUI structure: App -> Shell -> a fixed 2x2 panel grid, no modes

- **`cmd/lazycurl/main.go`** wires the three stores/executor together, runs a startup `curl --version` check (hard-fails if `curl` is missing, warns if too old), and starts the Bubble Tea program.
- **`internal/tui.App`** (`app.go`) is the top-level `tea.Model`. It is a thin wrapper: it owns no editing state and just forwards Bubble Tea messages (window resize, quit) to `shell.Shell`.
- **`internal/tui/shell.Shell`** (`model.go`/`update.go`/`view.go`) renders a fixed 2x2 grid of four always-visible panels — `PanelRequest` (top-left), `PanelResponse` (top-right), `PanelCollections` (bottom-left), `PanelHistory` (bottom-right) — navigated with lazygit-style keys: `tab`/`shift+tab` to cycle focus, `0`-`3` to jump straight to a panel. There used to be separate Adhoc/Collections *modes* with different panel layouts (`[`/`]` to switch between them); that concept is gone — the panel grid, its labels, and their positions never change. Modal overlays (help, env-select, new-collection, confirm-delete, save-to-collection, request-name prompt) are handled via an `overlay` enum inside `Shell`, not as separate `tea.Model`s.
- **The Request panel *is* the form**: `Shell` owns a single embedded `form.Editor` and renders it directly inside `PanelRequest` — there is no separate list/editor split or sub-focus zone for that panel anymore. It always reflects one of two things: the collection-less **scratch request** (`s.scratchRequest`, `s.usingScratch == true`), or a request loaded from a collection (`s.requests[s.requestIdx]` bound to `s.loadedCollection`, `s.usingScratch == false`). Every keystroke inside the form is synced back into whichever target it reflects via `syncEditorToTarget()`, so nothing is written to disk until `ctrl+s`. Within `PanelRequest`, `ctrl+s` (`saveRequestPanel`) and `ctrl+r` (`sendRequestPanel`) work in both the form's normal and insert states; while the form is in insert state, all other keys are forwarded to it (bypassing global shortcuts like `q`/`0`-`3`/`tab` that would otherwise steal characters meant for text fields) — only `ctrl+c` stays global. In the form's normal state, `[`/`]` switch its Params/Headers/Auth/Body sub-tab.
- **The Collections panel is an accordion**, not a plain list: the collection under the cursor (`collectionIdx`) is always expanded inline to show its requests (`previewRequests`); other collections collapse to a single name line. `j`/`k` walk a flattened cursor down through the expanded collection's header and request rows, then on to the next collection's header (which becomes the newly-expanded one). `enter` on a request row calls `loadRequestIntoEditor`, which loads that request into `PanelRequest` and moves focus there; it does not send anything. `n` creates a new request in the currently-expanded collection and loads it straight into `PanelRequest`; `N` opens the new-collection name-prompt overlay; `c` duplicates the request under the cursor in place; `d`/`x` opens a delete-confirmation overlay; `E` opens the environment-select overlay (scoped to the collection under the cursor).
- Saving (`ctrl+s` in `PanelRequest`) behaves differently depending on the target: a loaded collection request writes the whole collection's request list back to its `.http` file (`saveLoadedRequests`); the scratch request instead opens the save-to-collection overlay (`overlaySaveTo`, pick an existing collection or create one). Either way, if the target request has no name yet, `ctrl+s` first opens `overlayRequestName` and defers the actual save until a name is entered. Sending (`ctrl+r`) similarly branches on `s.usingScratch`: `sendLoadedCurrent` expands `{{variable}}`s against the collection's active environment and rejects the send if any are undefined, while `sendScratchCurrent` sends the request as-is, with no expansion. Both run asynchronously via a `tea.Cmd` that returns `sendResultMsg` (or, for a `@stream` request, a `streamChunkMsg`/`streamDoneMsg` sequence); `ctrl+c` while sending cancels via a stored `context.CancelFunc` instead of quitting.
- The History panel (`j`/`k` to move the preview cursor, `enter` to point `PanelResponse` at that entry) and Response panel (display-only) are otherwise unchanged in spirit from the panel-mode era.
- **`internal/tui/form.Editor`** is the request-editing form itself (Name/Method/URL, then Params/Headers/Auth/Body tabs), converted to/from `httpfile.Request` via `ToRequest()`/`FromRequest()` — it has no direct reference to the collection store and no awareness of being embedded in a panel. Body editing has an external-`$EDITOR` escape hatch (`ctrl-e`) that shells out and reloads the file on process exit.

## Spec-driven workflow (openspec)

This project develops features through `@fission-ai/openspec` (installed separately via `npm install -g @fission-ai/openspec@latest`), not ad hoc changes. Specs live in `openspec/specs/<capability>/spec.md`; in-flight and archived work lives in `openspec/changes/`. The slash commands `/opsx:explore`, `/opsx:propose`, `/opsx:apply`, `/opsx:archive` (backed by skills in `.claude/skills/`) drive this: explore -> propose (generates `proposal.md`/`design.md`/`tasks.md`) -> apply (implements `tasks.md` in order) -> archive (moves the change into `openspec/changes/archive/`). All openspec artifacts (proposals, tasks, specs) are written in Japanese per `openspec/config.yaml`; code, identifiers, file paths, and technical terms (API, REST, HTTP, TUI, etc.) stay in English. There is no commit message convention (free-form).

## Storage layout (runtime, not repo)

Collections are stored globally, not per-project:

```
~/.config/lazycurl/
├── state.json                       # active environment per collection
└── collections/
    ├── <collection-name>.http       # one file per collection, ### per request
    └── env/<collection-name>/<env-name>.env.json
```
