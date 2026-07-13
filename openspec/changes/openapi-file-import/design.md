## Context

lazycurlは`.http`+プラグマ形式のcollectionストレージ(`internal/collection`)と、collectionごとのグローバルenvironmentストレージ(`internal/environment`、`~/.config/lazycurl/collections/env/<collection>/<name>.env.json`)をすでに持つ。どちらも`{{variable}}`という共通の変数記法(`internal/environment/expand.go`)でURL・Headers・Body・Authを展開する設計になっている。

今回追加するOpenAPI importは、この既存の2つのStore(`collection.Store`・`environment.Store`)にデータを書き込むだけの新規パッケージであり、両Storeの内部フォーマットや永続化ロジックには一切変更を加えない。TUIにも手を入れず、`cmd/lazycurl/main.go`にサブコマンド分岐を追加するのみ。

## Goals / Non-Goals

**Goals:**
- OpenAPI 3.0 / 3.1 (YAML/JSON) ファイル1つから、新規collection(`.http`ファイル)1つとenvironmentファイル群を生成する
- `lazycurl import <file> [--name NAME]` というCLIサブコマンドとして提供する(TUI起動なしで完結)
- path/query/headerパラメータをcollection横断でフラットな`{{variable}}`名前空間にマッピングし、servers[]ごとのenvironmentへ書き出す
- http basic/bearer/apiKey(header)のみを対象にAuth/Headerへマッピングする

**Non-Goals:**
- Swagger 2.0 (OpenAPI 2.0) のサポート
- `requestBody`のJSON Schemaからのサンプル値合成(example/examplesがない場合はBody空のまま)
- oauth2 / openIdConnect / apiKey(in: query, cookie) のマッピング
- `application/json`以外のcontent-type(multipart/form-data、application/x-www-form-urlencoded等)のBody変換
- 既存collectionへのマージインポート(常に新規collection作成のみ)
- TUIからのインポート起動

## Decisions

### 1. OpenAPIパースライブラリ: `pb33f/libopenapi`
kin-openapiと比較検討した結果、libopenapiを採用する。3.1はJSON Schema 2020-12に厳密準拠しており、3.0/3.1のどちらもネイティブに扱える。`$ref`解決(components/schemas, parameters, requestBodies, securitySchemes等)も内蔵しているため、自前でのref解決コードが不要になる。依存追加は許容されている。

### 2. CLIサブコマンドの実装方式
外部CLIフレームワーク(cobra等)は導入せず、`cmd/lazycurl/main.go`で`os.Args[1] == "import"`を見て分岐し、`flag.NewFlagSet("import", ...)`で`--name`を解釈する最小実装とする。サブコマンドが将来増えない限りこれで十分であり、依存を増やさずに済む。

### 3. 新規パッケージ`internal/openapi`
以下の責務を持つ:
- `Load(path string) (*Document, error)`: ファイル読み込み + libopenapiでのパース + バージョン判定(3.0/3.1以外はエラー)
- `Convert(doc *Document) (ImportResult, error)`: `paths`を走査し`[]httpfile.Request`と、server毎の`map[string]map[string]string`(environment名→変数マップ)を組み立てる
- `ImportResult`をcollectionのCLIコマンド層(`cmd/lazycurl`側)が受け取り、`collection.Store.CreateCollection` → `SaveRequests`、`environment.Store.Save`(server数分)、`environment.Store.SetActiveEnvironment`(先頭のenvironmentをアクティブにする)を呼び出す。

`internal/openapi`はcollection/environmentのStoreに依存せず、変換結果を返すだけの純粋な変換ロジックとする(既存の`internal/httpfile`が担っている責務分割と対称)。

### 4. collection名の決定
- `--name`指定時: その文字列をそのまま使う(空文字・`/`を含む場合はエラー)。
- 未指定時: `info.title`を kebab-case にスラグ化(小文字化、英数字以外の連続をハイフンに置換、先頭末尾のハイフン除去)。`info.title`が空、またはスラグ化結果が空文字になる場合は`imported-api`をデフォルト名にする。
- 生成しようとしたcollection名が既に存在する場合はエラーで中断する(上書き・自動リネームはしない。ユーザーが`--name`で明示的に選び直す)。

### 5. request名の決定
`summary` → `operationId` → `"<METHOD> <path>"`(例: `GET /users/{id}`)の優先順で採用する。`.http`の`### <name>`行として使われるだけの表示用ラベルであり、collection内での一意性は要求しない(既存の`DuplicateRequest`も名前の一意性を前提としていない)。

