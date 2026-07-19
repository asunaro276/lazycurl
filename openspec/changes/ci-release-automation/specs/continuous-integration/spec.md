## ADDED Requirements

### Requirement: push/PR時のビルド検証
システムは、任意ブランチへの push または pull request 作成・更新のたびに `go build ./...` を実行し、ビルドが失敗した場合はチェックを失敗として報告しなければならない(SHALL)。

#### Scenario: ビルドが通るコミット
- **WHEN** ビルドエラーの無いコミットを含む pull request が作成・更新される
- **THEN** ビルド検証ジョブは成功として報告される

#### Scenario: ビルドが失敗するコミット
- **WHEN** コンパイルエラーを含むコミットを含む pull request が作成・更新される
- **THEN** ビルド検証ジョブは失敗として報告される

### Requirement: push/PR時のテスト検証
システムは、任意ブランチへの push または pull request 作成・更新のたびに `go test ./...` を実行し、失敗したテストがある場合はチェックを失敗として報告しなければならない(SHALL)。

#### Scenario: 全テストが通るコミット
- **WHEN** 全パッケージのテストが成功するコミットを含む pull request が作成・更新される
- **THEN** テスト検証ジョブは成功として報告される

#### Scenario: テストが失敗するコミット
- **WHEN** 失敗するテストを含むコミットを含む pull request が作成・更新される
- **THEN** テスト検証ジョブは失敗として報告される

### Requirement: push/PR時のフォーマット検証
システムは、任意ブランチへの push または pull request 作成・更新のたびに `gofmt` による差分の有無を検証し、`gofmt`未適用のファイルが存在する場合はチェックを失敗として報告しなければならない(SHALL)。

#### Scenario: フォーマット済みのコミット
- **WHEN** 全ファイルが `gofmt` 適用済みの状態のコミットを含む pull request が作成・更新される
- **THEN** フォーマット検証ジョブは成功として報告される

#### Scenario: フォーマット未適用のコミット
- **WHEN** `gofmt` 未適用のファイルを含むコミットを含む pull request が作成・更新される
- **THEN** フォーマット検証ジョブは失敗として報告される

### Requirement: releaseブランチへのマージ前提としての必須チェック
システムは、`release`ブランチを対象とする pull request について、ビルド・テスト・フォーマットの各検証ジョブが成功していない限りマージを許可してはならない(SHALL NOT)。

#### Scenario: 必須チェックが未完了の状態でのマージ試行
- **WHEN** `version/**`ブランチから`release`ブランチへの pull request の必須チェックがまだ成功していない
- **THEN** GitHubはその pull request のマージ操作を拒否する

#### Scenario: 必須チェックが全て成功した状態でのマージ
- **WHEN** `version/**`ブランチから`release`ブランチへの pull request の全ての必須チェックが成功している
- **THEN** GitHubはその pull request のマージ操作を許可する
