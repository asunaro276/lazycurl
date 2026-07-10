## Purpose

`.http`リクエストを`curl`サブプロセス実行に変換し、レスポンス(ヘッダー/body/タイミング)を構造化して受け取る。自前のHTTPクライアント実装は用いず、`curl`バイナリの存在確認・プラグマのオプション変換・エラーハンドリングを担う。

## Requirements

### Requirement: curlサブプロセスによるリクエスト実行
lazycurlは変数展開済みのリクエストを、`curl`バイナリのサブプロセス実行に変換して送信しなければならない(SHALL)。自前のHTTPクライアント実装(net/http等)でリクエストを送信してはならない。

#### Scenario: GETリクエストの実行
- **WHEN** ユーザーがGETリクエストの送信を実行する
- **THEN** 対応する`curl`コマンドがサブプロセスとして起動され、レスポンスが返る

#### Scenario: JSON bodyを持つPOSTリクエストの実行
- **WHEN** ユーザーがJSON bodyを持つPOSTリクエストを送信する
- **THEN** bodyは一時ファイルに書き出され、`--data-binary @<tmpfile>`としてcurlに渡される

### Requirement: レスポンスの構造化取得
lazycurlはcurl実行結果から、レスポンスヘッダー・body・タイミング情報(ステータスコード、応答時間等)を分離して取得しなければならない(SHALL)。

#### Scenario: ヘッダーとbodyの分離取得
- **WHEN** リクエストを送信する
- **THEN** レスポンスヘッダーは`-D`オプションで、bodyは`-o`オプションで別々のファイルに出力され、それぞれ個別に読み取られる

#### Scenario: タイミング情報の取得
- **WHEN** リクエストを送信する
- **THEN** `-w '%{json}'`により標準出力からステータスコード・応答時間等のメタデータがJSONとして取得され、TUIに表示される

### Requirement: プラグマのcurl引数への変換
`.http`ファイルに記述されたプラグマ(`@insecure`、`@timeout`、`@no-redirect`)は、対応するcurlコマンドラインオプションに変換されなければならない(SHALL)。

#### Scenario: insecureプラグマの変換
- **WHEN** リクエストに`# @insecure`プラグマが付与されている
- **THEN** curl実行時に`-k`オプションが付与される

#### Scenario: timeoutプラグマの変換
- **WHEN** リクエストに`# @timeout 5s`プラグマが付与されている
- **THEN** curl実行時に`--max-time 5`オプションが付与される

### Requirement: リクエストのキャンセル
実行中のリクエストはユーザーの操作によって中断できなければならない(SHALL)。

#### Scenario: 実行中リクエストの中断
- **WHEN** ユーザーが応答待ちの状態で中断操作(`ctrl-c`等)を行う
- **THEN** 実行中のcurlサブプロセスが終了され、TUIは中断された旨を表示する

### Requirement: エラーの人間可読な表示
curlの終了コードは、技術的な生の値ではなく人間可読なメッセージに変換して表示しなければならない(SHALL)。

#### Scenario: 接続失敗時のメッセージ
- **WHEN** curlが終了コード7(接続失敗)で終了する
- **THEN** TUIには「接続できませんでした」のような人間可読なエラーメッセージが表示される

### Requirement: curlバイナリの検証
lazycurlは起動時に`curl`バイナリの存在とバージョンを検証しなければならない(SHALL)。

#### Scenario: curl未検出
- **WHEN** システムに`curl`コマンドが存在しない状態でlazycurlを起動する
- **THEN** curlが見つからない旨のエラーメッセージを表示して起動を中止する

#### Scenario: バージョン不足
- **WHEN** インストールされている`curl`のバージョンが7.70未満である
- **THEN** `%{json}` write-outが使用できない旨の警告を表示する
