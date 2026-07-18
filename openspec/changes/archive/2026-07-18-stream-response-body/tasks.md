## 1. `@stream`プラグマのパース/シリアライズ

- [x] 1.1 `httpfile.Pragmas`に`Stream bool`フィールドを追加する
- [x] 1.2 `@stream`プラグマコメントのパース実装(`internal/httpfile/parse.go`)
- [x] 1.3 `Stream`プラグマを`.http`+プラグマ形式へシリアライズする実装
- [x] 1.4 パーサ/シリアライザの往復変換テストに`@stream`のケースを追加する

## 2. curl argv構築への反映

- [x] 2.1 `buildArgs`(`internal/curlexec/argv.go`)で`Stream`プラグマ付き時に`-N`(`--no-buffer`)を付与する
- [x] 2.2 ストリーミング時は`-o`に一時ファイルではなく標準出力(`-`)を指定する分岐を実装する
- [x] 2.3 argv構築の単体テストに`@stream`ケース(`-N`付与、`-o -`)を追加する

## 3. ストリーミング実行経路(`internal/curlexec`)

- [x] 3.1 `cmd.StdoutPipe()`でcurlの標準出力を逐次読み取る実行関数(例: `ExecuteStreaming`)を実装する(既存の`Execute`には手を入れない)
- [x] 3.2 読み取ったchunkを外部へ通知するためのchannelベースのAPIを設計・実装する(`Runner`同様、テスト時に実プロセス無しで差し替え可能な形にする)
- [x] 3.3 `-D headerFile`から得たヘッダー/ステータスコードを取得する処理を既存の`parseHeaderFile`を再利用して実装する
- [x] 3.4 送信開始からの経過時間をGo側で計測し`Response.TimeTotal`として扱う実装(`-w '%{json}'`には依存しない)
- [x] 3.5 `ctrl-c`によるコンテキストキャンセル時、その時点までに受信済みのchunkを結合した`Response`を返す実装
- [x] 3.6 プロセス自然終了時(サーバー側の接続クローズ等)も同様に受信済みchunkを結合した`Response`を返す実装
- [x] 3.7 ストリーミング実行経路のユニットテスト(モックによるchunk受信・中断・自然終了の各パターン)

## 4. Bubble Teaへの逐次通知

- [x] 4.1 chunk受信を表す`tea.Msg`(例: `streamChunkMsg`)とストリーム終了を表す`tea.Msg`(例: `streamDoneMsg`)を定義する(`internal/tui/shell/update.go`)
- [x] 4.2 channelから1件読み取って`tea.Msg`を返すCmdを実装し、`Update`内でメッセージ処理後に同じCmdを再度積み直す(subscribeパターン)
- [x] 4.3 `sendCurrent()`相当の送信開始処理で、`Stream`プラグマの有無により`Execute`/`ExecuteStreaming`のどちらを呼ぶか分岐する

## 5. Shellの状態管理とResponseパネルの逐次描画

- [x] 5.1 `Shell`にlive response状態(受信中の`*curlexec.Response`)を保持するフィールドを追加する
- [x] 5.2 `streamChunkMsg`受信時にlive responseのbodyへ追記し、`Update`が再描画をトリガーする
- [x] 5.3 `streamDoneMsg`(打ち切り・自然終了いずれも)受信時にlive responseを`HistoryEntry`として`History`に確定pushする(既存の`sendResultMsg`処理と揃える)
- [x] 5.4 `viewResponse()`を、`@stream`送信中はlive responseを`renderResponse`で逐次描画し、非streamの送信中は既存通り「送信中...」を表示するよう分岐させる(`internal/tui/shell/view.go`)

## 6. 結合・検証

- [ ] 6.1 SSEを返す簡易テストサーバー(またはモック)を用いた手動確認(逐次表示、`ctrl-c`打ち切り、自然終了の3パターン)
- [x] 6.2 `@stream`プラグマ無しリクエストの既存挙動に回帰が無いことを確認する
- [x] 6.3 README等のドキュメントに`@stream`プラグマの説明を追記する
