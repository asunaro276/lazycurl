# lazycurl

lazygit/lazydocker/lazysql にインスパイアされた、ターミナルで完結するHTTPクライアント。`curl` をサブプロセスとして呼び出し、結果をBubble TeaベースのTUIで確認する。

## 環境構築が必要

新しいセッション・コンテナでは、以下が未セットアップの可能性があります。作業前に確認してください。

- Go の依存関係: `go mod download`
- `openspec` CLI(このプロジェクトの仕様駆動開発ワークフロー `/opsx:*` に必須): 未インストールの場合は
  ```sh
  npm install -g @fission-ai/openspec
  ```
  で導入する。**`npm install -g openspec`(スコープなし)は無関係の別パッケージなので使わないこと。**
- `curl`(7.70以上推奨。`-w '%{json}'` によるレスポンスメタデータ取得に必要)

詳細な手順・トラブルシューティングは [`docs/claude-code-remote-setup.md`](docs/claude-code-remote-setup.md) を参照。

## 開発コマンド

```sh
go build -o lazycurl ./cmd/lazycurl   # ビルド
go test ./...                          # テスト
go fmt ./...                           # フォーマット
```

## 仕様駆動開発(openspec)

このリポジトリは `openspec/` 配下で仕様(`specs/`)と変更提案(`changes/`)を管理する。新機能追加は `/opsx:propose` → `/opsx:apply` → `/opsx:archive` の流れで進める。詳細は `openspec/config.yaml` と `docs/claude-code-remote-setup.md` を参照。
