## ADDED Requirements

### Requirement: モックサーバーは独立したGoモジュールとして実装される
モックHTTPサーバーは、lazycurl本体のroot moduleとは別に独自の`go.mod`を持つGoモジュールとして`testing/mockserver/`配下に実装されなければならない(MUST)。root moduleのGoコードはモックサーバーのパッケージをimportしてはならない(MUST NOT)。自動テスト・手動確認のいずれもHTTP経由でのみモックサーバーと通信する。

#### Scenario: root moduleのビルドがモックサーバーの依存を含まない
- **WHEN** root moduleで`go build ./...`または`go mod tidy`を実行する
- **THEN** `testing/mockserver/go.mod`に記載された依存(HTTPフレームワーク等)がroot moduleの`go.mod`/`go.sum`に現れない

### Requirement: モックサーバーは単一のDockerfileからビルド可能である
モックサーバーは単一の`Dockerfile`からコンテナイメージとしてビルド可能でなければならない(MUST)。このDockerfileは自動テスト(testcontainers-go)と手動確認(docker compose)の両方から参照される、唯一のビルド元となる。

#### Scenario: Dockerfileから単独でイメージがビルドできる
- **WHEN** `testing/mockserver/`をbuild contextとして`docker build`を実行する
- **THEN** モックサーバーの実行可能なコンテナイメージが生成される

### Requirement: モックサーバーはcurl argvの検証に必要なエンドポイント群を提供する
モックサーバーは以下のエンドポイントを提供しなければならない(MUST):
- `/echo`: リクエストのメソッド・ヘッダー・bodyをJSONで返す
- `/status/{code}`: 指定したHTTPステータスコードで応答する
- `/redirect/{n}`: `n`回のリダイレクトを経て最終的に200を返す
- `/delay/{sec}`: 指定秒数待機してから応答する
- `/stream`: レスポンスbodyを複数チャンクに分け、時間差(chunked transfer)で逐次送出する。分割数・送出間隔をクエリパラメータ(`chunks`/`interval`)で指定できる
- `/auth/basic`, `/auth/bearer`: 送信された`Authorization`ヘッダーの妥当性を検証し、結果を返す

#### Scenario: /statusが指定コードを返す
- **WHEN** `/status/404`にリクエストを送る
- **THEN** レスポンスのHTTPステータスコードが404になる

#### Scenario: /redirectが指定回数のリダイレクトを発生させる
- **WHEN** `/redirect/2`にリクエストを送る
- **THEN** 2回のリダイレクト応答(3xx)を経て、最終的に200が返る

#### Scenario: /delayが指定秒数応答を遅延させる
- **WHEN** `/delay/3`にリクエストを送る
- **THEN** リクエスト送信からおよそ3秒後に応答が返る

#### Scenario: /streamがbodyを複数チャンクに分けて逐次送出する
- **WHEN** `/stream`にリクエストを送る
- **THEN** レスポンスbodyが単一の送出ではなく、時間差のある複数チャンクとしてクライアントに到達する

#### Scenario: /streamのチャンク数・間隔をクエリパラメータで指定できる
- **WHEN** `/stream?chunks=5&interval=500`にリクエストを送る
- **THEN** レスポンスbodyが5個のチャンクに分かれ、約500ミリ秒間隔でクライアントに到達する

#### Scenario: /auth/basicが正しい認証情報を検証する
- **WHEN** 正しいユーザー名・パスワードで`Authorization: Basic ...`ヘッダーを付けて`/auth/basic`にリクエストを送る
- **THEN** 200が返る

#### Scenario: /auth/bearerが不正なトークンを拒否する
- **WHEN** 不正なトークンで`Authorization: Bearer ...`ヘッダーを付けて`/auth/bearer`にリクエストを送る
- **THEN** 401が返る

### Requirement: モックサーバーはdocker composeで固定ポート公開して手動起動できる
リポジトリは、モックサーバーのDockerfileを参照し固定のホストポートで公開する`docker-compose.yml`を含まなければならない(MUST)。開発者は`docker compose up`のみでモックサーバーを常駐起動できなければならない(MUST)。

#### Scenario: docker compose upで常駐起動できる
- **WHEN** 開発者が`docker compose up`を実行する
- **THEN** モックサーバーが固定のホストポートで待受を開始し、`.http`ファイルのリクエストからそのポート宛にアクセスできる状態になる
