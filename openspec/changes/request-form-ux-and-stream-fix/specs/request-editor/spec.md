## MODIFIED Requirements

### Requirement: フォームベースのリクエスト編集
lazycurlはMethod/URL/Query Params/Headers/Auth/Optionsをフォーム形式(キー・バリューのグリッド編集、行の追加・削除・有効/無効切り替え、プラグマのトグル/テキスト入力)で編集できなければならない(SHALL)。生の`.http`テキストを直接編集することを必須としてはならない。KVGridは非編集時、選択中の行だけでなく選択中のセル(KeyまたはValue)を視覚的に区別して表示しなければならない(SHALL)。リクエストの編集は、独立した全画面のフォームへ遷移するのではなく、`[0] Request`パネルの中でインラインに行われなければならない(SHALL)。

フォームのキー操作は3階層で構成されなければならない(SHALL): Level 0(Method/URL/Contentゾーン間の移動、およびContentゾーンにいる間のタブ切替)、Level 1(`enter`で入る、各ゾーン/タブ内部のナビゲーション)、Level 2(Level 1からさらに`enter`で入る、セル・フィールドのテキスト編集)。`esc`は常に1階層だけ上に戻らなければならない(SHALL)。

#### Scenario: Headerの追加
- **WHEN** ユーザーがHeadersタブで新しい行を追加し、キーと値を入力する
- **THEN** そのHeaderがリクエストに追加され、保存時に`.http`ファイルへ反映される

#### Scenario: Headerの無効化
- **WHEN** ユーザーが既存のHeader行のチェックボックスを外す
- **THEN** そのHeaderは送信対象から除外されるが、行自体はフォーム上に残る

#### Scenario: Authタイプの選択
- **WHEN** ユーザーがAuthタブで「Bearer Token」を選択しトークン値を入力する
- **THEN** 送信時に`Authorization: Bearer <値>`ヘッダーが自動的に付与される

#### Scenario: 既存行のKey列への移動
- **WHEN** ユーザーがParams/HeadersタブのLevel 1(グリッドのナビゲーション状態)で既存のParam/Header行にカーソルを合わせ、選択中のセルがValue列である状態で`h`/`left`/`shift+tab`を押す
- **THEN** カーソルがKey列に移動し、その列が視覚的にハイライトされ、`enter`でKeyを編集できる

#### Scenario: 行作成直後のセル表示
- **WHEN** ユーザーが`a`で新しい行を作成しKey・Valueを入力し終える
- **THEN** カーソルはValue列に留まるが、Value列が選択中セルとして視覚的にハイライトされ、Key列にいないことが画面上で判別できる

#### Scenario: Level 1入場時の常時ハイライト
- **WHEN** ユーザーがParams/Headersタブで`enter`を押しLevel 1(グリッドのナビゲーション状態)に入る、またはAuthタブで`enter`を押しLevel 1(タイプセレクタの状態)に入る
- **THEN** それ以上のキー入力を待たずに、カーソルが乗っているセル(KVGridのEnabled/Key/Value列)またはAuthのタイプセレクタが視覚的にハイライトされた状態で表示される

#### Scenario: 空グリッドでのenterによる新規行作成
- **WHEN** ユーザーがParams/Headersタブで行が1つも無い状態でLevel 1に入り`enter`を押す
- **THEN** `a`キーと同じ挙動で新しい行が作成され、Key列の編集(Level 2)が直ちに開始される

#### Scenario: Methodの視覚的ヒント
- **WHEN** ユーザーがLevel 0でMethodにフォーカスを合わせる
- **THEN** Method値が左右の矢印(例: `◀ GET ▶`)付きで表示され、`h`/`left`または`l`/`right`で値が直接切り替わることが視覚的に示される

#### Scenario: サブタブの切り替え
- **WHEN** ユーザーがLevel 0でContentゾーン(Params/Headers/Auth/Body/Optionsのいずれかが表示されている状態)にフォーカスがあるときに`h`/`left`または`l`/`right`を押す
- **THEN** 表示中のサブタブが前後に切り替わる。この操作はLevel 1(タブの中に`enter`で入った状態)には影響せず、Level 0でのみ機能する