### 6. path/queryパラメータ → `{{variable}}`
- pathパラメータの`{id}`は、既存の変数記法にあわせてそのまま`{{id}}`へ変換する(文字置換のみ)。
- queryパラメータは`internal/tui/form/convert.go`の`joinURL`と同じ`?key=value&...`形式でURLに付与し、値部分を`{{key}}`にする。
- headerパラメータはHeadersへ`Key: {{key}}`として追加する(Enabled: true)。
- 変数名は**collection全体でフラットな1つの名前空間**を共有する(environmentファイルの構造がcollectionごとにフラットなmap[string]stringであるため)。同名だが意味の異なるpathパラメータが複数operationにまたがる場合、生成されるenvironment上は1つの変数に統合される(既知の制限。詳細はRisksを参照)。

### 7. servers[] → environmentファイル
- `servers`が空の場合は`host`変数を持たない1つの`default`environmentのみを生成する(値はrequest変換で発見した変数のみ)。
- `servers`が1件のみの場合、environment名は`default`。
- `servers`が複数件の場合、各環境の名前は`servers[i].description`をスラグ化したもの、descriptionが空またはスラグ化結果が空の場合は`server-<i+1>`(1-indexed)とする。
- server URLに`servers[i].variables`によるテンプレート変数(`https://{env}.api.example.com`のような`{var}`)が含まれる場合、importの時点で各変数の`default`値を使って静的なURL文字列に解決してから`host`として書き込む(`{{}}`の二重ネストは行わない)。
- 生成される各environmentファイルには、`host`(servers[]から)に加えて、request変換で収集した全変数名(path/query/header)を含める。値は該当パラメータに`example`/`examples`または`schema.default`があればそれを、なければ空文字とする。同じ変数名がexample値ありとなしの両方で出現した場合は、最初に見つかったexample値を優先する。
- 生成したenvironmentのうち先頭(1件目)を`environment.Store.SetActiveEnvironment`でアクティブに設定する。

### 8. Auth / セキュリティスキームのマッピング
`security`(operation単位、未指定ならglobalの`security`にフォールバック)配列の各要素を順に見て、最初にサポート対象のスキームが見つかった時点で採用し、以降は無視する(OR条件のうち1つだけ反映する簡易実装):
- `type: http, scheme: basic` → `Auth{Type: AuthBasic, Username: "{{<slug>}}", Password: "{{<slug>}}_password"}` のように、ユーザー名・パスワードそれぞれに変数を割り当てる
- `type: http, scheme: bearer` → `Auth{Type: AuthBearer, Token: "{{<slug>}}"}`
- `type: apiKey, in: header` → `Headers`に`Key: <scheme.name>, Value: "{{<slug>}}"`を追加(Enabled: true)
- 上記以外(oauth2, openIdConnect, apiKey in query/cookie)は無視する
- `<slug>`はセキュリティスキーム名(components.securitySchemes配下のキー)をスラグ化したもの。これらの変数もvariable収集の対象とし、environmentファイルに空文字で追加する(認証情報は例示値を書き込まない)。

### 9. requestBodyの変換
`requestBody.content`に`application/json`キーが存在する場合のみBodyを生成する。優先順位は`example` → `examples`(先頭の1件) → 空文字。スキーマからの合成は行わない。それ以外のcontent-typeキーのみが存在する場合はBodyを空のままにする。

### 10. インポート結果のサマリ出力
標準出力に、生成collection名・request件数・environment件数・無視したoperation/パラメータ/セキュリティスキームがあればその件数を表示する。パースエラー(非対応バージョン、不正なYAML/JSON等)は標準エラーへ出力しexit code 1とする。

## Risks / Trade-offs

- **[Risk] 変数名の名前空間衝突** 同名だが意味が異なるpath/queryパラメータが複数operationに存在すると、environment上で1つの変数に統合されてしまう → **Mitigation**: v1では許容し、必要ならユーザーがimport後にrequest編集フォームでURL/Headers内の変数名を手動リネームする。将来的にrequestスコープの変数機構が必要になれば別提案とする。
- **[Risk] `pb33f/libopenapi`の型APIが複雑** 3.0/3.1双方の差異を吸収するAPI設計になっており学習コストがある → **Mitigation**: `internal/openapi`パッケージ内に閉じ込め、外部(collection/environment/TUI)からはlibopenapiの型が一切見えないようにする。
- **[Risk] 大きなOpenAPIファイル(数百operation)で肥大化したcollectionが生成される** 一括インポートによりcollection内のrequest数が多すぎて一覧性が落ちる可能性 → **Mitigation**: v1では許容(フィルタリング機能は将来課題)。サマリ出力で件数を明示し、ユーザーが把握できるようにする。
- **[Trade-off] Body合成をしない** example/examplesがないrequestBodyはBody空になる → 手動編集フォームでの補完が前提。スキーマ合成の複雑さ(allOf/oneOf/$ref循環等)を避けるための意図的なスコープ縮小。

## Migration Plan

新規追加のみで既存データ・既存機能への影響はない。ロールバックはコマンド追加分のコードを削除するだけで足りる(生成されたcollection/environmentファイルは通常のファイルとして残るが、他機能に影響しない)。

## Open Questions

なし(本提案の範囲内の決定事項はDecisionsセクションで確定済み)。
