## 1. 依存関係の追加

- [ ] 1.1 `go get github.com/pb33f/libopenapi` で依存を追加し、`go.mod`/`go.sum`を更新する

## 2. `internal/openapi`パッケージ: パース

- [ ] 2.1 `Load(path string) (*Document, error)`を実装する。ファイル読み込み、libopenapiでのパース、OpenAPIバージョン判定(3.0/3.1以外はエラー)、YAML/JSON構文エラーのハンドリングを行う
- [ ] 2.2 パース失敗時・非対応バージョン時のエラーメッセージを整備する(呼び出し側でそのまま標準エラーに出せる文面にする)

## 3. `internal/openapi`パッケージ: 変数収集とスラグ化

- [ ] 3.1 文字列をkebab-caseにスラグ化するヘルパーを実装する(collection名・environment名・security scheme名の生成で共用)
- [ ] 3.2 パラメータ(path/query/header)から変数名と初期値(example/examples/schema.defaultの優先順、無ければ空文字)を収集するロジックを実装する
- [ ] 3.3 収集した変数をrequest群横断でマージし、重複変数名は最初に見つかったexample値を優先する処理を実装する

## 4. `internal/openapi`パッケージ: requestへの変換

- [ ] 4.1 `paths`を走査し、operationごとに`httpfile.Request`を組み立てる関数を実装する
- [ ] 4.2 request名の決定ロジック(summary → operationId → "METHOD path")を実装する
- [ ] 4.3 pathパラメータの`{param}` → `{{param}}`置換を実装する
- [ ] 4.4 queryパラメータをURLのクエリ文字列(`{{param}}`値)として付与する処理を実装する
- [ ] 4.5 headerパラメータをHeadersへ`{{param}}`値の有効な行として追加する処理を実装する
- [ ] 4.6 `requestBody.content["application/json"]`のexample/examplesをBodyへ転記する処理を実装する(未定義・非JSON content-typeのみの場合は空のまま)
- [ ] 4.7 `security`(operation単位、フォールバックでglobal)からAuth/Headersへのマッピングを実装する: http basic → Auth(Basic)、http bearer → Auth(Bearer)、apiKey(header) → Headers行。oauth2/openIdConnect/apiKey(query,cookie)は無視する

## 5. `internal/openapi`パッケージ: environment生成

- [ ] 5.1 `servers[]`からenvironment名を決定するロジックを実装する(1件なら`default`、複数件ならdescriptionのスラグ化、descriptionが無ければ`server-<index>`、0件なら`default`のみ)
- [ ] 5.2 `servers[i].variables`のテンプレート変数(`{var}`)をdefault値で静的なURL文字列に解決する処理を実装する
- [ ] 5.3 各environmentに`host`+収集済み全変数(値はexample優先・無ければ空文字)を含む`map[string]string`を組み立てる処理を実装する
- [ ] 5.4 `ImportResult`型(collection名候補、`[]httpfile.Request`、environment名→変数マップ、無視した項目のサマリ情報)を定義し、2〜5の結果をまとめて返す`Convert`関数を実装する

## 6. CLIサブコマンド

- [ ] 6.1 `cmd/lazycurl/main.go`に`os.Args[1] == "import"`のサブコマンド分岐を追加し、既存のTUI起動パスと分離する
- [ ] 6.2 `import`サブコマンド用の`flag.FlagSet`を実装し、位置引数(ファイルパス)と`--name`オプションを解釈する
- [ ] 6.3 collection名の決定処理(`--name`指定 or `info.title`のスラグ化、空の場合は`imported-api`)を実装する
- [ ] 6.4 既存collection名との衝突チェックを実装し、衝突時はcollectionを作成せずエラー終了する
- [ ] 6.5 `internal/openapi.Convert`の結果を使い、`collection.Store.CreateCollection` → `SaveRequests`、`environment.Store.Save`(environmentごと)、`environment.Store.SetActiveEnvironment`(先頭environment)を呼び出す配線を実装する
- [ ] 6.6 成功時のサマリ出力(collection名、request件数、environment件数)を標準出力に実装する
- [ ] 6.7 失敗時(パースエラー、collection名衝突、ファイル不在等)はエラーメッセージを標準エラーに出し、非ゼロ終了コードで終了する処理を実装する

## 7. テスト

- [ ] 7.1 `internal/openapi`のユニットテストを作成する: OpenAPI 3.0/3.1それぞれのサンプルファイルからのrequest変換、request名決定の優先順位、path/query/headerパラメータの変数化
- [ ] 7.2 servers[]の件数パターン(0件/1件/複数件、description有無、variables有無)ごとのenvironment生成をテストする
- [ ] 7.3 security schemeマッピング(basic/bearer/apiKey header/非対応スキーム無視)のテストを追加する
- [ ] 7.4 requestBodyのexample転記(JSON有り/example無し/非JSON content-typeのみ)のテストを追加する
- [ ] 7.5 非対応バージョン・不正ファイルに対するエラーハンドリングのテストを追加する
- [ ] 7.6 CLIサブコマンド(`cmd/lazycurl`)の統合テスト、または`internal/openapi`+Store呼び出しを組み合わせたテストで、collection名衝突時に既存collectionが変更されないことを検証する

## 8. ドキュメント

- [ ] 8.1 `openspec/specs/`配下に本変更をアーカイブする際、`openapi-import`のspecを反映する(`/opsx:archive`実行時)
- [ ] 8.2 必要であればREADME等に`lazycurl import`コマンドの使い方を追記する
