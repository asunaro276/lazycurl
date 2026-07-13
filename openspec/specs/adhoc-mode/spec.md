## Purpose

collectionを介さずにその場でリクエストを組み立て・送信できるAdhocモードを担う。`Adhoc`/`Collections`間のモード切り替え、Adhoc画面の3ペインレイアウト、メモリ内限定のリクエスト状態管理、collectionへの保存導線、履歴の共有を提供する。

## Requirements

### Requirement: モード切り替え
lazycurlは`Adhoc`と`Collections`の2つのモードを持ち、`[`/`]`キーで相互に切り替えられなければならない(SHALL)。現在アクティブなモードはハイライト表示され、他方のモードと区別できなければならない(SHALL)。

#### Scenario: モードの切り替え
- **WHEN** ユーザーが`]`キーを押す
- **THEN** 表示中のモードが次のモードへ切り替わり、レイアウトが切り替え後のモードのものに変わる

#### Scenario: アクティブモードの表示
- **WHEN** いずれかのモードが表示されている
- **THEN** 現在のモード名がハイライト表示され、非アクティブなモードと視覚的に区別できる

#### Scenario: 起動時のデフォルトモード
- **WHEN** lazycurlを起動する
- **THEN** `Adhoc`モードがデフォルトで表示される

### Requirement: Adhocモードのレイアウト
`Adhoc`モードは、リクエスト編集フォーム・Response・Historyの3ペインを表示しなければならない(SHALL)。`Adhoc`モードの利用にあたって、collectionの作成や選択を要求してはならない(SHALL NOT)。

#### Scenario: collection無しでのリクエスト編集
- **WHEN** collectionが1つも存在しない状態で`Adhoc`モードを開く
- **THEN** リクエスト編集フォームが即座に操作可能な状態で表示される

#### Scenario: 送信結果の確認
- **WHEN** `Adhoc`モードでリクエストを送信する
- **THEN** Responseペインに結果が表示され、Historyペインにも記録される

### Requirement: Adhocリクエストの一時性
`Adhoc`モードで組み立てられたリクエストは、保存されるまでメモリ内にのみ保持されなければならない(SHALL)。保存されていない`Adhoc`リクエストは、アプリケーション終了時に破棄されなければならない(SHALL)。`Adhoc`モードでは`{{variable}}`の展開を行ってはならない(SHALL NOT)。

#### Scenario: 未保存での終了
- **WHEN** `Adhoc`モードでリクエストを組み立てたが保存せずにlazycurlを終了する
- **THEN** 次回起動時にそのリクエストは復元されない

#### Scenario: environment変数展開の非対象
- **WHEN** `Adhoc`モードでリクエストを組み立てる
- **THEN** `{{variable}}`の展開は行われず、入力した値がそのまま送信に使われる

### Requirement: collectionへの保存
ユーザーは`Adhoc`モードのリクエストを、任意のタイミングでcollectionへ保存できなければならない(SHALL)。保存時には既存collectionの選択、または新規collection作成のいずれかを選べなければならない(SHALL)。

#### Scenario: 既存collectionへの保存
- **WHEN** ユーザーが`Adhoc`モードで`s`キーを押し、既存collectionを選択する
- **THEN** そのリクエストが選択したcollectionの`.http`ファイルに`###`ブロックとして追記される

#### Scenario: 新規collection作成による保存
- **WHEN** ユーザーが`Adhoc`モードで`s`キーを押し、新規collection名を入力する
- **THEN** 新しいcollectionファイルが作成され、そのリクエストが最初のリクエストとして保存される

#### Scenario: 保存後のモード切り替え
- **WHEN** `Adhoc`モードのリクエストの保存が完了する
- **THEN** 自動的に`Collections`モードへ切り替わり、保存先collectionと保存したリクエストが選択された状態になる

### Requirement: 履歴の共有
実行履歴は`Adhoc`モードと`Collections`モードの間で共有されなければならない(SHALL)。

#### Scenario: Adhocでの送信が共有履歴に残る
- **WHEN** `Adhoc`モードでリクエストを送信する
- **THEN** その送信結果が`Collections`モードのHistoryパネルからも参照できる
