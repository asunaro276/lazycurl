## MODIFIED Requirements

### Requirement: パネルベースのレイアウト
lazycurlは`Collections`モードにおいて、コレクション一覧・リクエスト一覧・レスポンスビューを常時表示するパネルレイアウトを提供しなければならない(SHALL)。

#### Scenario: Collectionsモード表示時のレイアウト
- **WHEN** ユーザーが`Collections`モードを表示する
- **THEN** コレクション一覧パネル、選択中コレクションのリクエスト一覧パネル、レスポンス表示パネルが同時に表示される
