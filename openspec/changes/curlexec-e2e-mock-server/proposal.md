## Why

`internal/curlexec`の既存テストは`Runner`インターフェースを`fakeRunner`に差し替えて検証しており、`buildArgs`が組み立てたargvが実際のcurlバイナリ・実際のHTTPサーバーに対して正しく動作するかは一度も検証されていない。加えて、進行中の`stream-response-body`変更(task 6.1)はSSE的な逐次配信を目視確認するための簡易テストサーバーを前提にしているが、これも未整備。自動テストと手動確認の両方で使える、制御可能なモックHTTPサーバーが必要。

## What Changes

- 新規の独立Goモジュール(`testing/mockserver/`、専用の`go.mod`を持つ)としてモックHTTPサーバーを実装する。lazycurl本体のcurl argv(`-X`/`-H`/`-k`/`--max-time`/`-L`/`--data-binary`/`-w '%{json}'`)を実地検証できるエンドポイント群(echo、status code指定、redirect、delay、chunked/SSE的な逐次配信、Basic/Bearer認証エコー)を提供する
- モックサーバーを単一の`Dockerfile`としてビルド可能にする
- `internal/curlexec`にE2Eテストを追加する。`testcontainers-go`で上記Dockerfileからイメージをビルド・起動し、実際のcurlサブプロセスを経由してモックサーバーに対してリクエストを送り、`Executor.Execute`の結果を検証する。モックサーバーはステートレスなため、`TestMain`でコンテナを1つだけ起動しテスト間で共有する(テストごとの起動コストを避ける)
- 同じDockerfileを参照する`docker-compose.yml`を追加し、開発者が手元で常駐起動してlazycurlのTUIから実際に叩けるようにする(`@stream`プラグマの逐次表示・ctrl-c打ち切り・自然終了の目視確認を主目的とする)
- ビルドタグ等によるE2Eテストの隔離は今回は行わない(`go test ./...`にそのまま含める)。CI整備は別途後続で行う想定

## Capabilities

### New Capabilities
- `e2e-mock-server`: curl argvの実地検証・手動確認用に、設定可能なエンドポイント群を提供する独立モジュールのモックHTTPサーバー。Docker化されており、テストと手動確認の双方から同一イメージを利用する
- `curlexec-e2e-testing`: `testcontainers-go`でモックサーバーコンテナを起動し、実curlサブプロセス経由で`internal/curlexec`の挙動を検証する自動E2Eテストスイート

### Modified Capabilities
(なし。既存の`curl-execution`等の要件は変更しない。テスト・検証手段の追加のみ)

## Impact

- 新規ディレクトリ`testing/mockserver/`(独自`go.mod`、HTTPハンドラ実装、`Dockerfile`)
- 新規`docker-compose.yml`(リポジトリルートまたは`testing/`配下)
- `internal/curlexec`に新規E2Eテストファイルを追加(`testcontainers-go`が root module の依存に追加される)
- 新規依存: `testcontainers-go`(root module)。モックサーバー側の依存(HTTPフレームワーク選定含む)は`testing/mockserver/go.mod`に閉じ、root moduleの依存グラフには影響しない
- `CLAUDE.md`の「Commands」セクションにE2Eテストの実行方法(Dockerが必要である旨)を追記する想定
