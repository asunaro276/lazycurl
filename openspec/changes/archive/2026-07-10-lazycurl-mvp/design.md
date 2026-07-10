## Context

lazycurlは、lazygitと同じ「本家CLIツール(curl)の上に乗るビュワー」という設計思想を採る。Goのnet/httpで自前のHTTPクライアントを実装するのではなく、`curl`バイナリをサブプロセスとして実行し、その出力を解釈してTUIに表示する。保存形式は`.http`(IntelliJ HTTP Client/VSCode REST Client/httpYacと同系の業界標準記法)を採用し、curl固有のオプション(TLS検証スキップ、タイムアウトなど)は軽量なプラグマコメントで拡張する。

コレクションはプロジェクトローカルではなく`~/.config/lazycurl/`にグローバル保存する(Postman/Insomniaに近い体験)。

## Goals / Non-Goals

**Goals:**
- `.http`+プラグマ形式を読み書きできるパーサ/シリアライザ
- curlサブプロセス実行によるリクエスト送信と、構造化されたレスポンス(status/headers/body/timing)の取得
- Method/URL/Params/Headers/Authをフォームで編集し、Bodyはテキストエリア(+外部`$EDITOR`連携)で編集するハイブリッドUI
- environmentファイルによる`{{variable}}`展開
- lazygit系のキーバインド・パネルレイアウトを持つBubble Tea製TUI

**Non-Goals:**
- 事前/事後スクリプト、レスポンスチェイニング(`{{prev.response.body}}`等)
- JSON折り畳みツリー、レスポンス差分表示、グローバルファジー検索、外部変更ハイライト、変数のインライン解決プレビュー
- MCPサーバー化・AIヘッドレス操作
- CI/バッチ実行(アサーション付きテストランナー)
- プロジェクトローカルでの既存`.http`資産の自動読み込み(将来のimport機能候補ではあるが、MVPはlazycurlが所有するコレクションのみを扱う)

## Decisions

### D1: 保存形式は`.http` + プラグマコメント拡張

`.http`のヘッダー行→空行→bodyという構造は、curlの`-H`/`-d`引数へほぼそのまま変換できるため、実行エンジンとの相性がよい。curl固有のオプション(HTTPリクエストそのものではない振る舞い)は、リクエスト直前の`#`コメント行にプラグマとして記述する。

```
### Get user (self-signed dev server)
# @insecure
# @timeout 5s
GET {{host}}/users/{{id}}
Authorization: Bearer {{token}}
```

対応プラグマ(MVP):`@insecure`(`-k`)、`@timeout <duration>`(`--max-time`)、`@no-redirect`(デフォルトで`-L`を付与しない指定)。それ以外のプラグマ行はパーサが無視し前方互換を保つ。

代替案として検討したが不採用:
- 独自バイナリ/JSON形式で所有 → 可搬性・既存エコシステムとの互換性を失う
- プラグマなしで完全に素の`.http`のみ → Auth種別(Bearer/Basic以外)やTLS設定など、フォームUIが必要とする設定の置き場所がなくなる

### D2: 実行エンジンはcurlサブプロセス

`os/exec.CommandContext`で`curl`を起動する。入出力は3チャンネルに分離する。

- レスポンスbody → `-o <tmpfile>`
- レスポンスヘッダー → `-D <tmpfile>`
- タイミング/ステータスのメタデータ → `-w '%{json}'`(curl 7.70+)を標準出力から取得

リクエストbodyは常に一時ファイルへ書き出し、`--data-binary @<tmpfile>`で渡す(argv長制限の回避、エスケープ事故の防止)。`ctx`のキャンセルはそのままサブプロセスへのシグナル送出になるため、TUI側の「送信中リクエストを中断」がそのまま実装できる。

exit code(6=名前解決失敗、7=接続失敗、28=タイムアウト、35=TLSエラー等)を人間向けメッセージにマッピングして表示する。

「レスポンスをyankしてcurlコマンドを得る」機能は、実行のために組み立てたargvをシェルクォートして表示するだけで実現でき、実行パスと共通化できる。

代替案: net/http実装 → 構造化データの取得は楽だが、プロキシ/`.netrc`/クライアント証明書/HTTP2等curlが持つ実績のある挙動を再実装する必要があり不採用。

### D3: 保存場所はホームディレクトリ配下グローバル

`~/.config/lazycurl/`(XDG準拠)に以下の構造で保存する。

```
~/.config/lazycurl/
├── config.yaml                  # アクティブworkspace、直近使用状態など
└── collections/
    └── <collection-name>.http   # 1コレクション = 1ファイル、###区切りで複数リクエスト
        env/
        ├── dev.env.json
        ├── staging.env.json
        └── prod.env.json
```

MVPでは1コレクション=1`.http`ファイルとし、フォルダ階層によるネストは行わない(シンプルさ優先)。

### D4: 編集はハイブリッドUI

Method/URL/Params/Headers/Authはキー・バリューのグリッド編集(行の追加/削除/有効・無効チェックボックス)。Bodyのみテキストエリア(`bubbles/textarea`)とし、`ctrl-e`で内容を一時ファイルに書き出して`$EDITOR`を起動、保存後に再読み込みする。フォームの内容は保存時に`.http`+プラグマとしてシリアライズし直す(フォームは`.http`の読み書きビューという位置づけ)。

### D5: キーバインドはlazygit互換を優先

パネル間移動(`tab`/`shift+tab`)、上下移動(`j`/`k`)、パネル番号ジャンプ(`1`-`4`程度)、ヘルプ(`?`)、送信(`enter`)など、lazygit使用者が初見で触れることを設計基準とする。lazycurl独自の操作は最小限にとどめ、必要な場合は`?`のヘルプ画面に集約する。

## Risks / Trade-offs

- [Risk] `curl`未インストール環境では動作しない → Mitigation: 起動時に`curl --version`をチェックし、未検出/バージョン不足時は明確なエラーメッセージを出す
- [Risk] `-w '%{json}'`が古いcurl(7.70未満)で使えない → Mitigation: 起動時バージョンチェックで警告し、個別`%{}`変数を区切り文字で連結するフォールバックを用意する余地を残す(MVPでは最低バージョンを要求する方針でよい)
- [Risk] `.http`+プラグマは他ツール(VSCode REST Client等)には認識されない拡張構文である → Mitigation: プラグマは通常のコメント行として書式互換を保つため、他ツールで開いても壊れず単に無視される
- [Risk] グローバル保存のため、プロジェクトごとのgit管理には乗らない → Mitigation: MVPでは許容する。`~/.config/lazycurl/`自体を各自のdotfilesリポジトリに含めることは妨げない
- [Risk] サブプロセス実行のためユニットテストがしづらい(実プロセス起動が絡む) → Mitigation: curl呼び出し部分をインターフェースで抽象化し、テスト時はモック実行、結合テストでのみ実curlを使う

## Open Questions

- プラグマの語彙(`@insecure`/`@timeout`/`@no-redirect`)をMVP後にどこまで増やすか(`@proxy`、`@cert`等)
- `~/.config/lazycurl/`のディレクトリ配置がXDG非準拠OS(Windows)でどうあるべきか
