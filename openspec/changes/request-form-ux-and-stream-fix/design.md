## Context

`[0] Request`パネルは`internal/tui/form.Editor`が単一の`tea.Model`として実装しており、`focusZone`(Method/URL/Content)という1段のフォーカス概念と、フォーム全体の`editing`真偽値(normal/insert)という2段のフォーカス概念しか持っていない。Params/HeadersのKVGrid、Authのタイプ+フィールド選択、Bodyのテキストエリアは、それぞれ独立した内部状態(`KVGrid.focused`/`editing`、`Editor.authField`など)を持つが、フォーム全体の状態遷移とは疎結合で、以下の問題を生んでいる。

- `Editor.FromRequest`/`ToRequest`が`httpfile.Pragmas`のうち`Stream`を往復させていない(調査済みの実装上のバグ)。`Shell.syncEditorToTarget()`がキー入力ごとに`ToRequest()`の結果で`s.requests[idx]`を上書きするため、`@stream`付きリクエストをパネルで一度でも操作すると`Pragmas.Stream`がメモリ上で失われ、`sendLoadedCurrent`が`beginStreamingSend`ではなく`Execute()`(バッチ実行)に分岐してしまう。
- 4つのプラグマ(Stream/Insecure/NoRedirect/Timeout)をフォームから編集する手段が無い。
- `[`/`]`によるタブ切替、`h`/`l`によるMethod直接変更、KVGrid/Authの「フォーカスしても何も光らない」状態、`footerHint()`の2値分岐など、キー操作とその視覚的フィードバックが噛み合っていない。

## Goals / Non-Goals

**Goals:**
- `Pragmas.Stream`を含む4プラグマ全てをEditorが往復させ、Optionsタブから編集可能にする
- フォームのキー操作を3階層(Level 0/1/2)として明確化し、既存のKVGrid/Auth内部状態をLevel 1に位置づける
- Level 1に入った時点(`enter`直後)でカーソル/セレクタが常時ハイライトされるようにする
- タブ切替を`h`/`l`に、Methodの変更を視覚的に分かるようにする(インタラクションは維持)
- ステータスバーのキーバインドヒントをサブタブ・内部状態ごとに動的化する

**Non-Goals:**
- `curlexec`のストリーミング受信配線(`ExecuteStreaming`/`streamChunkMsg`)自体の変更。調査の結果、逐次受信は正しく動作しており、今回のバグはEditor側の往復漏れに限定される
- Adhoc/Collectionsモードの復活や、パネル構成自体の変更(既に単一の`[0] Request`パネルに統合済みで、CLAUDE.mdの現行アーキテクチャに従う)
- Params/Headers以外のグリッド系UIパターンの汎用化

## Decisions

### 1. プラグマの往復漏れは「取りこぼしバグの修正」として最小差分で直す
`Editor`構造体にStream用のフィールド(例: `pragStream bool`)を追加し、`FromRequest`/`ToRequest`で他の3プラグマと全く同じパターンで読み書きする。既存の`pragK`/`pragTO`/`pragNoRdir`と実装を揃えることで、レビューコストを下げる。

### 2. プラグマ編集用に5番目のタブ「Options」を新設する
- 名称は「Pragmas」ではなく「Options」(ユーザー判断)
- `tabLabels`に`"Options"`を追加し、`tab`型の定数に`TabOptions`を追加する
- 表示は4行固定・見た目統一(Stream/Insecure/NoRedirectはチェックボックス風、Timeoutは同じ行の見た目でテキスト入力欄を右側に持つ)とし、KVGridのような可変長リストではなく、固定4行の専用ビュー(またはKVGridに似た最小限の専用コンポーネント)として実装する。行の追加・削除は無いため、KVGridそのものを再利用する必要はない。
- 代替案として検討したがユーザーの意向で不採用: Method/URL行の下に常設ストリップとして表示する案、Authタブに統合する案。いずれも既存のタブという一貫した操作モデルから外れるため、5番目のタブとして追加する案を採用。

### 3. フォームのキー操作を3階層(Level 0/1/2)として整理する
既存の`focusZone`(Method/URL/Content)による移動をLevel 0とし、`enter`で各ゾーン/タブの内部状態(KVGridの`focused`、Authの`authField`、Bodyのtextarea)に入ることをLevel 1への遷移として明確化する。KVGridのセル編集(`editing`)、Authの資格情報フィールド編集は、Level 1からさらに`enter`で入るLevel 2として位置づける。`esc`は常に1階層だけ戻る(既存の「1階層戻る」実装方針を維持)。

