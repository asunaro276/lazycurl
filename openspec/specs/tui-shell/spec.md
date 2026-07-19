## Purpose

パネルレイアウト(コレクション/リクエスト一覧/レスポンスビュー/履歴)とlazygit系キーバインドによるTUIシェルを担う。リクエスト送信操作とレスポンスの基本表示(色分けバッジ含む)を提供する。

## Requirements

### Requirement: パネルベースのレイアウト
lazycurlは、`Request`編集フォーム・`Response`・`Collections`・`History`の4パネルを常時同時表示する固定レイアウトを提供しなければならない(SHALL)。モード切り替えによってパネル構成や表示パネル数が変化してはならない(SHALL NOT)。CollectionsパネルとHistoryパネルは、カーソルが乗っている1項目のみを展開表示し、他の項目は1行summaryに畳んで表示しなければならない(SHALL)。

#### Scenario: 起動時のレイアウト
- **WHEN** lazycurlを起動する
- **THEN** `[0] Request`・`[1] Response`・`[2] Collections`・`[3] History`の4パネルが同時に表示される

#### Scenario: Collectionsパネルのアコーディオン表示
- **WHEN** ユーザーが`[2] Collections`パネルでコレクション行の間をカーソル移動する
- **THEN** カーソルが乗っているコレクションのみリクエスト一覧が展開表示され、他のコレクションは名前のみの1行に畳まれる

### Requirement: lazygit互換キーバインド
lazycurlは、パネル間移動・上下移動・ヘルプ表示についてlazygitと共通のキーバインドを採用しなければならない(SHALL)。パネル番号キー(`0`-`3`)は、いずれかの入力欄が文字入力を占有するinsert状態にない限り、常にパネル間フォーカス移動として機能しなければならない(SHALL)。

#### Scenario: パネル間移動
- **WHEN** ユーザーが`tab`キーを押す
- **THEN** フォーカスが次のパネルへ移動する

#### Scenario: パネル番号による直接移動
- **WHEN** ユーザーがinsert状態にない状態で`0`-`3`のいずれかのキーを押す
- **THEN** 対応する`Request`/`Response`/`Collections`/`History`パネルへフォーカスが移動する。これは現在`[0] Request`パネルにフォーカスしていても同様に機能する

#### Scenario: 上下移動
- **WHEN** ユーザーが選択中のパネルで`j`/`k`キーを押す
- **THEN** リスト内の選択項目が下/上に移動する

#### Scenario: ヘルプ表示
- **WHEN** ユーザーが`?`キーを押す
- **THEN** 現在のパネルで利用可能なキーバインド一覧が表示される

### Requirement: リクエスト送信
ユーザーはリクエスト一覧から選択したリクエストを、キー操作で送信できなければならない(SHALL)。

#### Scenario: リクエストの送信
- **WHEN** ユーザーがリクエスト一覧でリクエストを選択し`enter`キーを押す
- **THEN** そのリクエストが送信され、結果がレスポンスパネルに表示される

### Requirement: レスポンス表示
レスポンスパネルはステータスコード・応答時間・レスポンスヘッダー・bodyを表示しなければならない(SHALL)。ステータスコードとHTTPメソッドは種別に応じて色分け表示しなければならない(SHALL)。`@stream`プラグマ付きリクエストの送信中は、送信完了を待たずに受信済みのbodyを逐次表示しなければならない(SHALL)。

#### Scenario: 成功レスポンスの表示
- **WHEN** ステータスコード200のレスポンスを受け取る
- **THEN** ステータスコードが2xx系の色で表示され、応答時間・body・headersが確認できる

#### Scenario: メソッドの色分け
- **WHEN** リクエスト一覧にGET/POST/DELETEのリクエストが並んでいる
- **THEN** それぞれのメソッドが異なる色のバッジで表示される

#### Scenario: ストリーミング送信中の逐次表示
- **WHEN** `@stream`プラグマ付きリクエストの送信中にbodyの断片が届く
- **THEN** レスポンスパネルの表示内容が送信完了を待たずに更新される

#### Scenario: 非ストリーミング送信中の表示
- **WHEN** `@stream`プラグマの無いリクエストが送信中である
- **THEN** レスポンスパネルは送信完了まで「送信中...」の表示を維持する

### Requirement: 実行履歴の表示
lazycurlは送信済みリクエストの履歴を一覧表示しなければならない(SHALL)。Historyパネルはカーソルが乗っているエントリのみを展開して詳細を表示し、他のエントリは1行summaryに畳んで表示しなければならない(SHALL)。

