## Purpose

`@stream`プラグマの解釈、stdoutパイプ経由の逐次読み取り実行経路、Bubble Tea側への逐次通知の仕組み、Responseパネルの逐次描画、および中断時のHistory確定を担う。SSE(Server-Sent Events)のような接続維持型のレスポンスを、送信完了(またはタイムアウト/`ctrl-c`打ち切り)を待たずにResponseパネルへ逐次表示するための機能群。

## Requirements

### Requirement: `@stream`プラグマによるストリーミング表示のオプトイン
lazycurlは`.http`ファイルのリクエストに`@stream`プラグマが付与されている場合、そのリクエストの送信をストリーミング実行経路で行わなければならない(SHALL)。`@stream`が付与されていないリクエストは既存のバッチ実行経路のまま変更されない。

#### Scenario: streamプラグマ付きリクエストの送信
- **WHEN** ユーザーが`# @stream`プラグマ付きのリクエストを送信する
- **THEN** レスポンスbodyが一時ファイルへの書き出し完了を待たず、逐次読み取りされる実行経路で処理される

#### Scenario: streamプラグマ無しリクエストの送信
- **WHEN** ユーザーが`@stream`プラグマの無いリクエストを送信する
- **THEN** 現行通りcurlプロセスの終了を待ってからレスポンスが確定表示される

### Requirement: Responseパネルの逐次描画
`@stream`プラグマ付きリクエストの送信中、Responseパネルは受信済みのbodyを到着のたびに追記表示しなければならない(SHALL)。SSEの`event:`/`data:`等のフレーム解釈は行わず、受信した生バイト列をそのまま表示に反映する。

#### Scenario: bodyの逐次追記表示
- **WHEN** `@stream`プラグマ付きリクエストの送信中にサーバーから新しいbodyの断片が届く
- **THEN** Responseパネルの表示内容に届いた断片が追記される

### Requirement: 打ち切り時のHistory確定
`@stream`プラグマ付きリクエストの送信が`ctrl-c`等により途中で中断された場合、lazycurlはその時点までに受信済みのbodyを最終的なレスポンスとして実行履歴(History)に確定させなければならない(SHALL)。

#### Scenario: ストリーミング中の中断
- **WHEN** ユーザーが`@stream`プラグマ付きリクエストの送信中に`ctrl-c`で中断する
- **THEN** 中断時点までに受信済みのbodyを持つレスポンスがHistoryに追加される

#### Scenario: ストリーミングの自然終了
- **WHEN** `@stream`プラグマ付きリクエストの送信中にサーバー側が接続を閉じてcurlプロセスが終了する
- **THEN** それまでに受信済みのbodyを持つレスポンスがHistoryに追加される