この整理によって、`h`/`l`の意味が階層ごとに変わっても衝突しないことを確認済み: Level 0のContentゾーンでは`h`/`l`はタブ切替、Level 1に入った後は各タブが独自に`h`/`l`を再定義する(KVGridは列移動、Authはタイプ選択、Bodyはテキストエリアなので特に意味を持たない)。タブを切り替えたい場合は`esc`でLevel 0に戻ってから`h`/`l`を押す。

- 代替案として検討したがユーザーの意向で不採用: タブ切替を数字キーまたは頭文字キーにする案(パネル番号`0`-`3`との衝突を避ける意図だったが、数字キーはパネル移動用に予約したいとの要望のため不採用)。KVGridの列移動が端に達したら次のタブに「こぼれる」案(Auth側のタイプ選択がループ挙動のため端が無く、成立しないため不採用)。

### 4. Level 1入場時の常時ハイライトは既存の`focused`フラグの立ち上げタイミングを早めるだけで実現する
現在も`KVGrid.View()`は`g.focused`が真であればカーソルセルをハイライトする実装になっている。問題は`Editor.enterInsert()`が`editing`と`focused`を同時に立てるタイミングが「Level 0→Level 1」の遷移そのものであるにもかかわらず、ユーザー体感としては「まだ何も選んでいないのに強調表示が始まる」という誤解に繋がっていた点ではなく、実際には既に`enter`一発でハイライトされる実装だった。今回の変更で本質的に変わるのは、KVGridの初期`cursorCol`が`colEnabled`(チェックボックス列)であるため、Level 1に入った直後にチェックボックス列がハイライトされ、Key/Valueへの導線が分かりにくかった点である。この点は今回、Level 1入場時の初期カーソル列を明示的に扱う(既存挙動を保つか、Keyから始めるかは実装時に決める)。Authタブも同様に、Level 1に入った瞬間からタイプセレクタが常時ハイライトされるよう`focusAuthField()`の呼び出し条件を調整する。

### 5. Methodの視覚的ヒントは表示のみの変更とする
インタラクション(`h`/`l`で直接値変更)は変えず、`View()`内のMethod表示を`◀ GET ▶`のように左右矢印を付けて描画する。フォーカスされていない時は矢印を出さない(誤って全ゾーンで矢印が出て情報過多にならないようにする)。

### 6. フッターヒントの動的化は`footerHint()`の分岐を拡張する形で実装する
現在`footerHint()`は`s.focus`と`s.editor.Editing()`の2値だけを見ている。これを、`Editor`が現在のサブタブ・内部状態(Level 0/1/2のどこにいるか、KVGridなら行/列移動中かセル編集中か、Authならどのフィールドか)を問い合わせられるgetterを介して取得し、状態ごとに異なるヒント文字列を返すよう拡張する。`Editor`自身にヒント文字列生成ロジックを持たせるか、`Shell`側で状態を問い合わせて組み立てるかは実装時の詳細判断とする。

## Risks / Trade-offs

- [Risk] `h`/`l`の意味が階層によって変わることが、今回の変更以前の挙動を覚えているユーザーには混乱を招く可能性がある → Mitigation: `?`ヘルプとフッターヒントの両方を今回合わせて更新し、常に「今どのキーが何をするか」が画面に出ている状態にする
- [Risk] Optionsタブの追加により、既存の`internal/tui/form`のキー処理テスト(`[`/`]`や`h`/`l`のアサーションを含むもの)の多くが変更対象になり、実装工数がUIの見た目以上に大きくなる可能性がある → Mitigation: `tasks.md`でテスト修正を独立したタスクとして明示し、既存テストの意図(何を守るためのテストか)を壊さないよう1つずつ確認する
- [Risk] KVGridの初期カーソル列を変更する場合、既存のE2E/ユニットテストで前提としているカーソル位置がずれる → Mitigation: 変更するかどうかは実装時にテストへの影響を見て判断する(design時点では方針を確定しない)

## Migration Plan

既存の`.http`ファイルやenvironmentファイルのフォーマットは変更しない。UIとキー操作のみの変更であり、データマイグレーションは不要。既存の`@stream`付きリクエストは、今回の修正後は`[0] Request`パネルで操作しても`@stream`が保持されるようになる(修正前は操作すると失われていたため、実質的な挙動改善)。

## Open Questions

- Level 1入場時のKVGrid初期カーソル列を`colEnabled`のままにするか`colKey`に変えるかは未確定(実装時に既存テストとの整合を見て決める)
- Optionsタブの4行のうち、Timeoutのテキスト入力の検証(不正な値の扱い)は既存の`timeoutSeconds()`のエラーハンドリング方針に委ねる想定だが、フォーム側でのフィードバック方法は未確定
