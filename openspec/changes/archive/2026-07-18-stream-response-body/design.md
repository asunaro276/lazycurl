## Context

現在の`internal/curlexec.Executor.Execute`は完全なバッチ実行モデルである。

- bodyは`-o outFile`で一時ファイルに書き出させ、`os/exec`の`cmd.Output()`でcurlプロセスの終了まで待つ
- 終了後に`headerFile`/`outFile`を一括で読み、`-w '%{json}'`の標準出力からステータスコード・応答時間を得る
- TUI側(`internal/tui/shell`)は`sendCurrent()`が返す`tea.Cmd`が完了時に1回だけ`sendResultMsg`を発火し、`Shell.Update`がそれを`History`に確定させる。送信中は`viewResponse()`が固定文字列「送信中... (ctrl-c で中断)」を返すのみで、途中経過を表示する仕組みは無い

SSEのような接続維持型のレスポンスをテストする際、この設計だと送信完了(またはタイムアウト/`ctrl-c`打ち切り)までResponseパネルに何も表示されない。

## Goals / Non-Goals

**Goals:**
- `@stream`プラグマが付与されたリクエストで、bodyが届いた分だけResponseパネルに逐次表示する
- `ctrl-c`による途中打ち切り時も、それまでに受信済みのbodyをHistoryに確定させる
- 既存の非ストリーミング実行経路(`Execute`、`-o`/`-w`によるバッチ取得)には手を入れない。`@stream`が付与されていないリクエストの挙動・実行経路は変更しない

**Non-Goals:**
- SSEの`event:`/`data:`フレームをパースし、イベント単位で分割表示すること(生bodyをそのまま追記表示するのみ)
- WebSocketなど、curlの単純なリクエスト/レスポンスモデルを超えるプロトコルへの対応
- リクエストbody側のストリーミング(アップロード進捗表示)
- 逐次表示中のスクロール位置制御やページング等、表示の高度化

## Decisions

### `@stream`はオプトイン、`-N`(`--no-buffer`)を付与する
既存の`@insecure`/`@timeout`/`@no-redirect`と同じ形式のプラグマとして追加する。付与時は`buildArgs`相当の処理で`-N`を追加し、curl内部のバッファリングによる出力の遅延到着を防ぐ。付与されない限り現行のバッチ実行経路のまま。

代替案として「レスポンスヘッダーの`Content-Type: text/event-stream`を見て自動判定する」も検討したが、送信前にはcontent-typeが分からずcurl起動オプションを事前に決められない(`-N`はプロセス起動時にしか指定できない)ため不採用。

### ストリーミング時は`-o outFile`をやめ、`-o -`でstdoutパイプから読む
既存のtailポーリング方式(ファイルを一定間隔で読み直す)も検討したが、ポーリング間隔分のレイテンシが発生し、ファイル成長中の読み取り境界(マルチバイト文字が途中で切れる等)の面倒を自前で処理する必要がある。`cmd.StdoutPipe()` + `bufio.Reader`でストリームをそのまま読み進める方式の方が、レイテンシもコードの複雑さも小さい。

`-D headerFile`によるヘッダー取得は現行のまま維持する(ヘッダーはbody転送開始前に書き出されるため、bodyのストリーミングと衝突しない)。

### `-w '%{json}'`はストリーミング時に使わず、Go側で経過時間を計測する
`-w`の出力はcurlプロセスが自身の判断で転送を完了した場合にのみ標準出力の末尾に出力される。`ctrl-c`によるコンテキストキャンセルでプロセスを強制終了した場合、`-w`の出力は得られない。ストリーミングは打ち切りが常用される操作であるため、`-w`に依存せずGo側で送信開始からの経過時間を計測し`Response.TimeTotal`として使う。ステータスコードは`-D`で得られる`headerFile`から取得する(現行の`parseHeaderFile`をそのまま流用できる)。

### Bubble TeaへはPub/Subチャンネル + 自己再発行Cmdパターンで通知する
`*tea.Program`への参照をExecutor/business logicに持ち込み`p.Send()`を直接呼ぶ方式も検討したが、既存コードは`Runner`インターフェース(`NewExecutorWithRunner`)によってテスト時に実プロセスを起動せず差し替えられる設計になっている。ストリーミング経路も同様に、chunkを流すchannelを介する構造にすることで、テスト時は本物のcurlプロセスなしにchannelへ直接値を流し込んでUpdateの挙動を検証できる。具体的には、chunk受信のたびに1件読んで`tea.Msg`を返すCmdを、`Update`側でメッセージ処理後に同じCmdを積み直す(定番のsubscribeパターン)。

### Live response状態は`Shell`に保持し、確定時のみ`History`へpush
`viewResponse()`が`s.sending`中に逐次描画できるよう、`Shell`に`liveResponse *curlexec.Response`相当のフィールドを追加し、chunk受信のたびにbodyを追記する。`History`への確定pushは現行通り、送信完了(または`ctrl-c`打ち切り)の1点でのみ行う。これにより`History`は「確定した送信結果のみを保持する」という既存の不変条件を崩さない。

## Risks / Trade-offs

- [`-N`を付与してもOS/ネットワーク層のバッファリングにより、期待通り逐次到着しないサーバーが存在しうる] → 対応はcurlに委ねる(lazycurl側でできることは`-N`の付与まで)。本機能はあくまで「サーバーが逐次送ってくる分は逐次見せる」ものであり、全ての環境で滑らかな逐次表示を保証するものではない
- [ストリーミング実行経路とバッチ実行経路が並存することでExecutor内のコード分岐が増える] → 既存の`Execute`には手を入れず、新規メソッド(例: `ExecuteStreaming`)として完全に分離することで、既存の非ストリーミング挙動への影響をゼロに保つ
- [同時並行で進行中の`adhoc-request-mode`変更も`internal/tui/shell/`配下の同じファイル(`model.go`/`update.go`/`view.go`)を変更する予定であり、マージ順序次第でコンフリクトが発生する] → 実装着手前にどちらを先行させるか、あるいはマージ順を調整する

## Open Questions

- `@stream`かつ`@timeout`が併用された場合の挙動は、既存の`@timeout`の意味(`--max-time`によるcurl自体のタイムアウト)をそのまま適用し、特別扱いはしない方針。実装時に想定通りか再確認する
