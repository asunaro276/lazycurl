## Why

現在lazycurlは`Adhoc`/`Collections`という2つのモードを`[`/`]`で切り替える構造になっているが、これが2つの操作性問題を生んでいる。第一に、Editorパネル(埋め込みフォーム)にフォーカスすると`inFormZone()`が`ctrl+c`以外の全キーを奪うため、パネル番号キー(`1`-`4`)によるパネル間移動がEditorにいる間だけ機能しない。第二に、フォーム内の入力欄が種類によらず同じキーバインドヒントしか出さないため、今どの操作ができるのか画面から読み取れない。これらはどちらも「モードとフォームが同時に画面を専有する」という現在の構造に起因しており、モードそのものを撤廃し単一画面へ統合することで解消する。

## What Changes

- **BREAKING**: `Adhoc`/`Collections`のモード切り替え(`[`/`]`キー)を廃止し、常時4パネル固定の単一画面に統合する
- パネル構成を`[0] Request`(編集フォーム) / `[1] Response` / `[2] Collections` / `[3] History`の2x2グリッドとし、4パネルを常時同時表示する(いずれも画面を専有する「モード」は存在しない)
- Collectionsパネルは、カーソルが乗っているコレクションのみアコーディオン展開してリクエスト一覧を表示し、他のコレクションは名前のみの1行に畳む。リクエスト行で`enter`すると`[0] Request`パネルにそのリクエストが読み込まれ、フォーカスが`[0]`へ移る
- Historyパネルは、カーソルが乗っているエントリのみアコーディオン展開して表示し、他のエントリは1行summaryに畳む。`enter`で確定すると`[1] Response`パネルにそのエントリの結果が反映される
- `[0] Request`パネルにnormal/insertの2段階フォーカス状態を導入する: normal状態では`j`/`k`等でName/Method/URL/タブ間を移動でき、`0`-`3`キーは常にパネル切替として機能する。`enter`で対象フィールドのinsert状態に入ると文字入力を奪い、`esc`でnormalへ戻る。この状態機械は`internal/tui/form/kvgrid.go`の`KVGrid`が既に持つ`editing`パターンをフォーム全体に一段引き上げたものである
- 下部のキーバインドヒントは、フォーカス中のパネルとnormal/insert状態から動的に導出し、常に「今押せるキー」を表示する
- `Adhoc`モードが担っていた「保存するまでメモリ内のみで一時保持」「collectionへの保存」「履歴の共有」という性質は、モードではなく「Requestパネルがcollectionに属していない状態」として`tui-shell`capabilityに引き継がれる **BREAKING**(`adhoc-mode`capabilityとしては廃止)

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `tui-shell`: パネルレイアウト要求を「モード別レイアウト」から「常時4パネル固定グリッド」へ全面的に書き換える。lazygit互換キーバインド要求にnumber-key(0-3)によるパネル切替とnormal/insert状態機械を追加する。`adhoc-mode`から移管される一時性・collectionへの保存の要求を追加する
- `request-editor`: 「リクエストの新規作成・複製・削除」要求からモード(Adhoc/Collections)への言及を除去し、「collectionに属した状態/属さない状態」という表現に置き換える

### Removed Capabilities

- `adhoc-mode`: モード概念そのものが廃止されるため、このcapabilityは廃止する。各要求は`tui-shell`(モード切替以外)へ移管、または(モード切替自体は)不要になるため削除する

## Impact

- `internal/tui/shell/model.go`: `Mode`型・`ModeAdhoc`/`ModeCollections`定数を削除。`Panel`の並びを`[0]Request [1]Response [2]Collections [3]History`に固定。`requestZone`(zoneList/zoneForm)を、Requestパネルのnormal/insert状態に置き換え
- `internal/tui/shell/update.go`: `handleKey`/`toggleMode`/`panelsForMode`を全面書き換え。Collections/Historyのアコーディオン用カーソル移動処理を追加
- `internal/tui/shell/view.go`: `viewAdhocLayout`/`viewCollectionsLayout`/`viewModeTabs`を廃止し、常時2x2グリッドを描画する単一のView関数に統合。Collections/Historyのアコーディオン描画、状態に応じたフッターヒント描画を追加
- `internal/tui/form/editor.go`: フォーム全体にnormal/insert状態(`editing bool`相当)を追加し、`KVGrid`と同様のパターンに揃える
- 実装順序: 別change `request-name-on-save`(Name欄をフォーム常設から保存時プロンプトへ)を先に実装し、保存フローが単純化された状態でこのchangeに着手することを推奨する
