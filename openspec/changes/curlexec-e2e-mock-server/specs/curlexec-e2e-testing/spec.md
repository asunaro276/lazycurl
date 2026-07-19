## ADDED Requirements

### Requirement: E2Eテストはtestcontainers-goでモックサーバーコンテナを起動する
`internal/curlexec`のE2Eテストは、`testcontainers-go`を用いて`testing/mockserver/Dockerfile`からイメージをビルドし、コンテナとして起動しなければならない(MUST)。事前にビルド済みのイメージをレジストリから取得する方式には依存しない。

#### Scenario: Dockerが利用可能な環境でE2Eテストがコンテナを起動する
- **WHEN** Dockerデーモンが起動しているマシンで`go test ./internal/curlexec/...`を実行する
- **THEN** `testing/mockserver/Dockerfile`からイメージがビルドされ、モックサーバーのコンテナが起動する

### Requirement: モックサーバーコンテナはテストバイナリごとに1つだけ起動され共有される
E2Eテストは、`TestMain`でモックサーバーコンテナを1回だけ起動し、そのベースURLをパッケージ内の全E2Eテストで共有しなければならない(MUST)。個々のテストケースごとにコンテナを起動し直してはならない(MUST NOT)。テストバイナリの終了時にコンテナを終了させなければならない(MUST)。

#### Scenario: 複数のE2Eテストケースが同一コンテナを再利用する
- **WHEN** 同一パッケージ内に複数のE2Eテストケースが存在する状態で`go test`を実行する
- **THEN** モックサーバーコンテナの起動は1回のみ発生し、全テストケースがそのコンテナに対してリクエストを送る

#### Scenario: テストバイナリ終了時にコンテナが終了する
- **WHEN** パッケージ内の全テストの実行が完了する
- **THEN** 起動していたモックサーバーコンテナが終了(terminate)される

### Requirement: E2Eテストは実curlサブプロセス経由でExecutorの挙動を検証する
E2Eテストは`Runner`を`fakeRunner`に差し替えず、`curlexec.NewExecutor()`(実curlサブプロセスを起動する実装)を用いてモックサーバーに対してリクエストを送り、`buildArgs`が生成するcurlフラグ(`-X`/`-H`/`-L`/`--max-time`/`--data-binary`/`-D`/`-o`/`-w '%{json}'`)が実際のHTTP通信に対して正しく機能することを検証しなければならない(MUST)。

#### Scenario: リダイレクト追従(-L)が実curl経由で機能する
- **WHEN** `@no-redirect`を指定しないリクエストでモックサーバーの`/redirect/1`に対して`Executor.Execute`を呼ぶ
- **THEN** 最終的なレスポンスのステータスコードが200になる

#### Scenario: タイムアウト(--max-time)が実curl経由で機能する
- **WHEN** `@timeout`に短い時間を指定したリクエストでモックサーバーの`/delay/{長い秒数}`に対して`Executor.Execute`を呼ぶ
- **THEN** 指定したタイムアウト時間内にエラーが返り、応答を待ち続けない

#### Scenario: Basic認証ヘッダーが実curl経由で正しく送信される
- **WHEN** `Auth`にBasic認証情報を設定したリクエストでモックサーバーの`/auth/basic`に対して`Executor.Execute`を呼ぶ
- **THEN** レスポンスのステータスコードが200になる

### Requirement: E2Eテストは実curlサブプロセス経由で`@stream`の逐次配信を検証する
E2Eテストは、`Pragmas.Stream`を設定したリクエストで`Executor.ExecuteStreaming`をモックサーバーの`/stream`に対して呼び、返される`<-chan StreamEvent`が実際のHTTP接続を通じて複数回の`StreamEvent{Chunk: ...}`を経てから終端の`StreamEvent{Done: ...}`に到達することを検証しなければならない(MUST)。

#### Scenario: /streamからの応答が複数回のチャンクとして届く
- **WHEN** `chunks=3`以上を指定した`/stream`に対して`Executor.ExecuteStreaming`を呼び、返されたchannelを最後まで読み切る
- **THEN** `Done`が届く前に2回以上`Chunk`付きの`StreamEvent`を受信する

#### Scenario: 自然終了時に全チャンクを連結したbodyが確定する
- **WHEN** キャンセルせずに`/stream`からの応答を最後まで受信する
- **THEN** 終端の`StreamDone.Response.Body`が送出された全チャンクを結合した内容と一致し、`StreamDone.Err`が`nil`になる

### Requirement: E2Eテストは実curlサブプロセス経由で`@stream`のctrl-c打ち切りを検証する
E2Eテストは、`Executor.ExecuteStreaming`に渡す`context.Context`を送信途中(最初のチャンク受信後、完了前)でキャンセルし、`ctrl-c`による打ち切りと同等の状況を実curl経由で再現・検証しなければならない(MUST)。

#### Scenario: 送信途中のキャンセルで部分bodyが確定する
- **WHEN** `/stream`からの応答を一部のチャンクだけ受信した時点で`ExecuteStreaming`に渡したcontextをキャンセルする
- **THEN** 終端の`StreamDone.Response.Body`がキャンセル時点までに受信済みのチャンクのみで構成され、`StreamDone.Err`が`nil`になる(キャンセルは失敗ではなく正常な早期終了として扱われる)
