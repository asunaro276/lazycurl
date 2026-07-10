## Why

Postman/Insomnia/ApidogのようなAPIクライアントはGUI・クラウド前提で重厚だが、ターミナルで完結する軽量なHTTPクライアントには決定版がない。curl単体は強力だが、リクエストの再利用・環境変数切り替え・実行履歴の閲覧が手打ちでは煩わしい。lazygit/lazydocker/lazysqlが確立した「キーボード駆動TUIで対象ドメインを歩き回る」体験を、HTTPリクエストの実行に持ち込む。

## What Changes

- 新規Goプロジェクトとして `lazycurl` を作成する(Bubble Tea製TUI、単一バイナリ)
- リクエストコレクションを `.http` 形式(+ curl固有オプション用の軽量プラグマ拡張)で保存する
- コレクションは `~/.config/lazycurl/` 配下にグローバルに保存する(プロジェクトローカルではない)
- リクエストの実行は自前のHTTPクライアントではなく、`curl` バイナリをサブプロセスとして呼び出す方式にする
- リクエスト編集はハイブリッドUI(Method/URL/Params/Headers/Authはフォーム、Bodyはテキストエリア、`ctrl-e`で`$EDITOR`に脱出可能)
- `{{variable}}` 展開のための environment ファイル(dev/staging/prodなど)を管理する
- キーバインドはlazygit系(`hjkl`移動、`tab`パネル切替、数字キーでパネルジャンプ、`?`ヘルプ)との一貫性を最優先する
- 事前/事後スクリプトやレスポンスチェイニングはMVPでは対応しない(単純な変数展開のみ)

MVPでは以下は対象外とする(将来のchangeで扱う):
- JSON折り畳みツリー、レスポンス差分表示、グローバルファジー検索
- 外部変更ハイライト、変数のインライン解決プレビュー
- MCPサーバー化・AIヘッドレス操作、CI/バッチ実行(アサーション付き)

## Capabilities

### New Capabilities
- `collection-storage`: `.http`+プラグマ形式でのリクエストコレクションの永続化。`~/.config/lazycurl/`配下のディレクトリ構造、1コレクション=1`.http`ファイル、`###`区切りでの複数リクエスト管理
- `request-editor`: リクエストを作成・編集するハイブリッドフォームUI(Method/URL/Params/Headers/Authはフォーム、Bodyはテキストエリア+外部エディタ連携)
- `curl-execution`: `.http`リクエストを`curl`サブプロセス実行に変換し、レスポンス(ヘッダー/body/タイミング)を構造化して受け取る
- `environment-variables`: environmentファイルによる`{{variable}}`展開、アクティブ環境の切り替え
- `tui-shell`: パネルレイアウト(コレクション/リクエスト一覧/レスポンスビュー/履歴)、lazygit系キーバインド、レスポンスの基本表示(色分けバッジ含む)

### Modified Capabilities
(なし。新規プロジェクトのため既存specへの変更はない)

## Impact

- 新規Goモジュール一式(cmd/内部パッケージ、Bubble Tea依存)
- 実行時に外部コマンド`curl`が必要(バージョン7.70以降を推奨、`%{json}` write-out対応のため)
- ユーザーのホームディレクトリ配下(`~/.config/lazycurl/`)に新規ディレクトリ・ファイルを作成する
- 既存のPostman/Insomnia/`.http`資産との直接連携(インポート等)はMVPスコープ外
