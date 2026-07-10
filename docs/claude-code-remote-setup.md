# lazycurl — Claude Code リモートセッション 環境構築ガイド

本ドキュメントは、`lazycurl` リポジトリを **Claude Code のリモートセッション**で開発するための環境構築手順をまとめたものです。

---

## 目次

1. [前提条件](#前提条件)
2. [リポジトリのクローン](#リポジトリのクローン)
3. [Claude Code リモートセッションの起動](#claude-code-リモートセッションの起動)
4. [プロジェクト固有の初期設定](#プロジェクト固有の初期設定)
5. [openspec のインストールと設定](#openspec-のインストールと設定)
6. [開発ワークフロー](#開発ワークフロー)
7. [プロジェクト構成の概要](#プロジェクト構成の概要)
8. [カスタムスラッシュコマンド（/opsx）](#カスタムスラッシュコマンドopsx)
9. [注意事項・トラブルシューティング](#注意事項トラブルシューティング)

---

## 前提条件

以下のツール・アカウントが必要です。

### 必須ツール

| ツール | 推奨バージョン | 確認コマンド |
|--------|--------------|------------|
| Go | 1.24.7 以上 | `go version` |
| curl | 7.70 以上 | `curl --version` |
| git | 任意 | `git --version` |
| Claude Code CLI | 最新版 | `claude --version` |
| Node.js / npm | LTS 版 | `node --version` |

> **curl のバージョンについて**: lazycurl は `-w '%{json}'` オプションを利用してレスポンスメタデータを取得します。curl 7.70 未満の場合は警告が表示され、一部機能が制限されます。

### 必要なアカウント

- **Anthropic アカウント**（Claude Code を利用するため）
- GitHub へのアクセス権（リポジトリのクローン用）

---

## リポジトリのクローン

```sh
git clone https://github.com/asunaro276/lazycurl.git
cd lazycurl
```

---

## Claude Code リモートセッションの起動

### 1. Claude Code CLI のインストール（未インストールの場合）

```sh
npm install -g @anthropic-ai/claude-code
```

### 2. 認証

```sh
claude auth login
```

ブラウザが開き、Anthropic アカウントでの認証を求められます。

### 3. リモートセッションの起動

リポジトリのルートで以下を実行します。

```sh
cd ~/sandbox/lazycurl
claude
```

> **WSL 上での起動**: Windows から WSL 環境へのリモートセッションとして利用する場合は、WSL ターミナル内で上記コマンドを実行してください。Claude Code は WSL 上の Linux 環境をネイティブに扱えます。

---

## プロジェクト固有の初期設定

### Go 依存関係のインストール

```sh
go mod download
```

全依存パッケージ（Bubble Tea、lipgloss、bubbles など）がダウンロードされます。

### ビルド確認

```sh
go build -o lazycurl ./cmd/lazycurl
./lazycurl
```

### `.claude/settings.local.json` について

`.claude/settings.local.json` はユーザーごとのローカル権限設定ファイルで、Claude Code の慣習上リポジトリにはコミットされません(このリポジトリにも含まれていません)。openspec コマンドなどの実行をセッションごとに毎回許可したくない場合は、各自で以下の内容を作成してください。

```json
{
  "permissions": {
    "allow": [
      "Bash(openspec *)"
    ]
  }
}
```

必要に応じて `WebFetch(domain:github.com)` や、ファイル整理用の `mv` / `rmdir` などの許可も追加できます。

---

## openspec のインストールと設定

`openspec` は本プロジェクトが採用している **仕様駆動開発（Spec-Driven Development）** のための CLI ツールです。Claude Code との連携でスペック・プロポーザル・タスクを自動生成する役割を担います。

### インストール

npm の `openspec` パッケージ名は無関係の別プロジェクトに占有されているため、必ずスコープ付きパッケージ名でインストールしてください。

```sh
npm install -g @fission-ai/openspec
```

> インストール後、`openspec --version` でバージョンが表示されることを確認してください。

### 動作確認

```sh
# リポジトリ内の openspec 設定を確認
openspec doctor

# 現在の変更一覧を確認
openspec list
```

### `openspec/` ディレクトリの構成

```
openspec/
├── config.yaml          # プロジェクトコンテキスト・ルール定義
├── specs/                 # 確定済みの機能仕様（spec.md）
│   ├── tui-shell/
│   ├── collection-storage/
│   ├── curl-execution/
│   ├── environment-variables/
│   └── request-editor/
└── changes/               # 進行中・完了した変更（proposal/design/tasks）
    ├── adhoc-request-mode/
    ├── stream-response-body/
    └── archive/            # アーカイブ済みの変更
```

各ディレクトリの詳細な構成はリポジトリ内の実体（`ls openspec/specs` / `ls openspec/changes`）で随時確認してください。

### `openspec/config.yaml` の概要

```yaml
schema: spec-driven
context: |
  言語：日本語
  すべての成果物（proposal, tasks, spec など）は日本語で作成
  ただし技術用語（API, REST, HTTP, TUI など）・コード・ファイルパスは英語のまま

  Tech stack:
    - 言語: Go
    - TUI: Bubble Tea + lipgloss + bubbles
rules:
  proposal:
    - 簡潔にまとめること
```

---

## 開発ワークフロー

### よく使うコマンド

```sh
# ビルド
go build -o lazycurl ./cmd/lazycurl

# 実行
./lazycurl

# テスト
go test ./...

# 特定パッケージのテスト
go test ./internal/tui/...

# フォーマット
go fmt ./...

# Lint（golangci-lint が必要）
golangci-lint run

# go.sum の更新
go mod tidy
```

### 機能追加の標準フロー（openspec 連携）

1. **アイデアの探索**（Claude Code セッション内）

   ```
   /opsx:explore
   ```

2. **変更プロポーザルの作成**

   ```
   /opsx:propose
   ```
   
   proposal.md・design.md・tasks.md が `openspec/changes/<name>/` に自動生成されます。

3. **実装**

   ```
   /opsx:apply
   ```
   
   tasks.md のタスクを順番に実装します。

4. **完了したら変更をアーカイブ**

   ```
   /opsx:archive
   ```

---

## プロジェクト構成の概要

```
lazycurl/
├── cmd/
│   └── lazycurl/
│       ├── main.go          # エントリポイント
│       ├── app.go           # アプリケーション本体
│       └── app_test.go
├── internal/
│   ├── collection/          # コレクション管理（.http ファイル）
│   ├── config/              # 設定ファイル管理
│   ├── curlexec/            # curl サブプロセス実行
│   ├── environment/         # 環境変数・変数展開
│   ├── httpfile/            # .http ファイルのパーサー
│   └── tui/                 # Bubble Tea TUI コンポーネント
│       ├── form/
│       ├── shell/
│       └── styles/
├── openspec/                # 仕様管理（openspec CLI 用）
│   ├── config.yaml
│   ├── specs/
│   └── changes/
├── .claude/
│   ├── settings.local.json  # Claude Code 権限設定（各自で作成、リポジトリには未コミット）
│   ├── commands/opsx/       # カスタムスラッシュコマンド定義
│   └── skills/              # openspec 連携スキル定義
├── go.mod
├── go.sum
└── README.md
```

---

## カスタムスラッシュコマンド（/opsx）

Claude Code セッション内で使用できるプロジェクト固有のコマンドです。

| コマンド | 説明 |
|---------|------|
| `/opsx:propose` | 新機能の変更プロポーザルを作成（proposal.md・design.md・tasks.md を自動生成） |
| `/opsx:apply` | 変更の tasks.md に基づいて実装を進める |
| `/opsx:explore` | 思考整理モード。実装はせず、問題探索・要件整理に集中 |
| `/opsx:archive` | 完了した変更をアーカイブに移動 |

> これらのコマンドはすべて `openspec CLI` が必要です。インストール後に使用してください。

---

## 注意事項・トラブルシューティング

### curl が見つからない / バージョンが古い

```sh
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y curl

# macOS
brew install curl

# バージョン確認
curl --version
```

### Go のバージョンが古い

```sh
# Go の公式サイトから最新版をインストール
# https://go.dev/dl/

# または mise / asdf などのバージョンマネージャーを使用
mise use go@1.24.7
```

### `openspec` コマンドが見つからない

```sh
# npm のグローバルパスが PATH に含まれているか確認
npm config get prefix
# 出力例: /home/user/.npm-global

# .zshrc または .bashrc に追加
export PATH="$HOME/.npm-global/bin:$PATH"
source ~/.zshrc
```

### Claude Code セッションで openspec コマンドの許可を毎回聞かれる

[`.claude/settings.local.json` について](#プロジェクト固有の初期設定)の項を参照し、`Bash(openspec *)` を許可する設定を作成してください。

### WSL 上での PATH 問題

WSL ではデフォルトで Windows 側の PATH が引き継がれます。Go や Node.js が Windows 側にインストールされている場合、WSL 内で別途インストールすることを推奨します。

```sh
# WSL 内に Go をインストール（mise 使用例）
curl https://mise.run | sh
mise use go@1.24.7 node@lts
```

### `go mod tidy` でエラーが出る

```sh
# ネットワーク経由でモジュールを取得する場合
export GOPROXY=https://proxy.golang.org,direct
go mod tidy
```

---

## 参考リンク

- [lazycurl GitHub リポジトリ](https://github.com/asunaro276/lazycurl)
- [Claude Code ドキュメント](https://docs.anthropic.com/claude-code)
- [Bubble Tea ドキュメント](https://github.com/charmbracelet/bubbletea)
- [openspec CLI](https://www.npmjs.com/package/@fission-ai/openspec)
