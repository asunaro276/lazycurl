## MODIFIED Requirements

### Requirement: フォームベースのリクエスト編集
lazycurlはMethod/URL/Query Params/Headers/Authをフォーム形式(キー・バリューのグリッド編集、行の追加・削除・有効/無効切り替え)で編集できなければならない(SHALL)。生の`.http`テキストを直接編集することを必須としてはならない。リクエストの編集は、独立した全画面のフォームへ遷移するのではなく、`Requests`パネル(Collectionsモード)または`Editor`パネル(Adhocモード)の中でインラインに行われなければならない(SHALL)。

#### Scenario: Headerの追加
- **WHEN** ユーザーがHeadersパネルで新しい行を追加し、キーと値を入力する
- **THEN** そのHeaderがリクエストに追加され、保存時に`.http`ファイルへ反映される

#### Scenario: Headerの無効化
- **WHEN** ユーザーが既存のHeader行のチェックボックスを外す
- **THEN** そのHeaderは送信対象から除外されるが、行自体はフォーム上に残る

#### Scenario: Authタイプの選択
- **WHEN** ユーザーがAuthパネルで「Bearer Token」を選択しトークン値を入力する
- **THEN** 送信時に`Authorization: Bearer <値>`ヘッダーが自動的に付与される

#### Scenario: パネルフォーカスによる編集開始
- **WHEN** ユーザーがAdhocモードで`Editor`パネルにフォーカスする、またはCollectionsモードの`Requests`パネルでリストゾーンから`tab`を押してフォームゾーンへ進む
- **THEN** 編集モードへ遷移するための追加のキー操作を挟まずに、フォームの現在のフィールドへ直接文字を入力できる

#### Scenario: 編集中の他パネルの表示継続
- **WHEN** ユーザーがリクエストのフォームゾーンで編集している
- **THEN** モードタブ・`Collections`パネル(該当する場合)・`Response`・`History`・ステータスバーは非表示にならず表示され続ける

#### Scenario: サブセクションの切り替え
- **WHEN** ユーザーがフォームのcontentゾーン(Params/Headers/Auth/Body)で`[`または`]`キーを押す
- **THEN** 表示中のサブセクションが前後に切り替わる

#### Scenario: フォームゾーンでの送信
- **WHEN** ユーザーがフォームゾーンにフォーカスがある状態で`ctrl+r`キーを押す
- **THEN** 現在編集中のリクエストが送信される。`enter`キーはフィールドへの入力(Bodyでは改行)に使われるため送信には使われない

## ADDED Requirements

### Requirement: 編集内容のメモリ即時反映と明示的なディスク保存
フォームで入力された内容は、キー入力のたびにメモリ上のリクエストへ即座に反映されなければならない(SHALL)。`.http`ファイルへの書き込みは`ctrl+s`によって明示的に行われなければならない(SHALL)。編集内容を破棄してフォーム表示前の状態に戻す操作は提供しない(SHALL NOT)。

#### Scenario: 入力の即時反映
- **WHEN** ユーザーがCollectionsモードの`Requests`パネルで選択中リクエストのURLを編集する
- **THEN** 同じパネルのリスト表示(サマリー行のMethod/Name)にも変更が即座に反映される

#### Scenario: 明示的なディスク保存
- **WHEN** ユーザーがフォームゾーンで`ctrl+s`を押す
- **THEN** Collectionsモードでは選択中コレクションの`.http`ファイルへ現在のリクエスト一覧が書き込まれ、Adhocモードでは保存先コレクションを選択するオーバーレイが表示される
