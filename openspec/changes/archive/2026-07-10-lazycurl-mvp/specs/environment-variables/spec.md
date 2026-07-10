## ADDED Requirements

### Requirement: environmentファイルによる変数定義
lazycurlはコレクションごとに複数のenvironmentファイル(例: dev/staging/prod)を管理し、それぞれがキー・バリュー形式の変数を定義できなければならない(SHALL)。

#### Scenario: environmentファイルの読み込み
- **WHEN** コレクションに`env/dev.env.json`と`env/prod.env.json`が存在する
- **THEN** 両方のenvironmentが選択可能な一覧として表示される

### Requirement: アクティブ環境の切り替え
ユーザーはコレクションに対して1つのアクティブなenvironmentを選択できなければならない(SHALL)。

#### Scenario: 環境の切り替え
- **WHEN** ユーザーがenvironment一覧から`prod`を選択する
- **THEN** 以降そのコレクション内のリクエストの変数展開には`prod`環境の値が使われる

### Requirement: `{{variable}}`展開
リクエストのURL・Headers・Body中の`{{variable}}`記法は、アクティブなenvironmentの値で展開されてからcurlに渡されなければならない(SHALL)。

#### Scenario: URL中の変数展開
- **WHEN** リクエストが`GET {{host}}/users/{{id}}`であり、アクティブ環境で`host=https://api.example.com`が定義されている
- **THEN** 実際に送信されるURLは`https://api.example.com/users/{{id展開後の値}}`になる

#### Scenario: 未定義変数の扱い
- **WHEN** リクエストが参照する変数がアクティブ環境に定義されていない
- **THEN** lazycurlは送信前に未定義変数がある旨をエラーとして表示し、送信を行わない
