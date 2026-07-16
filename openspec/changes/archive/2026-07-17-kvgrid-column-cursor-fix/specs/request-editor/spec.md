## MODIFIED Requirements

### Requirement: フォームベースのリクエスト編集
lazycurlはMethod/URL/Query Params/Headersをフォーム形式(キー・バリューのグリッド編集、行の追加・削除・有効/無効切り替え)で編集できなければならない(SHALL)。生の`.http`テキストを直接編集することを必須としてはならない。KVGridは非編集時、選択中の行だけでなく選択中のセル(KeyまたはValue)を視覚的に区別して表示しなければならない(SHALL)。

#### Scenario: Headerの追加
- **WHEN** ユーザーがHeadersパネルで新しい行を追加し、キーと値を入力する
- **THEN** そのHeaderがリクエストに追加され、保存時に`.http`ファイルへ反映される

#### Scenario: Headerの無効化
- **WHEN** ユーザーが既存のHeader行のチェックボックスを外す
- **THEN** そのHeaderは送信対象から除外されるが、行自体はフォーム上に残る

#### Scenario: Authタイプの選択
- **WHEN** ユーザーがAuthパネルで「Bearer Token」を選択しトークン値を入力する
- **THEN** 送信時に`Authorization: Bearer <値>`ヘッダーが自動的に付与される

#### Scenario: 既存行のKey列への移動
- **WHEN** ユーザーが既存のParam/Header行にカーソルを合わせ、選択中のセルがValue列である状態で`h`/`left`/`shift+tab`を押す
- **THEN** カーソルがKey列に移動し、その列が視覚的にハイライトされ、`enter`でKeyを編集できる

#### Scenario: 行作成直後のセル表示
- **WHEN** ユーザーが`a`で新しい行を作成しKey・Valueを入力し終える
- **THEN** カーソルはValue列に留まるが、Value列が選択中セルとして視覚的にハイライトされ、Key列にいないことが画面上で判別できる
