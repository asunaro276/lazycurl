## Why

このリポジトリには `.github/` が存在せず、CIが一切無い。ゲートは `go build` / `go test ./...` / `go fmt ./...` をローカルで手動実行するのみで、push/PR時に機械的に検証する仕組みが無い。また `git tag` が1つも無く、リリースの切り方(バージョン番号の確定、タグ付け、リリースノート作成)も決まっていない。`go install github.com/asunaro276/lazycurl/cmd/lazycurl@latest` が現状唯一の配布手段だが、タグが無いため pseudo-version しか解決できない。

継続的な検証(CI)と、リリース時にタグとリリースノートを機械的に作る仕組み(リリースタグ自動化)を導入する。

## What Changes

- GitHub Actions ワークフローを新設し、push/PR時に `go build ./...` / `go test ./...` / `gofmt -l` の差分ゼロを検証する
- 固定名の `release` ブランチを新設する(リリース時のみ進む、通常開発には使わない)
- リリースの都度 `main` から `version/vX.Y.Z` という使い捨てブランチを切る運用を導入する
  - `version/**` ブランチへの直接pushはbranch protectionで禁止し、`main`からの切り直し以外の更新経路を塞ぐ
- `version/vX.Y.Z` → `release` への PR をトリガーに、マージ後自動でタグ(`vX.Y.Z`)を作成・pushし、GitHub Releaseをリリースノート自動生成付きで作成するワークフローを新設する
  - `release` ブランチには CIの必須ステータスチェックを設定し、CIが通っていないとマージできないようにする
  - マージ済みの `version/vX.Y.Z` ブランチは自動削除する

このchangeでは扱わないこと(スコープ外):
- golangci-lint の導入(既存方針どおり `go vet` / `gofmt` のみ)
- goreleaser等によるクロスコンパイル済みバイナリの配布

## Capabilities

### New Capabilities
- `continuous-integration`: push/PRごとに build・test・フォーマットチェックを自動実行する仕組み
- `release-tagging`: `version/vX.Y.Z` ブランチを `release` ブランチにマージすることでタグ付けとGitHub Release作成を自動化する仕組み

### Modified Capabilities
(既存spec無し。上記2つはいずれも新規capability)

## Impact

- 新規: `.github/workflows/ci.yml`(build/test/gofmtチェック)
- 新規: `.github/workflows/release.yml`(`version/**` → `release` マージ検知、タグ作成、GitHub Release作成)
- 新規: `release` ブランチ(GitHub上のブランチ保護設定を含む)
- GitHub リポジトリ設定: `release` ブランチの必須ステータスチェック、`version/**` へのpush制限、マージ後のヘッドブランチ自動削除設定
- 既存コードへの変更なし(ワークフロー/リポジトリ設定のみ)
