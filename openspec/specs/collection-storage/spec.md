## Purpose

`.http`+プラグマ形式でのリクエストコレクションの永続化を担う。`~/.config/lazycurl/`配下のディレクトリ構造、1コレクション=1`.http`ファイル、`###`区切りでの複数リクエスト管理を提供する。

## Requirements

### Requirement: グローバルコレクションディレクトリ
lazycurlはリクエストコレクションを`~/.config/lazycurl/collections/`配下にグローバルに保存しなければならない(SHALL)。プロジェクトローカルのディレクトリには保存しない。

#### Scenario: 初回起動時のディレクトリ作成
- **WHEN** `~/.config/lazycurl/`が存在しない状態でlazycurlを初回起動する
- **THEN** `~/.config/lazycurl/collections/`を含む必要なディレクトリ構造が作成される

#### Scenario: 既存ディレクトリの再利用
- **WHEN** `~/.config/lazycurl/collections/`が既に存在する状態で起動する
- **THEN** 既存のディレクトリ・ファイルはそのまま読み込まれ、上書きされない

### Requirement: `.http`+プラグマ形式での永続化
lazycurlはリクエストを`.http`形式(IntelliJ HTTP Client/VSCode REST Client互換のMethod/URL/Headers/Body構造)で読み書きしなければならない(SHALL)。HTTPリクエストとして表現できないcurl固有の設定は、リクエスト直前の`#`プラグマコメント行として保存する。

#### Scenario: リクエストの保存で.http形式に書き戻される
- **WHEN** ユーザーがフォームでリクエストを編集して保存する
- **THEN** Method/URL/Headers/Bodyが標準的な`.http`構文としてファイルに書き込まれる

#### Scenario: プラグマ付きリクエストの保存
- **WHEN** ユーザーがAdvanced設定(TLS検証スキップなど)を有効にして保存する
- **THEN** 対応するプラグマ行(例: `# @insecure`)がリクエスト直前に書き込まれる

#### Scenario: 未知のプラグマは無視される
- **WHEN** 手動編集やバージョン差異により未対応のプラグマ行を含む`.http`ファイルを読み込む
- **THEN** lazycurlは未知のプラグマ行を無視し、他のリクエスト情報は正常に読み込む(エラーで停止しない)

### Requirement: `###`区切りによる複数リクエストの管理
1つの`.http`ファイル(1コレクション)は、`###`で区切られた複数のリクエストを含むことができなければならない(SHALL)。

#### Scenario: 複数リクエストを含むファイルの読み込み
- **WHEN** `###`で区切られた3件のリクエストを含む`.http`ファイルを開く
- **THEN** 3件のリクエストが個別の項目としてリストされる

#### Scenario: リクエスト名の抽出
- **WHEN** `### Get user`のように`###`の後にテキストが続くリクエストを読み込む
- **THEN** そのテキストがリクエストの表示名として使われる

### Requirement: コレクション一覧の取得
lazycurlは`~/.config/lazycurl/collections/`配下の`.http`ファイルをコレクションとして列挙できなければならない(SHALL)。

#### Scenario: コレクション一覧の表示
- **WHEN** `~/.config/lazycurl/collections/`に複数の`.http`ファイルが存在する状態でlazycurlを起動する
- **THEN** それぞれのファイルがコレクションとしてTUIのパネルに一覧表示される
