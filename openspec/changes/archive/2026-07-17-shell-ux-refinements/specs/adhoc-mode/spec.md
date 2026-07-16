## MODIFIED Requirements

### Requirement: モード切り替え
lazycurlは`Adhoc`と`Collections`の2つのモードを持ち、`[`/`]`キーで相互に切り替えられなければならない(SHALL)。ただし、フォームのcontentゾーン(Params/Headers/Auth/Body)にフォーカスがある間、`[`/`]`キーはサブセクションの切り替えに使われ、モード切り替えには使われない(SHALL NOT)。現在アクティブなモードはハイライト表示され、他方のモードと区別できなければならない(SHALL)。

#### Scenario: モードの切り替え
- **WHEN** ユーザーが`]`キーを押す
- **THEN** 表示中のモードが次のモードへ切り替わり、レイアウトが切り替え後のモードのものに変わる

#### Scenario: アクティブモードの表示
- **WHEN** いずれかのモードが表示されている
- **THEN** 現在のモード名がハイライト表示され、非アクティブなモードと視覚的に区別できる

#### Scenario: 起動時のデフォルトモード
- **WHEN** lazycurlを起動する
- **THEN** `Adhoc`モードがデフォルトで表示される

#### Scenario: フォーム編集中のモード切り替え抑制
- **WHEN** ユーザーが`Editor`パネルのフォームでcontentゾーン(Params/Headers/Auth/Body)にフォーカスしている状態で`[`または`]`キーを押す
- **THEN** モードは切り替わらず、表示中のサブセクションが切り替わる

### Requirement: Adhocモードのレイアウト
`Adhoc`モードは、リクエスト編集フォーム・Response・Historyの3ペインを表示しなければならない(SHALL)。`Adhoc`モードの利用にあたって、collectionの作成や選択を要求してはならない(SHALL NOT)。`Editor`パネルにフォーカスが移った時点で、追加のキー操作なしにリクエストのフィールドを直接編集できなければならない(SHALL)。

#### Scenario: collection無しでのリクエスト編集
- **WHEN** collectionが1つも存在しない状態で`Adhoc`モードを開く
- **THEN** リクエスト編集フォームが即座にフィールド編集可能な状態で表示される

#### Scenario: 送信結果の確認
- **WHEN** `Adhoc`モードでリクエストを送信する
- **THEN** Responseペインに結果が表示され、Historyペインにも記録される

#### Scenario: フォーカス直後の直接編集
- **WHEN** ユーザーが`tab`キーで`Editor`パネルにフォーカスを移す
- **THEN** `e`キーなどの編集モードへの遷移操作を挟まずに、Nameフィールドから直接文字を入力できる

### Requirement: collectionへの保存
ユーザーは`Adhoc`モードのリクエストを、任意のタイミングでcollectionへ保存できなければならない(SHALL)。保存時には既存collectionの選択、または新規collection作成のいずれかを選べなければならない(SHALL)。

#### Scenario: 既存collectionへの保存
- **WHEN** ユーザーが`Adhoc`モードで`ctrl+s`キーを押し、既存collectionを選択する
- **THEN** そのリクエストが選択したcollectionの`.http`ファイルに`###`ブロックとして追記される

#### Scenario: 新規collection作成による保存
- **WHEN** ユーザーが`Adhoc`モードで`ctrl+s`キーを押し、新規collection名を入力する
- **THEN** 新しいcollectionファイルが作成され、そのリクエストが最初のリクエストとして保存される

#### Scenario: 保存後のモード切り替え
- **WHEN** `Adhoc`モードのリクエストの保存が完了する
- **THEN** 自動的に`Collections`モードへ切り替わり、保存先collectionと保存したリクエストが選択された状態になる
