## Purpose

`main`とは独立した固定名の`release`ブランチと、リリースごとに使い捨てで切る`version/vX.Y.Z`ブランチによるリリース運用を担う。`version/vX.Y.Z`ブランチから`release`ブランチへの pull request がマージされたときに、マージコミットへのタグ`vX.Y.Z`の作成・push、自動生成リリースノート付きGitHub Releaseの作成、マージ済みブランチの自動削除までを機械的に行う。

## Requirements

### Requirement: 固定名releaseブランチ
システムは、`main`ブランチとは独立して存在する固定名の`release`ブランチを持ち、`release`ブランチは`version/**`ブランチのマージによってのみ更新されなければならない(SHALL)。

#### Scenario: releaseブランチが通常のpushで更新されない
- **WHEN** 開発者が`main`ブランチに新しいコミットをpushする
- **THEN** `release`ブランチのHEADは変化しない

### Requirement: versionブランチの使い捨て運用
システムは、`version/vX.Y.Z`という命名規則のブランチのみを`main`から作成することを許可し、`version/**`ブランチへの直接pushを禁止しなければならない(SHALL NOT allow direct push)。

#### Scenario: mainからのversionブランチ作成
- **WHEN** 開発者が`main`の最新コミットから`version/v0.2.0`ブランチを作成する
- **THEN** ブランチ作成は成功する

#### Scenario: versionブランチへの直接push
- **WHEN** 開発者が既存の`version/v0.2.0`ブランチに対して直接コミットをpushしようとする
- **THEN** GitHubはそのpushを拒否する

### Requirement: マージによるタグ自動作成
システムは、`version/vX.Y.Z`ブランチから`release`ブランチへのpull requestがマージされたとき、そのマージコミットに対してタグ`vX.Y.Z`(マージ元ブランチ名から`version/`プレフィックスを除いた文字列)を作成し、リモートリポジトリにpushしなければならない(SHALL)。

#### Scenario: versionブランチのマージによるタグ作成
- **WHEN** `version/v0.2.0`ブランチから`release`ブランチへのpull requestがマージされる
- **THEN** マージコミットに対して`v0.2.0`というタグが作成され、リモートリポジトリにpushされる

#### Scenario: releaseブランチ以外へのマージではタグを作成しない
- **WHEN** `version/**`ではないブランチから`release`ブランチへのpull requestがマージされる
- **THEN** タグ作成処理は実行されない

#### Scenario: releaseブランチ以外を対象とするマージではタグを作成しない
- **WHEN** `version/v0.2.0`ブランチから`release`以外のブランチ(`main`など)へのpull requestがマージされる
- **THEN** タグ作成処理は実行されない

### Requirement: リリースノート付きGitHub Release自動作成
システムは、タグ`vX.Y.Z`の作成に伴い、GitHubの自動生成リリースノート機能を用いて当該タグに対応するGitHub Releaseを作成しなければならない(SHALL)。

#### Scenario: タグ作成に伴うRelease作成
- **WHEN** `v0.2.0`タグが作成される
- **THEN** `v0.2.0`を対象とするGitHub Releaseが、前回リリース以降の変更を含む自動生成リリースノート付きで作成される

### Requirement: マージ済みversionブランチの自動削除
システムは、`version/vX.Y.Z`ブランチが`release`ブランチへマージされた後、そのブランチをリモートリポジトリから自動的に削除しなければならない(SHALL)。

#### Scenario: マージ後のブランチ削除
- **WHEN** `version/v0.2.0`ブランチから`release`ブランチへのpull requestがマージされる
- **THEN** `version/v0.2.0`ブランチはリモートリポジトリから削除される
