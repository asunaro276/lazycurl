## Context

`README.md` と `docs/claude-code-remote-setup.md` は現在すべて日本語で書かれている。リポジトリには `LICENSE` ファイルが存在しない。lazycurlはGitHub上でパブリックに公開されているOSSツールであり、README・ライセンスは英語圏を含む潜在的なユーザー・コントリビューターが最初に接する情報である。

本変更は探索的な会話（openspec-explore）を経て、以下の点がすでに決定事項として固まっている:

- ロゴはASCIIアートのワードマークとし、`figlet -f standard lazycurl` の出力をそのまま採用する
- 英語化の対象は `README.md` と `docs/claude-code-remote-setup.md` の2ファイル。日本語版（`README.ja.md` 等）は残さず完全に置き換える
- ライセンスはMIT Licenseを新規追加する

このためdesignフェーズで残っている論点は多くないが、実装順序に影響する決定（ライセンスの著作権表記、READMEの構成）をここに記録する。

## Goals / Non-Goals

**Goals:**
- `README.md` を英語のみの内容に置き換え、冒頭にASCIIロゴを追加する
- `docs/claude-code-remote-setup.md` を英語のみの内容に置き換える
- `LICENSE`（MIT）をリポジトリルートに追加し、READMEからも参照する

**Non-Goals:**
- README・docs以外のドキュメント（`CLAUDE.md`、`openspec/` 配下の仕様書）の英語化は対象外（`CLAUDE.md` は既に英語で書かれている）
- デモGIF・スクリーンショットなど新しい視覚素材の追加は対象外
- READMEの構成自体を大幅に再設計すること（Features/Comparisonなど新規セクションの追加）は対象外。既存の章立て（インストール/使い方/保存形式/environment）を維持したまま英訳する
- lazycurl本体のコード・振る舞いへの変更は一切含まない

## Decisions

- **ロゴ形式: ASCIIワードマーク（`standard`フォント）をREADME内のコードブロックとして埋め込む**
  - 代替案として検討したSVG画像アセット（`docs/logo.svg`）は、TUIツールという性質上ミスマッチであり、かつ実装（このセッション）に画像デザインを反復するツールがないため採用しない
  - ASCII方式はアセットファイル不要・GitHub上でもローカルの`cat README.md`でも同一の見た目で表示できる利点がある
- **日本語版READMEは残さず完全に置き換える**
  - `README.ja.md` への退避は行わない。ユーザーの明示的な指示（「完全に置き換え」）に基づく
- **ライセンス: MIT License**
  - curlをサブプロセスとして呼び出すだけの軽量なOSSツールという性質上、利用・改変・再配布に対する制約が最小のMITが妥当と判断
  - 著作権表記の年・名義はリポジトリのGitHub Ownerに合わせる（`Copyright (c) 2026 asunaro276`）
- **`docs/claude-code-remote-setup.md` も同一変更に含める**
  - README以外で日本語のまま残っている唯一のドキュメントであり、スコープを分けても得るものがないため一括で対応する

## Risks / Trade-offs

- [日本語版を残さないことで、既存の日本語話者コントリビューターが読めなくなる] → プロジェクトの主要な対象読者を英語圏に広げる意図的なトレードオフとして許容する（ユーザー承認済み）
- [ASCIIロゴは`figlet`という外部コマンドで生成するが、生成物自体はテキストとしてREADMEに埋め込むため、リポジトリのビルド・実行時に`figlet`への依存は発生しない] → 影響なし
- [MITライセンスの選定は法的な最終判断ではない] → 必要であれば別途ライセンスの妥当性確認を促す
