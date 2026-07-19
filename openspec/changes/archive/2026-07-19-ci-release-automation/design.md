## Context

現状 `.github/` が存在せず、CIは無い。ローカルゲートは `go build` / `go test ./...` / `go fmt ./...` の3つ(CLAUDE.md記載)。`golangci-lint` はドキュメント(`docs/claude-code-remote-setup.md`)に言及があるが設定ファイルは無く、実体の無い記述になっている。

`git tag` は1つも存在しない。配布手段は README記載の `go install github.com/asunaro276/lazycurl/cmd/lazycurl@latest` のみで、goreleaser・Dockerfile・バージョン埋め込み(ldflags)は無い。タグが無いため、現状の `go install @latest` は mainブランチの pseudo-version を解決している。

OS依存コード(`//go:build`等)は無く、プラットフォーム分岐は存在しない。

## Goals / Non-Goals

**Goals:**
- push/PRごとに `go build` / `go test` / `gofmt` を機械的に検証する CI を導入する
- リリース版のバージョン番号を人間が決め、そのタイミングで `git tag` とリリースノート付き GitHub Release を自動生成する仕組みを作る
- リリース対象は「CIが通ったコミット」であることを GitHub 側の仕組み(branch protection)で担保する
- 誤操作(バージョンブランチへの直接コミット等)を技術的に防止する

**Non-Goals:**
- `golangci-lint` の導入(既存方針である `go vet` / `gofmt` の範囲に留める)
- goreleaser等によるクロスコンパイル済みバイナリの配布・Homebrew tap整備
- ホットフィックス専用ワークフロー(パッチリリースも「mainから切り直す」通常フローに乗せる)
- semantic-release/release-please等によるバージョン番号の自動算出(バージョン番号は人間が決める)

## Decisions

### 1. リリース検知は `release` への PR マージ、push/create イベントではない

`version/vX.Y.Z` ブランチを作った瞬間(`create`イベント)や、そこへのpush(`push`イベント)ではなく、`release`ブランチへの **PRがマージされたタイミング**(`pull_request` イベント、`types: [closed]` かつ `merged == true`)をトリガーにする。

理由: `release`ブランチに必須ステータスチェック(CI)を設定しておけば、「CIが通っていないとマージできない」をGitHub自身に強制させられる。リリースワークフロー側で「CIの結果を待つ」ロジックを自前で書く必要が無くなる。

