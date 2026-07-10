## Why

lazycurlでAPIを操作するには、collectionとrequestを手作業で1件ずつ作成する必要がある。すでにOpenAPI(Swagger)仕様が公開されているAPIであっても、そこからrequestを作り直す作業はゼロから書くのとほぼ変わらない。OpenAPI 3.0/3.1ファイルを取り込んで、pathsに定義された各operationをcollection内のrequestとして自動生成できれば、既存APIの検証・操作を始めるまでの立ち上がりが大きく短縮される。

## What Changes

- 新しいCLIサブコマンド`lazycurl import <file> [--name NAME]`を追加する。TUIの起動なしで完結する、独立したコマンドとする。
- OpenAPI 3.0/3.1形式のYAML/JSONファイルをパースする(ライブラリ: `pb33f/libopenapi`)。
- パース結果から新規collectionを1つ作成する。collection名は`--name`指定があればそれを使い、なければ`info.title`から生成する。
- `paths`配下の各operation(method + path)を1件の`.http` requestに変換する:
  - path/queryパラメータ → URL中の`{{variable}}`
  - headerパラメータ → Headers行
  - `requestBody`の`example`/`examples`があればBodyとして転記(スキーマからの合成は行わない)
  - `securitySchemes`のうち`http basic`/`http bearer`/`apiKey(in: header)`のみAuth・Headerへマッピング。oauth2/openIdConnect/apiKey(query,cookie)は無視する
- `servers[]`の各エントリごとに、既存のグローバルenvironmentストア(`~/.config/lazycurl/collections/env/<collection>/`)へenvironmentファイルを生成する。各environmentには`host`と、request変換時に発見したpath/query/headerの変数名すべてを(example値があればそれを、なければ空文字で)含める。
- インポート結果のサマリ(作成したcollection名、request件数、environment件数、無視した項目があればその旨)を標準出力に表示する。

## Capabilities

### New Capabilities
- `openapi-import`: OpenAPI 3.0/3.1ファイルをパースし、collectionのrequest群とenvironment群を生成するCLIインポート機能

### Modified Capabilities
(なし。生成したcollection/environmentの永続化は既存の`collection-storage`・`environment-variables`のStore APIをそのまま利用し、要件変更は発生しない)

## Impact

- 新規パッケージ`internal/openapi`(パース・マッピングロジック)を追加
- `cmd/lazycurl/main.go`にサブコマンド分岐を追加(現状は引数解釈なしでTUIを起動するのみ)
- 新規依存: `github.com/pb33f/libopenapi`(および付随するYAMLパーサ等の間接依存)
- 既存の`internal/collection`・`internal/environment`パッケージは変更なしで再利用
