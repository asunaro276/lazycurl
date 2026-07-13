## ADDED Requirements

### Requirement: CLIサブコマンドによるインポート実行
lazycurlは`lazycurl import <file> [--name NAME]`というCLIサブコマンドを提供しなければならない(SHALL)。このサブコマンドはTUIを起動せず、インポート処理のみを実行して終了しなければならない(SHALL)。

#### Scenario: ファイルパスのみ指定してインポート
- **WHEN** ユーザーが`lazycurl import ./petstore.yaml`を実行する
- **THEN** TUIは起動せず、`petstore.yaml`の内容から新規collectionが作成され、結果のサマリが標準出力に表示されてコマンドが終了する

#### Scenario: --nameでcollection名を指定
- **WHEN** ユーザーが`lazycurl import ./petstore.yaml --name my-api`を実行する
- **THEN** 作成されるcollectionの名前は`my-api`になる

### Requirement: OpenAPI 3.0/3.1のパース
lazycurlはOpenAPI 3.0および3.1形式のYAML/JSONファイルをパースできなければならない(SHALL)。対応していないバージョン(OpenAPI 2.0/Swaggerなど)や不正なファイルを指定した場合はエラーを標準エラーへ出力し、collectionを作成せずにゼロ以外の終了コードで終了しなければならない(SHALL)。

#### Scenario: OpenAPI 3.0ファイルの読み込み
- **WHEN** `openapi: 3.0.3`と宣言されたYAMLファイルをインポートする
- **THEN** ファイルが正常にパースされ、collectionが作成される

#### Scenario: OpenAPI 3.1ファイルの読み込み
- **WHEN** `openapi: 3.1.0`と宣言されたJSONファイルをインポートする
- **THEN** ファイルが正常にパースされ、collectionが作成される

#### Scenario: 非対応バージョンの拒否
- **WHEN** `swagger: "2.0"`と宣言されたファイルをインポートしようとする
- **THEN** lazycurlはエラーメッセージを標準エラーへ出力し、collectionを作成せずに終了コード非ゼロで終了する

#### Scenario: 不正なファイルの拒否
- **WHEN** YAML/JSONとして構文的に不正なファイルをインポートしようとする
- **THEN** lazycurlはパースエラーを標準エラーへ出力し、collectionを作成せずに終了コード非ゼロで終了する

### Requirement: collectionの新規作成
lazycurlはインポートのたびに新規collectionを1つ作成しなければならない(SHALL)。既存collectionへのマージは行わない。同名のcollectionが既に存在する場合はエラーとし、既存collectionを上書きしてはならない(SHALL NOT)。

#### Scenario: info.titleからのcollection名生成
- **WHEN** `--name`を指定せず、`info.title`が`Petstore API`であるファイルをインポートする
- **THEN** `petstore-api`のようなkebab-case名でcollectionが作成される

#### Scenario: info.titleが空の場合のフォールバック
- **WHEN** `--name`を指定せず、`info.title`が空文字またはスラグ化結果が空になるファイルをインポートする
- **THEN** `imported-api`という名前でcollectionが作成される

#### Scenario: 既存collection名との衝突
- **WHEN** 生成しようとしたcollection名が既存collectionの名前と一致する
- **THEN** lazycurlはエラーを表示してインポートを中断し、既存collectionの内容を変更しない

### Requirement: operationからrequestへの変換
lazycurlは`paths`配下の各operation(HTTPメソッド x パス)を1件の`.http` requestへ変換しなければならない(SHALL)。requestの表示名は`summary`、なければ`operationId`、どちらもなければ`"<METHOD> <path>"`の優先順で決定しなければならない(SHALL)。

#### Scenario: summaryがある場合のrequest名
- **WHEN** operationに`summary: "ユーザー取得"`が設定されている
- **THEN** 生成されるrequestの名前は`ユーザー取得`になる

#### Scenario: summaryがなくoperationIdがある場合のrequest名
- **WHEN** operationに`summary`が無く`operationId: getUserById`が設定されている
- **THEN** 生成されるrequestの名前は`getUserById`になる

#### Scenario: summaryもoperationIdも無い場合のrequest名
- **WHEN** operationに`summary`も`operationId`も設定されていない`GET /users/{id}`が定義されている
- **THEN** 生成されるrequestの名前は`GET /users/{id}`になる

### Requirement: path/queryパラメータの変数化
lazycurlはpathパラメータおよびqueryパラメータを、対応するrequestのURL中で`{{variable}}`記法に変換しなければならない(SHALL)。

#### Scenario: pathパラメータの変換
- **WHEN** operationのパスが`/users/{id}`であり、`id`がpathパラメータとして定義されている
- **THEN** 生成されるrequestのURLは`{id}`部分が`{{id}}`に置換される

#### Scenario: queryパラメータの変換
- **WHEN** operationに`limit`という名前のqueryパラメータが定義されている
- **THEN** 生成されるrequestのURLに`?limit={{limit}}`のようなクエリ文字列が付与される

### Requirement: headerパラメータの変換
lazycurlはheaderパラメータを、対応するrequestのHeadersへ`{{variable}}`記法の値として追加しなければならない(SHALL)。

#### Scenario: headerパラメータの変換
- **WHEN** operationに`X-Request-Id`という名前のheaderパラメータが定義されている
- **THEN** 生成されるrequestのHeadersに`X-Request-Id: {{X-Request-Id}}`が有効な行として追加される

