# lazycurl

```
 _                                 _ 
| | __ _ _____   _  ___ _   _ _ __| |
| |/ _` |_  / | | |/ __| | | | '__| |
| | (_| |/ /| |_| | (__| |_| | |  | |
|_|\__,_/___|\__, |\___|\__,_|_|  |_|
             |___/
```

A keyboard-driven, terminal-only HTTP client inspired by lazygit/lazydocker/lazysql.
Instead of implementing its own HTTP client, lazycurl shells out to `curl` and lets you inspect the results in a TUI.

## Installation

```sh
go install github.com/asunaro276/lazycurl/cmd/lazycurl@latest
```

Or clone this repository and build it yourself.

```sh
git clone https://github.com/asunaro276/lazycurl.git
cd lazycurl
go build -o lazycurl ./cmd/lazycurl
```

### Dependency: `curl`

lazycurl uses the `curl` binary to execute requests. `curl` must already be installed (version 7.70 or later is recommended, since `-w '%{json}'` is required to fetch response metadata).

```sh
curl --version
```

If `curl` is not found, lazycurl shows an error and exits on startup. If the version is below 7.70, it shows a warning but still starts.

## Usage

```sh
lazycurl
```

### The shell: four always-visible panels

lazycurl has no modes. On startup you always see the same fixed 2x2 grid of panels: **Request** (top-left), **Response** (top-right), **Collections** (bottom-left), **History** (bottom-right). There is no separate Adhoc/Collections mode and no layout to switch between.

- **Request**: the request-editing form (Name/Method/URL, then Params/Headers/Auth/Body tabs). It always shows either a scratch request that doesn't belong to any collection yet, or a request loaded from a collection via the Collections panel. Edits are kept in memory only; nothing is written to disk until you save (`ctrl+s`).
- **Response**: shows the result of whichever request was last sent, or a selected History entry. Display-only.
- **Collections**: an accordion list of your collections. The collection under the cursor is expanded inline to show its requests; pressing `enter` on a request loads it into the Request panel.
- **History**: every request/response pair you've sent, oldest first. Selecting an entry (`enter`) shows it in the Response panel.

A scratch request built in the Request panel can be saved at any time with `ctrl+s`. If it has no name yet, you're prompted for one first; then you choose an existing collection or create a new one to save it into. The request stays loaded in the Request panel afterward, now bound to that collection.

### Keybindings (lazygit-style)

| Key | Action |
| --- | --- |
| `tab` / `shift+tab` | Move between panels |
| `0`-`3` | Jump directly to a panel (Request/Response/Collections/History) |
| `j` / `k` | Move up/down within the focused panel |
| `?` | Show help |
| `q` / `ctrl-c` | Quit (cancels the in-flight request if one is sending) |

**Collections panel:**

| Key | Action |
| --- | --- |
| `enter` | Load the request under the cursor into the Request panel |
| `n` | Create a new request in the expanded collection |
| `N` | Create a new collection |
| `c` | Duplicate the request under the cursor |
| `d` / `x` | Delete the request under the cursor (with confirmation) |
| `E` | Switch the active environment for this collection |

**Request panel:**

| Key | Action |
| --- | --- |
| `enter` | Enter insert mode on the focused field |
| `esc` | Leave insert mode |
| `h` / `l` | Change the HTTP method (normal state) |
| `[` / `]` | Switch the Params/Headers/Auth/Body tab (normal state) |
| `ctrl+r` | Send the request |
| `ctrl+s` | Save the request (prompts for a name first if unnamed) |

In the Body tab, `ctrl-e` launches `$EDITOR` and reloads the file's contents once the external editor process exits.

**History panel:**

| Key | Action |
| --- | --- |
| `enter` | Show the selected entry in the Response panel |

## Collection storage format

Requests are stored globally under `~/.config/lazycurl/` (not project-local).

```
~/.config/lazycurl/
├── state.json                       # state such as the active environment
└── collections/
    ├── <collection-name>.http       # one file per collection, multiple requests separated by ###
    └── env/
        └── <collection-name>/
            ├── dev.env.json
            ├── staging.env.json
            └── prod.env.json
```

Collection files use the [IntelliJ HTTP Client](https://www.jetbrains.com/help/idea/http-client-in-product-code-editor.html) / [VS Code REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)-compatible `.http` format, extended with lightweight pragma comments for curl-specific options.

```http
### Get user (self-signed dev server)
# @insecure
# @timeout 5s
GET {{host}}/users/{{id}}
Authorization: Bearer {{token}}
```

Supported pragmas:

| Pragma | Translates to |
| --- | --- |
| `# @insecure` | `curl -k` (skip TLS verification) |
| `# @timeout <duration>` | `curl --max-time <seconds>` |
| `# @no-redirect` | Do not follow redirects (`-L` is added by default if this is omitted) |
| `# @stream` | `curl -N` (disable buffering). The body is read incrementally from stdout instead of via a temp file, and is appended to the Response pane as it arrives |

Unknown pragma lines are ignored, so files still open correctly in other tools.

`@stream` is an opt-in pragma for watching a long-lived response, such as Server-Sent Events (SSE), as it comes in. When set, the body is displayed incrementally in the Response pane without waiting for the request to finish (or to be cancelled with ctrl-c). Response time is measured as Go-side elapsed wall time rather than curl's `-w` timing. If the request is cancelled, whatever body was received up to that point is still recorded as the final response in history.

## Environments and variable expansion

`{{variable}}` placeholders are expanded using values from the active environment (`env/<collection>/<name>.env.json`) before being passed to `curl`. If a request references an undefined variable, an error is shown before sending and the request is not sent.

## License

lazycurl is licensed under the [MIT License](LICENSE).
