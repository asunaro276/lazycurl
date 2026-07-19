## 1. CIワークフロー

- [ ] 1.1 `.github/workflows/ci.yml` を作成し、`main`へのpushおよび任意ブランチからのpull requestをトリガーに `go build ./...` を実行するジョブを定義する
- [ ] 1.2 同ワークフローに `go test ./...` を実行するジョブ(または既存ジョブ内のステップ)を追加する
- [ ] 1.3 同ワークフローに `gofmt -l` の出力が空であることを検証するジョブ(または既存ジョブ内のステップ)を追加する。差分が存在する場合はジョブを失敗させる
- [ ] 1.4 `main`向けのテストPRを作成し、3つのチェックがそれぞれ正しく成功/失敗を報告することを確認する

## 2. releaseブランチとリポジトリ設定

- [ ] 2.1 現在の`main`から`release`ブランチを作成し、リモートにpushする
- [ ] 2.2 `release`ブランチへのpull requestに対し、1章のCIジョブ(build/test/gofmt)を必須ステータスチェックとして設定する(branch protection rule)
- [ ] 2.3 `version/**`パターンのブランチに対し、直接pushを禁止しpull request経由のマージのみを許可するbranch protection ruleを設定する
- [ ] 2.4 リポジトリ設定で「マージ後にヘッドブランチを自動削除」を有効化する

## 3. リリースタグ付けワークフロー

- [ ] 3.1 `.github/workflows/release.yml` を作成し、`pull_request`イベントの`closed`タイプをトリガーに設定する
- [ ] 3.2 ジョブの実行条件を `github.event.pull_request.merged == true` かつ base ブランチが`release`かつ head ブランチが`version/`で始まる、に限定する
- [ ] 3.3 head ブランチ名から`version/`プレフィックスを除去してタグ名(`vX.Y.Z`)を導出するステップを実装する
- [ ] 3.4 導出したタグ名で `pull_request.merge_commit_sha` に対して git タグを作成し、リモートにpushするステップを実装する(ワークフローに`contents: write`権限を付与する)
- [ ] 3.5 作成したタグを対象に、GitHubの自動生成リリースノート機能を用いてGitHub Releaseを作成するステップを実装する

## 4. 動作確認

- [ ] 4.1 `main`から`version/v0.1.0`ブランチを作成し、`release`ブランチへのpull requestを作成する
- [ ] 4.2 必須ステータスチェックが未完了の間はマージ操作がGitHub上で無効化されていることを確認する
- [ ] 4.3 CIが成功した状態でpull requestをマージし、`v0.1.0`タグが作成・pushされることを確認する
- [ ] 4.4 `v0.1.0`タグに対応するGitHub Releaseがリリースノート付きで作成されることを確認する
- [ ] 4.5 マージ済みの`version/v0.1.0`ブランチが自動削除されていることを確認する
- [ ] 4.6 `version/v0.1.0`ブランチへの直接push(動作確認用の一時コミット)がbranch protectionにより拒否されることを確認する

## 5. ドキュメント更新

- [ ] 5.1 `CLAUDE.md`のCommandsセクションに、CIが存在すること・リリース手順(`version/vX.Y.Z`ブランチの切り方から`release`へのマージまで)の概要を追記する
- [ ] 5.2 `docs/claude-code-remote-setup.md`の`golangci-lint`への言及を、実体(未導入・`go vet`/`gofmt`のみが対象)に合わせて修正する
