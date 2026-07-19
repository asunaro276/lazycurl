## ADDED Requirements

### Requirement: TUI E2Eテストはteatestで実際のtea.Programを駆動する
`internal/tui`のE2Eテストは、`teatest`(`github.com/charmbracelet/x/exp/teatest`)を用いて`tui.App`を実際の`tea.Program`として起動しなければならない(MUST)。`Update()`をテストコードから直接呼び出す既存の手法(モデル単体テスト)には依存せず、キー入力の送出には`teatest`が提供するAPI(`tm.Send`等)を使わなければならない(MUST)。

#### Scenario: teatestでApp(tea.Program)が起動する
- **WHEN** `teatest.NewTestModel`で`tui.App`を初期化する
- **THEN** `tea.Program`が実際に起動し、初期状態のView出力が取得できる

### Requirement: TUI E2Eテストは実curlサブプロセス経由でモックサーバーに対してリクエストを送る
TUI E2Eテストは、`tui.New(...)`に渡す`curlexec.Executor`として`curlexec.NewExecutor()`(実curlサブプロセスを起動する実装)を使用しなければならない(MUST)。テストは`testing/mockserver`のコンテナを対象URLとするCollection/Requestを用意し、キー入力による送信操作が実際のHTTP通信としてモックサーバーに届くことを検証しなければならない(MUST)。

#### Scenario: キー操作でCollectionsからRequestを選択し送信する
- **WHEN** teatestで起動した`tui.App`に対し、Collectionsパネルへの移動・Request選択・送信キーに相当するキー入力を順に送る
- **THEN** モックサーバーが対応するエンドポイントへのHTTPリクエストを受信する

### Requirement: 送信結果はResponseパネルの描画に反映される
TUI E2Eテストは、送信後にResponseパネルへモックサーバーからのレスポンス内容(ステータスコード・body等)が描画されることを、`teatest`の出力(`teatest.WaitFor`または`FinalOutput`で取得する端末出力)から検証しなければならない(MUST)。

#### Scenario: レスポンス内容がResponseパネルに描画される
- **WHEN** モックサーバーの`/status/200`に対するリクエストを送信し、送信完了を待つ
- **THEN** `teatest`で取得した端末出力にステータスコード200を示す表示が含まれる

### Requirement: TUI E2Eテストのモックサーバーコンテナはテストバイナリごとに1つだけ起動され共有される
`internal/tui`のE2Eテストは、`internal/curlexec`のE2Eテストと同様に`TestMain`でモックサーバーコンテナを1回だけ起動し、パッケージ内の全TUI E2Eテストで共有しなければならない(MUST)。個々のテストケースごとにコンテナを起動し直してはならない(MUST NOT)。

#### Scenario: 複数のTUI E2Eテストケースが同一コンテナを再利用する
- **WHEN** `internal/tui`パッケージ内に複数のTUI E2Eテストケースが存在する状態で`go test`を実行する
- **THEN** モックサーバーコンテナの起動は1回のみ発生し、全テストケースがそのコンテナに対してリクエストを送る

### Requirement: `@stream`送信中はResponseパネルが送信完了前に逐次描画される
`@stream`プラグマ付きリクエストをTUI E2Eテストで送信する場合、送信完了(終端の`streamDoneMsg`到達)より前に、Responseパネルへ部分的な受信済みbodyが描画されていることを`teatest`の出力から検証しなければならない(MUST)。

#### Scenario: 送信完了前にResponseパネルへ部分的な内容が表示される
- **WHEN** `@stream`プラグマ付きリクエストで`/stream`(複数チャンク・間隔ありの設定)を送信し、送信完了前のタイミングで`teatest`の出力を確認する
- **THEN** その時点で受信済みのチャンクに相当する内容がResponseパネルの表示に含まれ、まだ全チャンク分の内容は表示されていない

### Requirement: `@stream`送信中のctrl-c打ち切りが打ち切り時点までの内容をHistoryに確定する
TUI E2Eテストは、`@stream`プラグマ付きリクエストの送信中に`ctrl-c`に相当するキー入力を送り、その時点までに受信済みのbodyを持つレスポンスがHistoryへ確定表示されることを検証しなければならない(MUST)。

#### Scenario: ctrl-c相当のキー入力で送信を打ち切りHistoryに確定する
- **WHEN** `@stream`プラグマ付きリクエストの送信中(受信途中)に`ctrl-c`に相当するキー入力を送る
- **THEN** Historyパネルにその時点までに受信済みのbodyを持つレスポンスが追加される
