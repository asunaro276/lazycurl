## 1. モックサーバーモジュールの雛形

- [x] 1.1 `testing/mockserver/`ディレクトリを作成し、独自の`go.mod`(root moduleとは別)を初期化する
- [x] 1.2 HTTPサーバーのエントリポイント(`main.go`)を作成し、環境変数またはフラグでlisten portを指定できるようにする

## 2. モックサーバーのエンドポイント実装

- [x] 2.1 `/echo`: リクエストのメソッド・ヘッダー・bodyをJSONで返すハンドラを実装する
- [x] 2.2 `/status/{code}`: 指定したHTTPステータスコードで応答するハンドラを実装する
- [x] 2.3 `/redirect/{n}`: `n`回のリダイレクトを経て200を返すハンドラを実装する
- [x] 2.4 `/delay/{sec}`: 指定秒数待機してから応答するハンドラを実装する
- [x] 2.5 `/stream`: レスポンスbodyを`http.Flusher`で複数チャンクに分け、時間差で逐次送出するハンドラを実装する。分割数(`chunks`)・間隔(`interval`)をクエリパラメータで指定可能にする
- [x] 2.6 `/auth/basic`, `/auth/bearer`: `Authorization`ヘッダーを検証し、正誤に応じて200/401を返すハンドラを実装する
- [x] 2.7 各エンドポイントの単体テスト(`testing/mockserver`モジュール内、`net/http/httptest`使用)を追加する

## 3. モックサーバーのDocker化

- [x] 3.1 `testing/mockserver/Dockerfile`(マルチステージビルド)を作成する
- [ ] 3.2 ローカルで`docker build`してイメージが起動し、`/status/200`等に応答することを確認する

## 4. docker composeによる手動起動

- [x] 4.1 `docker-compose.yml`を追加し、`testing/mockserver/Dockerfile`をbuildして固定ホストポートで公開する設定を書く
- [ ] 4.2 `docker compose up`で起動したモックサーバーに対し、lazycurlのTUI(または`curl`直接)からリクエストが通ることを手動確認する

## 5. E2Eテスト基盤(testcontainers-go)

- [x] 5.1 root moduleに`testcontainers-go`を依存として追加する(`go get`)
- [x] 5.2 `internal/curlexec`にE2Eテスト用ファイルを追加し、`TestMain`で`testcontainers-go`の`FromDockerfile`(context: `testing/mockserver/`)を使ってコンテナを1回だけ起動する
- [x] 5.3 起動したコンテナのマップ済みベースURLをパッケージ変数として保持し、テスト終了時(`TestMain`内)にコンテナをterminateする

## 6. E2Eテストケースの実装

- [x] 6.1 `curlexec.NewExecutor()`(実curl)経由で`/redirect/{n}`を叩き、`-L`によるリダイレクト追従を検証するテストを書く
- [x] 6.2 `@no-redirect`プラグマ付きリクエストで`/redirect/{n}`を叩き、リダイレクトが追従されないことを検証するテストを書く
- [x] 6.3 `@timeout`プラグマ付きリクエストで`/delay/{長い秒数}`を叩き、指定時間内にタイムアウトエラーになることを検証するテストを書く
- [x] 6.4 Basic/Bearer認証を設定したリクエストで`/auth/basic`・`/auth/bearer`を叩き、`Authorization`ヘッダーが正しく導出・送信されることを検証するテストを書く
- [x] 6.5 `/status/{code}`・`/echo`を用いて、ステータスコード・ヘッダー・bodyの取得(`-D`/`-o`/`-w '%{json}'`)が正しくパースされることを検証するテストを書く
- [x] 6.6 `Pragmas.Stream`を設定したリクエストで`Executor.ExecuteStreaming`を`/stream?chunks=3&interval=...`に対して呼び、`Done`到達前に複数回`Chunk`付き`StreamEvent`を受信すること、および自然終了時に全チャンク結合bodyが`StreamDone.Response.Body`と一致することを検証するテストを書く
- [x] 6.7 `ExecuteStreaming`に渡す`context`を送信途中でキャンセルし、`StreamDone.Response.Body`がキャンセル時点までの部分bodyになり`StreamDone.Err`が`nil`であることを検証するテストを書く

## 7. TUI E2Eテスト(teatest)

- [x] 7.1 root moduleに`teatest`(`github.com/charmbracelet/x/exp/teatest`)を依存として追加する(`go get`)
- [x] 7.2 `internal/tui`にE2Eテスト用ファイルを追加し、`TestMain`で`testing/mockserver`コンテナを1回だけ起動する(`internal/curlexec`と同様のsingletonパターン、コンテナはパッケージ単位で独立)
- [x] 7.3 `teatest.NewTestModel`で`tui.App`(`curlexec.NewExecutor()`による実curl実行)を起動し、モックサーバーを指すCollection/Requestを用意する
- [x] 7.4 Collectionsパネルへの移動・Request選択・送信に相当するキー入力を送り、モックサーバーが実際にHTTPリクエストを受信することを検証するテストを書く
- [x] 7.5 送信完了後、`teatest.WaitFor`等でResponseパネルの描画内容(ステータスコード等)をアサートするテストを書く
- [x] 7.6 `@stream`プラグマ付きリクエストで`/stream`(複数チャンク・間隔あり)を送信し、送信完了前の時点で`teatest.WaitFor`により部分的なbodyがResponseパネルに描画されていることを検証するテストを書く
- [x] 7.7 `@stream`送信中に`ctrl-c`相当のキー入力を送り、打ち切り時点までのbodyを持つレスポンスがHistoryパネルに確定表示されることを検証するテストを書く

## 8. ドキュメント

- [x] 8.1 `CLAUDE.md`の「Commands」セクションに、E2Eテストの実行方法(`go test ./internal/curlexec/...`、`go test ./internal/tui/...`、Dockerが必要である旨)を追記する
- [x] 8.2 `CLAUDE.md`または`testing/mockserver/`直下に、`docker compose up`での手動起動方法を追記する
