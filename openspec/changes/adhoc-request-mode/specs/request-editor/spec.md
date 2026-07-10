## MODIFIED Requirements

### Requirement: リクエストの新規作成・複製・削除
lazycurlはリクエストを新規作成・複製・削除できなければならない(SHALL)。新規作成は、collectionに属した状態(`Collections`モード)、またはcollectionに属さない状態(`Adhoc`モード)のいずれでも行えなければならない(SHALL)。複製・削除は既存のcollection内リクエストに対してのみ行う。

#### Scenario: コレクション内での新規リクエストの作成
- **WHEN** ユーザーが`Collections`モードのコレクション内で新規リクエスト作成を実行する
- **THEN** 空のMethod/URL/Headers/Bodyを持つ新しいリクエストがフォームで開かれ、保存するとコレクションファイルに`###`ブロックとして追記される

#### Scenario: collectionに属さない新規リクエストの作成
- **WHEN** ユーザーが`Adhoc`モードで新規リクエストを組み立てる
- **THEN** 空のMethod/URL/Headers/Bodyを持つ新しいリクエストがフォームで開かれるが、いずれのcollectionファイルにも追記されない

#### Scenario: リクエストの複製
- **WHEN** ユーザーが既存のリクエストを複製する
- **THEN** 同一内容を持つ新しいリクエストが同じコレクション内に追加される

#### Scenario: リクエストの削除
- **WHEN** ユーザーがリクエストの削除を実行し確認する
- **THEN** 対応する`###`ブロックがコレクションファイルから削除される