#### Scenario: 編集中の他パネルの表示継続
- **WHEN** ユーザーがリクエストのフォームで編集している
- **THEN** `Collections`・`Response`・`History`・ステータスバーは非表示にならず表示され続ける

#### Scenario: フォームゾーンでの送信
- **WHEN** ユーザーが`[0] Request`パネルにフォーカスがある状態で`ctrl+r`キーを押す
- **THEN** 現在編集中のリクエストが送信される。`enter`キーはフィールドへの入力(Bodyでは改行)に使われるため送信には使われない

## ADDED Requirements

### Requirement: Optionsタブでのプラグマ編集
lazycurlは`Options`タブで、Stream(`@stream`)・Insecure(`@insecure`)・No-redirect(`@no-redirect`)・Timeout(`@timeout`)の4つのプラグマをフォームから直接編集できなければならない(SHALL)。Stream/Insecure/No-redirectはチェックボックス風の行、Timeoutは同じ見た目でテキスト入力欄を持つ行として、4項目とも統一された見た目で表示しなければならない(SHALL)。フォーム上でのプラグマの状態は、他のフィールド(Method/URL/Params/Headers/Auth/Body)と同様にキー入力のたびにメモリ上のリクエストへ即座に反映され、`ctrl+s`によるディスク保存時にも失われてはならない(SHALL NOT)。

#### Scenario: Streamプラグマの有効化
- **WHEN** ユーザーが`Options`タブでStream行を選択し`enter`または`space`でチェックを入れる
- **THEN** そのリクエストの`Pragmas.Stream`が有効になり、`ctrl+s`で保存すると`.http`ファイルに`# @stream`が書き込まれる

#### Scenario: 既存の`@stream`リクエストを編集してもプラグマが消えない
- **WHEN** ユーザーが`# @stream`付きの既存リクエストを`[0] Request`パネルにロードし、Params/HeadersタブなどOptions以外の箇所を移動・編集してから`ctrl+r`で送信する
- **THEN** `Pragmas.Stream`は有効なままであり、送信はストリーミング実行経路(逐次受信)で行われる

#### Scenario: Timeoutの入力
- **WHEN** ユーザーが`Options`タブのTimeout行で`enter`を押し`5s`のような値を入力して確定する
- **THEN** そのリクエストの`Pragmas.Timeout`が更新され、送信時に`curl --max-time`へ反映される

### Requirement: サブタブ・セル状態に応じたキーバインドヒント
ステータスバーのキーバインドヒントは、`[0] Request`パネルにフォーカスがある間、フォームの`editing`真偽値だけでなく、現在のサブタブ(Params/Headers/Auth/Body/Options)およびそのタブ内部の状態(Level 0/Level 1/Level 2のどこにいるか、KVGridなら行移動中か列移動中かセル編集中か、Authならどのフィールドか)に応じて、その時点で実際に使えるキーだけを表示しなければならない(SHALL)。

#### Scenario: Headersタブのグリッドナビゲーション中のヒント
- **WHEN** ユーザーがHeadersタブでLevel 1(行/列移動)の状態にいる
- **THEN** ステータスバーに`j/k`(行移動)・`h/l`(列移動)・`enter`(セル編集開始)・`a`(行追加)・`d/x`(行削除)・`space`(有効/無効切替)が表示される

#### Scenario: Authタブのタイプセレクタ表示中のヒント
- **WHEN** ユーザーがAuthタブでLevel 1(タイプセレクタ)の状態にいる
- **THEN** ステータスバーに`h/l`(タイプ選択)・`enter`/`down`(認証情報フィールドへ)が表示される

#### Scenario: Methodフォーカス時のヒント
- **WHEN** ユーザーがLevel 0でMethodにフォーカスしている
- **THEN** ステータスバーに`h/l: Method変更`が表示される
