## Why

現在の `README.md` は全文日本語で書かれており、`LICENSE` ファイルも存在しない。lazycurlはGitHub上で公開されているOSSプロジェクトだが、この2点は「英語圏の開発者が最初に見て判断材料にする情報」が欠けている状態であり、OSSとしての第一印象・発見性・利用のしやすさを損なっている。README上に視覚的なブランド要素（ロゴ）もなく、`lazygit`/`lazydocker`/`lazysql` にインスパイアされた同系統のツールと並んだときの見た目のまとまりにも欠ける。

## What Changes

- `README.md` を全文英訳し、日本語版は残さず完全に置き換える（`README.ja.md` は作成しない）
- `docs/claude-code-remote-setup.md` を全文英訳し、同様に完全に置き換える
- `README.md` 冒頭に `figlet -f standard lazycurl` で生成したASCIIワードマークをコードブロックとして追加する
- リポジトリルートに `LICENSE` ファイル（MIT License）を新規追加する
- `README.md` に既存の慣習（GitHubのOSSリポジトリ標準）に倣い、ライセンスへの言及セクションを追加する

## Capabilities

### New Capabilities

- `project-documentation`: リポジトリのトップレベルドキュメント（README、ライセンス）がOSSとして満たすべき内容・言語・構成を定義する（lazycurl本体の実行時の振る舞いは対象外）

### Modified Capabilities

（なし。既存spec配下（`adhoc-mode`, `streaming-response`, `environment-variables`, `curl-execution`, `collection-storage`, `request-editor`, `tui-shell`）のいずれの要件にも変更はない）

## Impact

- 変更対象ファイル: `README.md`（全文置き換え）, `docs/claude-code-remote-setup.md`（全文置き換え）, `LICENSE`（新規追加）
- コード（`cmd/`, `internal/`）・テスト・ビルドには一切影響しない
- 既存の日本語ドキュメントを参照している読者・コントリビューターへの影響: README/docsの一次言語が日本語から英語に変わる（日本語版は提供しない）
