## Why

`[0] Request`パネルのフォーム操作には、キー入力の意味づけとフィードバックに複数の実用上の問題がある。特に重大なのは、`form.Editor`が`Pragmas.Stream`を往復させていないため、`@stream`付きリクエストをパネルで一度でも操作すると`@stream`が消え、ストリーミング送信が事実上使えなくなっているバグである。加えて、そもそも4つのプラグマ(`@stream`/`@insecure`/`@timeout`/`@no-redirect`)をUIから設定する手段が存在しない。これに、タブ切替キーの分かりにくさ・カーソル位置が見えないタイミング・キーバインドヒント不足といった一連の操作性issueが重なり、フォームでの編集体験を損なっている。

## What Changes

- `form.Editor`が`Pragmas.Stream`を他の3プラグマ同様に`FromRequest`/`ToRequest`で往復させるよう修正し、フォーム操作で`@stream`が失われるバグを解消する
- Params/Headers/Auth/Bodyに続く5番目のタブ「Options」を新設し、4つのプラグマ(Stream/Insecure/NoRedirect/Timeout)をフォームから直接編集可能にする
- フォームのキー操作を3階層モデル(Level 0: ゾーン/タブ選択、Level 1: タブ内部のナビゲーション、Level 2: セル/フィールドのテキスト編集)に整理する
  - Params/Headers/Auth/Body/Optionsのタブ切替を`[`/`]`から、Contentゾーンにいる間の`h`/`l`に変更する
  - Level 1(タブの中)に入った時点で、KVGridの列カーソルやAuthのタイプセレクタを、追加のキー入力なしに常時ハイライト表示する
  - Params/HeadersのKVGridで行が0件のとき、Level 1で`enter`を押すと新規行作成+即編集開始になる(`a`はそのまま有効)
- Methodのインタラクション(`h`/`l`で直接値変更)は変更せず、表示を`◀ GET ▶`のように左右矢印付きにして視覚的に変更可能であることを示す
- フッターのキーバインドヒントを、フォームの`editing`真偽値だけでなく現在のサブタブとその内部状態まで見て動的に生成するよう拡張する

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `request-editor`: プラグマ(Stream/Insecure/NoRedirect/Timeout)をフォームから編集できることが要件として追加される。タブ切替・カーソル表示・キー階層・Methodの視覚表現・フッターヒントに関する要件が変更される。

## Impact

- `internal/tui/form/editor.go`: プラグマ用フィールドの追加、`FromRequest`/`ToRequest`の修正、Optionsタブの追加、Method/タブ切替のキー処理変更、View()の描画変更
- `internal/tui/form/kvgrid.go`: フォーカス直後の常時ハイライト、行0件時の`enter`挙動
- `internal/tui/shell/view.go`: `footerHint()`の動的化
- `openspec/specs/request-editor/spec.md`: 上記の要件変更を反映するdelta
- 既存の`internal/tui/form`・`internal/tui/shell`のテスト(キー操作のアサーションを含むもの)の一部修正が必要になる見込み