#### Scenario: 履歴からの再表示
- **WHEN** ユーザーが`[3] History`パネルでエントリにカーソルを合わせ`enter`キーを押す
- **THEN** その時点でのレスポンス内容が`[1] Response`パネルに表示される

#### Scenario: カーソル移動によるプレビュー展開
- **WHEN** ユーザーが`[3] History`パネルで`j`/`k`によりカーソルを移動する
- **THEN** カーソルが乗ったエントリがパネル内で展開表示され、他のエントリは1行summaryに畳まれる。`[1] Response`パネルの表示は`enter`が押されるまで変化しない

### Requirement: ステータスバーの自動クリア
lazycurlは、エラーや警告によってステータスバーにメッセージが表示された場合、表示から5秒後に自動的にキーバインド一覧の表示へ戻さなければならない(SHALL)。5秒以内に新しいメッセージが表示された場合は、そのメッセージ自身の表示から新たに5秒が計測されなければならない(SHALL)。

#### Scenario: エラーメッセージの自動クリア
- **WHEN** リクエスト送信でエラーが発生し、ステータスバーにエラーメッセージが表示される
- **THEN** ユーザーが何も操作しなくても、5秒後にステータスバーの表示がキーバインド一覧に戻る

#### Scenario: 連続したエラーでのタイマーの上書き
- **WHEN** ステータスバーにエラーメッセージが表示されてから3秒後に、別の操作で新しいエラーメッセージが表示される
- **THEN** 古いメッセージのタイマーによってキーバインド表示に戻ることはなく、新しいメッセージの表示から5秒後にキーバインド表示に戻る

### Requirement: Requestパネルのnormal/insert状態
`[0] Request`パネルは、フィールド間を移動するnormal状態と、単一フィールドへの文字入力を行うinsert状態の2段階のフォーカス状態を持たなければならない(SHALL)。normal状態でのみパネル番号キー(`0`-`3`)によるパネル切替が機能しなければならない(SHALL)。

#### Scenario: insert状態への遷移
- **WHEN** ユーザーが`[0] Request`パネルのnormal状態でフィールドにカーソルを合わせ`enter`キーを押す
- **THEN** そのフィールドがinsert状態になり文字入力を受け付ける。この間`0`-`3`キーはパネル切替として機能しない

#### Scenario: insert状態からの離脱
- **WHEN** ユーザーがinsert状態で`esc`キーを押す
- **THEN** フィールドがnormal状態に戻り、`0`-`3`キーによるパネル切替が再び機能する

### Requirement: collectionに属さないRequestの一時性
lazycurlは、collectionに属していないRequestを、保存されるまでメモリ内にのみ保持しなければならない(SHALL)。保存されていないRequestは、アプリケーション終了時に破棄されなければならない(SHALL)。collectionに属していないRequestでは`{{variable}}`の展開を行ってはならない(SHALL NOT)。

#### Scenario: 未保存での終了
- **WHEN** collectionに属さない状態でリクエストを組み立てたが保存せずにlazycurlを終了する
- **THEN** 次回起動時にそのリクエストは復元されない

#### Scenario: environment変数展開の非対象
- **WHEN** collectionに属さない状態でリクエストを組み立てる
- **THEN** `{{variable}}`の展開は行われず、入力した値がそのまま送信に使われる

### Requirement: collectionへの保存
ユーザーは`[0] Request`パネルで組み立てたcollectionに属さないリクエストを、任意のタイミングでcollectionへ保存できなければならない(SHALL)。保存時には既存collectionの選択、または新規collection作成のいずれかを選べなければならない(SHALL)。

#### Scenario: 保存操作の開始
- **WHEN** ユーザーが`[0] Request`パネルでcollectionに属さないリクエストを編集中に`ctrl+s`を押す
- **THEN** 保存先として既存collectionの選択、または新規collection作成のいずれかを選べるプロンプトが表示される

#### Scenario: 既存collectionへの保存
- **WHEN** ユーザーが保存先プロンプトで既存collectionを選択する
- **THEN** そのリクエストが選択したcollectionの`.http`ファイルに`###`ブロックとして追記される

#### Scenario: 新規collection作成による保存
- **WHEN** ユーザーが保存先プロンプトで新規collection名を入力する
- **THEN** 新しいcollectionファイルが作成され、そのリクエストが最初のリクエストとして保存される

#### Scenario: 保存後のフォーカス
- **WHEN** collectionに属さないリクエストの保存が完了する
- **THEN** `[2] Collections`パネルで保存先collectionと保存したリクエストが選択された状態になる
