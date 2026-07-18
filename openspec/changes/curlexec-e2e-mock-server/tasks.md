## 1. モックサーバーモジュールの雛形

- [ ] 1.1 `testing/mockserver/`ディレクトリを作成し、独自の`go.mod`(root moduleとは別)を初期化する
- [ ] 1.2 HTTPサーバーのエントリポイント(`main.go`)を作成し、環境変数またはフラグでlisten portを指定できるようにする

## 2. モックサーバーのエンドポイント実装

- [ ] 2.1 `/echo`: リクエストのメソッド・ヘッダー・bodyをJSONで返すハンドラを実装する
- [ ] 2.2 `/status/{code}`: 指定したHTTPステータスコードで応答するハンドラを実装する
- [ ] 2.3 `/redirect/{n}`: `n`回のリダイレクトを経て200を返すハンドラを実装する
- [ ] 2.4 `/delay/{sec}`: 指定秒数待機してから応答するハンドラを実装する
- [ ] 2.5 `/stream`: レスポンスbodyを`http.Flusher`で複数チャンクに分け、時間差で逐次送出するハンドラを実装する
- [ ] 2.6 `/auth/basic`, `/auth/bearer`: `Authorization`ヘッダーを検証し、正誤に応じて200/401を返すハンドラを実装する
- [ ] 2.7 各エンドポイントの単体テスト(`testing/mockserver`モジュール内、`net/http/httptest`使用)を追加する

## 3. モックサーバーのDocker化

- [ ] 3.1 `testing/mockserver/Dockerfile`(マルチステージビルド)を作成する
- [ ] 3.2 ローカルで`docker build`してイメージが起動し、`/status/200`等に応答することを確認する

## 4. docker composeによる手動起動

- [ ] 4.1 `docker-compose.yml`を追加し、`testing/mockserver/Dockerfile`をbuildして固定ホストポートで公開する設定を書く
- [ ] 4.2 `docker compose up`で起動したモックサーバーに対し、lazycurlのTUI(または`curl`直接)からリクエストが通ることを手動確認する

## 5. E2Eテスト基盤(testcontainers-go)

- [ ] 5.1 root moduleに`testcontainers-go`を依存として追加する(`go get`)
- [ ] 5.2 `internal/curlexec`にE2Eテスト用ファイルを追加し、`TestMain`で`testcontainers-go`の`FromDockerfile`(context: `testing/mockserver/`)を使ってコンテナを1回だけ起動する
- [ ] 5.3 起動したコンテナのマップ済みベースURLをパッケージ変数として保持し、テスト終了時(`TestMain`内)にコンテナをterminateする

## 6. E2Eテストケースの実装

- [ ] 6.1 `curlexec.NewExecutor()`(実curl)経由で`/redirect/{n}`を叩き、`-L`によるリダイレクト追従を検証するテストを書く
- [ ] 6.2 `@no-redirect`プラグマ付きリクエストで`/redirect/{n}`を叩き、リダイレクトが追従されないことを検証するテストを書く
- [ ] 6.3 `@timeout`プラグマ付きリクエストで`/delay/{長い秒数}`を叩き、指定時間内にタイムアウトエラーになることを検証するテストを書く
- [ ] 6.4 Basic/Bearer認証を設定したリクエストで`/auth/basic`・`/auth/bearer`を叩き、`Authorization`ヘッダーが正しく導出・送信されることを検証するテストを書く
- [ ] 6.5 `/status/{code}`・`/echo`を用いて、ステータスコード・ヘッダー・bodyの取得(`-D`/`-o`/`-w '%{json}'`)が正しくパースされることを検証するテストを書く

## 7. ドキュメント

- [ ] 7.1 `CLAUDE.md`の「Commands」セクションに、E2Eテストの実行方法(`go test ./internal/curlexec/...`、Dockerが必要である旨)を追記する
- [ ] 7.2 `CLAUDE.md`または`testing/mockserver/`直下に、`docker compose up`での手動起動方法を追記する