代替案として検討したもの:
- `create`イベント(ブランチ作成時) → 一度しか発火せず、CI結果を待てない
- `push`イベント(version/**への全push) → 毎push発火するため「タグが既に存在するなら何もしない」冪等性ガードが別途必要になり、CIグリーンの保証も自前で持つ必要がある

### 2. 固定名の `release` ブランチ(`latest`ではない)

リリース時のみ進む専用ブランチを `release` と命名する(`main`とは別に常設)。`version/vX.Y.Z`は使い捨てで、この`release`ブランチにPRでマージされる。

`latest`という名前も検討したが、`go install ...@latest`(Go modulesの「最新公開バージョン」を指す用語)やGitHubのReleaseページの「Latest」ラベルと用語が衝突し紛らわしいため不採用。機能的な衝突は無いが、ドキュメント上の混乱を避けるため`release`を採用。

固定名にしたことで、バージョン番号は `version/vX.Y.Z` というブランチ名から抽出できる(後述の決定3)。

### 3. バージョン番号はブランチ名から抽出する(VERSIONファイルは導入しない)

PRの `head.ref`(例: `version/v0.2.0`)から `version/` プレフィックスを除去してタグ名 `v0.2.0` を得る。

検討の過程で「`release`ブランチが`main`と同一コミットだと、そこへのPRが空(差分ゼロ)になりマージできないのでは」という懸念があったが、`release`ブランチは**リリース時にしか進まない**設計にしたため、`version/vX.Y.Z`(`main`から切った時点の状態)は常に「前回リリース以降に`main`へ積まれた全コミット」を差分として持つ。よって`release`ブランチが`main`から独立して存在する限り、`version/vX.Y.Z → release`のPRが空になることは無い。この性質により、バージョン番号確定用のVERSIONファイルを別途導入する必要が無くなった。

### 4. `version/**` ブランチへの直接コミットを branch protection で禁止

`version/vX.Y.Z`は`main`から切ることのみを許可し、ブランチ上への直接pushを禁止する(PR経由のマージのみ許可、かつそのPRの変更元は事実上存在しない=ブランチ自体は`main`のスナップショットのまま`release`へのPRを開く運用)。リリース準備中に不具合が見つかった場合は、`main`にコミットしてから`version`ブランチを切り直す(または既存のversionブランチを削除して作り直す)。

理由: `version`ブランチ上で直接修正すると、その修正が`main`に反映されないまま`release`にだけ取り込まれ、`main`と`release`の間に本来存在しないはずの差分(mainに存在しないコミット)が生まれてしまう。修正は必ず`main`を経由させることで、`release`ブランチの内容が常に「`main`のある時点のスナップショット」であることを保証する。

### 5. リリースノートは GitHub の自動生成機能を使う

`release`ブランチ上のタグ履歴を使って、GitHub Releases の自動生成ノート機能(前回リリースからの変更をマージ済みPRベースで列挙)を利用する。Conventional Commits等の規約は導入しない(既存コミット履歴が自由形式のため)。

### 6. CIチェック範囲は既存ローカルゲートと同一

`go build ./...` / `go test ./...` / `gofmt -l`(差分ゼロ確認)の3点のみ。`go vet`は既存のCLAUDE.mdの記述(「go vet/gofmtの範囲を超えるlintは想定しない」)に照らして「gofmtと同格の最低限のチェック」として含めるかは実装時に判断するが、`golangci-lint`は明確に対象外とする。

## Risks / Trade-offs

- [Risk] `version/**`へのpush制限をbranch protectionで設定し忘れると、直接コミットの禁止が口約束だけの運用ルールに後退してしまう → Mitigation: tasksにリポジトリ設定(branch protection rule作成)を明示的な実装項目として含める
- [Risk] 初回リリース時は`release`ブランチ自体が存在しない(このchangeで新規作成する)。GitHub Releasesの自動生成ノートは「前回のリリースタグ」を参照するため、初回だけは比較対象が無く、全コミット履歴が列挙される可能性がある → Mitigation: 初回リリースのみノート内容を手動確認する運用でよく、自動化の対象外として許容する
- [Risk] `release`ブランチの必須ステータスチェックが正しくCIワークフローのjob名と一致していないと、チェックが要求されず素通りする(GitHub Branch Protectionの既知の設定ミスパターン) → Mitigation: tasksでCIワークフローの導入 → branch protection設定の順で行い、実際にPRを作って必須チェックが機能することを確認する
- [Trade-off] バージョン番号の自動算出(release-please等)を採用しないため、パッチ/マイナー/メジャーの判断は引き続き人間が行う。将来コミット量が増えた場合は再検討の余地がある
- [Trade-off] バイナリ配布(goreleaser)を対象外としたため、`go install`できない非Go開発者への配布は今回解決しない。タグさえ切れていれば将来goreleaserをタグpushフックとして追加するのは容易

## Migration Plan

1. CIワークフロー(`go build` / `go test` / `gofmt -l`)を追加し、`main`へのPRで動作確認する
2. `main`から`release`ブランチを作成する(初期状態は`main`と同一)
3. `release`ブランチのbranch protectionを設定する(CIワークフローを必須ステータスチェックに指定)
4. `version/**`パターンのbranch protectionを設定する(直接push禁止、PRのみ許可)
5. リリースタグ付けワークフロー(`pull_request closed & merged, base=release, head=version/**`をトリガーにタグ作成・GitHub Release作成)を追加する
6. マージ後のヘッドブランチ自動削除をリポジトリ設定で有効化する
7. 実際に `version/v0.1.0` のような初回リリース用ブランチを切ってPRを作成し、一連の流れ(CI必須チェック→マージ→タグ→Release作成→ブランチ自動削除)を通しで確認する

ロールバック: ワークフローファイルとbranch protection設定を削除するのみで、既存コードには影響しないため容易に取り消せる。

## Open Questions

- `go vet`をCIの必須チェックに含めるかどうかは実装時に確定する(現状のCLAUDE.mdの記述からは含めても矛盾しないと考えられるが、明示的な合意はこのdesignの時点では取れていない)
