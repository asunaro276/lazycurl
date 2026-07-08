# lazycurl

lazygit/lazydocker/lazysql にインスパイアされた、ターミナルで完結するHTTPクライアント。
自前のHTTPクライアント実装ではなく `curl` をサブプロセスとして呼び出し、その結果をTUIで確認できます。

## インストール

```sh
go install github.com/asunaro276/lazycurl/cmd/lazycurl@latest
```

または、このリポジトリをクローンしてビルドします。

```sh
git clone https://github.com/asunaro276/lazycurl.git
cd lazycurl
go build -o lazycurl ./cmd/lazycurl
```

### 依存: `curl`

lazycurlはリクエストの実行に `curl` バイナリを使用します。事前に `curl` がインストールされている必要があります(バージョン7.70以上を推奨。`-w '%{json}'` によるレスポンスメタデータ取得に必要です)。

```sh
curl --version
```

`curl` が見つからない場合、lazycurlは起動時にエラーを表示して終了します。7.70未満の場合は警告を表示した上で起動します。

## 使い方

```sh
lazycurl
```

### キーバインド(lazygit互換)

| キー | 動作 |
| --- | --- |
| `tab` / `shift+tab` | パネル間移動 |
| `1`-`4` | パネルへジャンプ(Collections/Requests/Response/History) |
| `j` / `k` | 上下移動 |
| `enter` | 選択項目を送信・確定 |
| `n` | 新規作成(コレクション/リクエスト) |
| `e` | リクエストの編集 |
| `c` | リクエストの複製 |
| `d` / `x` | リクエストの削除 |
| `E` | environmentの切り替え |
| `?` | ヘルプ表示 |
| `q` / `ctrl-c` | 終了(送信中は中断) |

リクエスト編集フォーム内では `ctrl-s` で保存、`ctrl-q` で破棄して戻ります。Bodyタブでは `ctrl-e` で `$EDITOR` を起動し、外部エディタでの編集内容を再読み込みします。

## コレクションの保存形式

リクエストは `~/.config/lazycurl/` 配下にグローバルに保存されます(プロジェクトローカルではありません)。

```
~/.config/lazycurl/
├── state.json                       # アクティブenvironmentなどの状態
└── collections/
    ├── <collection-name>.http       # 1コレクション = 1ファイル、###区切りで複数リクエスト
    └── env/
        └── <collection-name>/
            ├── dev.env.json
            ├── staging.env.json
            └── prod.env.json
```

コレクションファイルは [IntelliJ HTTP Client](https://www.jetbrains.com/help/idea/http-client-in-product-code-editor.html) / [VS Code REST Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) 互換の `.http` 形式に、curl固有オプション用の軽量なプラグマコメントを加えた形式です。

```http
### Get user (self-signed dev server)
# @insecure
# @timeout 5s
GET {{host}}/users/{{id}}
Authorization: Bearer {{token}}
```

対応プラグマ:

| プラグマ | 変換先 |
| --- | --- |
| `# @insecure` | `curl -k`(TLS検証をスキップ) |
| `# @timeout <duration>` | `curl --max-time <秒数>` |
| `# @no-redirect` | リダイレクトを追従しない(付与しない場合はデフォルトで `-L` が付く) |

未知のプラグマ行は無視されるため、他ツールで開いても壊れません。

## environmentと変数展開

`{{variable}}` はアクティブなenvironment(`env/<collection>/<name>.env.json`)の値で展開されてから `curl` に渡されます。未定義の変数を参照している場合、送信前にエラーとして表示され、送信は行われません。
