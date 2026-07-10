## MODIFIED Requirements

### Requirement: プラグマのcurl引数への変換
`.http`ファイルに記述されたプラグマ(`@insecure`、`@timeout`、`@no-redirect`、`@stream`)は、対応するcurlコマンドラインオプションに変換されなければならない(SHALL)。

#### Scenario: insecureプラグマの変換
- **WHEN** リクエストに`# @insecure`プラグマが付与されている
- **THEN** curl実行時に`-k`オプションが付与される

#### Scenario: timeoutプラグマの変換
- **WHEN** リクエストに`# @timeout 5s`プラグマが付与されている
- **THEN** curl実行時に`--max-time 5`オプションが付与される

#### Scenario: streamプラグマの変換
- **WHEN** リクエストに`# @stream`プラグマが付与されている
- **THEN** curl実行時に`-N`(`--no-buffer`)オプションが付与される

## ADDED Requirements

### Requirement: ストリーミング実行経路でのbody逐次読み取り
`@stream`プラグマが付与されたリクエストを実行する場合、lazycurlはbodyを一時ファイルへの書き出し完了を待つのではなく、curlプロセスの標準出力パイプから逐次読み取らなければならない(SHALL)。

#### Scenario: 標準出力パイプからの読み取り
- **WHEN** `@stream`プラグマ付きリクエストを送信する
- **THEN** curlの`-o`オプションには標準出力(`-`)が指定され、bodyは標準出力パイプから読み取られる

### Requirement: ストリーミング時の応答時間計測
`@stream`プラグマが付与されたリクエストでは、lazycurlは`-w '%{json}'`によるcurl自身の計測に依存せず、送信開始から終了(打ち切りを含む)までの経過時間を計測し応答時間として扱わなければならない(SHALL)。

#### Scenario: 正常終了時の計測
- **WHEN** `@stream`プラグマ付きリクエストがサーバー側の接続終了により完了する
- **THEN** 送信開始からプロセス終了までの経過時間が応答時間として記録される

#### Scenario: 中断時の計測
- **WHEN** `@stream`プラグマ付きリクエストの送信中に`ctrl-c`で中断される
- **THEN** 送信開始から中断までの経過時間が応答時間として記録される
