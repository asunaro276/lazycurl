## Why

現状のlazycurlは、curlプロセスが終了しレスポンス全体をファイルに書き終えるまでResponseパネルに何も表示できない(`Executor.Execute`は`cmd.Output()`でブロックし、bodyを一時ファイルへ書き切ってから読む設計)。SSE(Server-Sent Events)のような、接続を維持したまま断続的にbodyが届くAPIをテストする際、送信完了(あるいはタイムアウト/手動打ち切り)までResponseパネルが「送信中...」のまま何も見えないのは実用上のボトルネックになっている。

## What Changes

- 新しいプラグマ`@stream`を追加し、リクエスト単位でストリーミング表示を明示的にオプトインできるようにする
- `@stream`が指定されたリクエストは、curlに`-N`(`--no-buffer`)を付与し、bodyを一時ファイル経由ではなくstdoutパイプから逐次読み取る実行経路(`ExecuteStreaming`)を通す
- `@stream`時は`-w '%{json}'`によるtime_total計測をやめ、Go側の実測経過時間を`Response.TimeTotal`として使う(ctrl-cによる途中打ち切り時はcurl自身の`-w`出力が得られないため)
- Responseパネルは、`@stream`送信中もbodyが届くたびに逐次再描画する(SSEイベントのパース・整形は行わず、生bodyをそのまま追記表示する)。`@stream`でない送信は現行通り完了まで「送信中...」表示のまま
- ctrl-cによる送信中断は現行の`cancelSend`をそのまま使う。中断時点までに受信済みのbodyを最終的な`Response`として履歴(History)に確定させる

## Capabilities

### New Capabilities
- `streaming-response`: `@stream`プラグマの解釈、stdoutパイプ経由の逐次読み取り実行経路、Bubble Tea側への逐次通知の仕組み、Responseパネルの逐次描画、および中断時のHistory確定を含む

### Modified Capabilities
- `curl-execution`: `@stream`プラグマからcurlオプション(`-N`)への変換、および`@stream`時の実行経路(stdoutストリーム読み取り、`-w`計測の代替)に関する要件を追加する
- `tui-shell`: Responseパネルの表示要件に、`@stream`送信中の逐次描画に関する要件を追加する

## Impact

- `internal/httpfile/`: プラグマのパース/シリアライズに`@stream`を追加(`types.go`/`parse.go`)
- `internal/curlexec/`: `argv.go`(`-N`付与)、新規のストリーミング実行経路(`executor.go`または新ファイル)、`Runner`インターフェースの拡張または並行するストリーミング用抽象の追加
- `internal/tui/shell/`(`model.go`/`update.go`/`view.go`): 逐次到着メッセージのハンドリング、live response状態の保持、Responseパネルの逐次描画
- 並行して進行中の`adhoc-request-mode`変更も同じ`internal/tui/shell/`配下のファイルに手を入れる予定のため、着地順序に注意が必要(実装フェーズで調整)
