## ADDED Requirements

### Requirement: フォームベースのリクエスト編集
lazycurlはMethod/URL/Query Params/Headers/Authをフォーム形式(キー・バリューのグリッド編集、行の追加・削除・有効/無効切り替え)で編集できなければならない(SHALL)。生の`.http`テキストを直接編集することを必須としてはならない。

#### Scenario: Headerの追加
- **WHEN** ユーザーがHeadersパネルで新しい行を追加し、キーと値を入力する
- **THEN** そのHeaderがリクエストに追加され、保存時に`.http`ファイルへ反映される

#### Scenario: Headerの無効化
- **WHEN** ユーザーが既存のHeader行のチェックボックスを外す
- **THEN** そのHeaderは送信対象から除外されるが、行自体はフォーム上に残る

#### Scenario: Authタイプの選択
- **WHEN** ユーザーがAuthパネルで「Bearer Token」を選択しトークン値を入力する
- **THEN** 送信時に`Authorization: Bearer <値>`ヘッダーが自動的に付与される

### Requirement: Body編集
Bodyはテキストエリアで編集できなければならない(SHALL)。また、ユーザーは`$EDITOR`環境変数で指定された外部エディタにBody編集を委譲できなければならない(SHALL)。

#### Scenario: テキストエリアでのBody編集
- **WHEN** ユーザーがBodyパネルにJSONテキストを入力する
- **THEN** 入力内容がリクエストのBodyとして保持される

#### Scenario: 外部エディタへの委譲
- **WHEN** ユーザーがBody編集中に`ctrl-e`を押す
- **THEN** 現在のBody内容が一時ファイルに書き出され、`$EDITOR`が起動する。エディタ終了後、ファイルの内容がBodyとして再読み込みされる

### Requirement: リクエストの新規作成・複製・削除
lazycurlはコレクション内でリクエストを新規作成・複製・削除できなければならない(SHALL)。

#### Scenario: 新規リクエストの作成
- **WHEN** ユーザーがコレクション内で新規リクエスト作成を実行する
- **THEN** 空のMethod/URL/Headers/Bodyを持つ新しいリクエストがフォームで開かれ、保存するとコレクションファイルに`###`ブロックとして追記される

#### Scenario: リクエストの複製
- **WHEN** ユーザーが既存のリクエストを複製する
- **THEN** 同一内容を持つ新しいリクエストが同じコレクション内に追加される

#### Scenario: リクエストの削除
- **WHEN** ユーザーがリクエストの削除を実行し確認する
- **THEN** 対応する`###`ブロックがコレクションファイルから削除される
