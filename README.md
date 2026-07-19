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

### Adhoc / Collections modes

lazycurl has two modes, `Adhoc` and `Collections`, which you can switch between at any time with the `[` / `]` keys. The active mode is highlighted in the tab at the top of the screen. `Adhoc` is the default mode on startup.

- **Adhoc**: Build and send a request on the fly, without creating or selecting a collection. It consists of three panes: the edit form, Response, and History. A request you build only exists in memory until you save it (`s` key), and `{{variable}}` expansion is not applied.
- **Collections**: The traditional four-pane layout of Collections/Requests/Response/History. Requests are managed per collection, with environment variable expansion and switching.

A request built in Adhoc mode can be saved at any time with the `s` key. You can choose an existing collection or create a new one; after saving, lazycurl automatically switches to `Collections` mode with the destination collection and request selected. The execution history (History) is shared between both modes.

### Keybindings (lazygit-style)

| Key | Action |
| --- | --- |
| `[` / `]` | Switch between Adhoc/Collections mode |
| `tab` / `shift+tab` | Move between panels |
| `1`-`4` | Jump to a panel (1-3: Editor/Response/History in Adhoc mode, 1-4: Collections/Requests/Response/History in Collections mode) |
| `j` / `k` | Move up/down |
| `enter` | Send/confirm the selected item (sends the in-progress request in Adhoc mode) |
| `n` | Create new (collection/request) |
| `e` | Edit the request (edits the in-progress request in Adhoc mode) |
| `s` | Save the Adhoc mode request to a collection |
| `c` | Duplicate a request |
| `d` / `x` | Delete a request |
| `E` | Switch environment |
| `?` | Show help |
| `q` / `ctrl-c` | Quit (cancels the in-flight request if one is sending) |

Inside the request edit form, `ctrl-s` saves and `ctrl-q` discards changes and returns. In the Body tab, `ctrl-e` launches `$EDITOR` and reloads the file's contents once the external editor process exits.

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