### Requirement: servers[]からのenvironment生成
lazycurlは`servers[]`の各エントリごとに、対象collectionのenvironmentファイルを既存のグローバルenvironmentストア(`~/.config/lazycurl/collections/env/<collection>/`)へ生成しなければならない(SHALL)。生成された最初のenvironmentは、そのcollectionのアクティブなenvironmentとして設定されなければならない(SHALL)。

#### Scenario: servers[]が1件のみの場合
- **WHEN** `servers`に1件のみエントリがあるファイルをインポートする
- **THEN** `default`という名前のenvironmentファイルが1件生成され、アクティブなenvironmentとして設定される

#### Scenario: servers[]が複数件の場合
- **WHEN** `servers`に`description: "Development"`と`description: "Production"`を持つ2件のエントリがあるファイルをインポートする
- **THEN** `development`と`production`のようにdescriptionをスラグ化した名前で2件のenvironmentファイルが生成される

#### Scenario: descriptionが無いservers[]エントリ
- **WHEN** `servers`に複数件あり、うち1件に`description`が設定されていない
- **THEN** そのエントリのenvironment名は`server-<インデックス>`のような形式になる

#### Scenario: servers[]が空の場合
- **WHEN** ファイルに`servers`が定義されていない、または空配列である
- **THEN** `host`変数を持たない`default`という名前のenvironmentファイルが1件生成される

#### Scenario: 変数の共有
- **WHEN** 複数のenvironmentが生成される
- **THEN** 各environmentファイルには`host`に加えて、request変換で発見した全てのpath/query/header変数名が(example値があればそれを、なければ空文字で)含まれる

### Requirement: パラメータのexample/default値の反映
lazycurlはパラメータに`example`・`examples`・`schema.default`のいずれかが定義されている場合、生成するenvironment変数の初期値としてその値を使用しなければならない(SHALL)。いずれも定義されていない場合は空文字を初期値としなければならない(SHALL)。

#### Scenario: exampleが定義されているパラメータ
- **WHEN** pathパラメータ`id`に`example: "123"`が定義されている
- **THEN** 生成されるenvironmentファイルの`id`変数の値は`"123"`になる

#### Scenario: example未定義のパラメータ
- **WHEN** queryパラメータ`limit`にexample・default値のいずれも定義されていない
- **THEN** 生成されるenvironmentファイルの`limit`変数の値は空文字になる

### Requirement: 認証(Auth)のマッピング
lazycurlはoperationに適用される`security`要件のうち、`http basic`・`http bearer`・`apiKey(in: header)`のいずれかに一致する最初のスキームを、requestのAuthまたはHeadersへ反映しなければならない(SHALL)。oauth2・openIdConnect・`apiKey(in: query または cookie)`は無視しなければならない(SHALL)。

#### Scenario: http basic認証のマッピング
- **WHEN** operationが`type: http, scheme: basic`のセキュリティスキームを要求する
- **THEN** 生成されるrequestのAuthはBasic認証として設定され、ユーザー名・パスワードにはそれぞれ`{{variable}}`が割り当てられる

#### Scenario: http bearer認証のマッピング
- **WHEN** operationが`type: http, scheme: bearer`のセキュリティスキームを要求する
- **THEN** 生成されるrequestのAuthはBearer認証として設定され、トークンには`{{variable}}`が割り当てられる

#### Scenario: apiKey(header)のマッピング
- **WHEN** operationが`type: apiKey, in: header, name: X-Api-Key`のセキュリティスキームを要求する
- **THEN** 生成されるrequestのHeadersに`X-Api-Key: {{variable}}`が追加される

#### Scenario: 非対応スキームの無視
- **WHEN** operationが`type: oauth2`のセキュリティスキームのみを要求する
- **THEN** 生成されるrequestのAuthはNoneのままであり、Headersにも何も追加されない

### Requirement: requestBodyの変換
lazycurlは`requestBody.content`に`application/json`が存在する場合、その`example`(なければ`examples`の先頭)をrequestのBodyとして転記しなければならない(SHALL)。どちらも存在しない場合、またはcontent-typeが`application/json`以外のみの場合、Bodyは空のままとしなければならない(SHALL)。

#### Scenario: exampleが定義されたrequestBody
- **WHEN** operationの`requestBody.content["application/json"].example`にJSONオブジェクトが定義されている
- **THEN** 生成されるrequestのBodyにそのJSONがそのまま転記される

#### Scenario: exampleが未定義のrequestBody
- **WHEN** operationに`requestBody`はあるが`example`・`examples`のいずれも定義されていない
- **THEN** 生成されるrequestのBodyは空文字のままになる

#### Scenario: JSON以外のcontent-typeのみのrequestBody
- **WHEN** operationの`requestBody.content`に`application/xml`のみが定義されている
- **THEN** 生成されるrequestのBodyは空文字のままになる

### Requirement: インポート結果のサマリ表示
lazycurlはインポート成功時に、作成したcollection名・生成したrequest件数・生成したenvironment件数を標準出力へ表示しなければならない(SHALL)。

#### Scenario: 成功時のサマリ
- **WHEN** 5件のoperationと2件のserversを含むファイルのインポートが成功する
- **THEN** collection名、request 5件、environment 2件が生成された旨が標準出力に表示される
