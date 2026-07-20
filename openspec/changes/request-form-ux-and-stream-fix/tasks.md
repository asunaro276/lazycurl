## 1. プラグマ往復バグの修正

- [ ] 1.1 `internal/tui/form/editor.go`の`Editor`構造体に`Pragmas.Stream`用フィールド(`pragStream bool`)を追加する
- [ ] 1.2 `FromRequest`で`req.Pragmas.Stream`を`e.pragStream`に読み込む
- [ ] 1.3 `ToRequest`で`e.pragStream`を`Pragmas.Stream`に書き戻す
- [ ] 1.4 `internal/tui/form`に、`FromRequest` → (何らかのキー操作を経て) → `ToRequest`の往復で`Pragmas.Stream`が保持されることを確認するユニットテストを追加する
- [ ] 1.5 `internal/tui/shell`に、`@stream`付きリクエストをロードしフォームを操作(ナビゲーションのみ)してから`ctrl+r`すると`beginStreamingSend`経路に入ることを確認するテストを追加する(既存の`shell_test.go`のストリーミング関連テストを参考にする)

## 2. Optionsタブの追加

- [ ] 2.1 `internal/tui/form/editor.go`の`tab`定数に`TabOptions`を追加し、`tabLabels`に`"Options"`を追加する
- [ ] 2.2 Stream/Insecure/NoRedirect/Timeoutの4行を表示する専用ビュー(`viewOptions()`)を実装する。4項目とも同じ見た目の行とし、Stream/Insecure/NoRedirectはチェックボックス風、Timeoutは同じ行の見た目でテキスト入力欄を持つ
- [ ] 2.3 Options内部のカーソル移動(`j`/`k`で4行の間を移動)と選択中行の常時ハイライトを実装する
- [ ] 2.4 チェックボックス行(Stream/Insecure/NoRedirect)の`enter`/`space`によるトグルを実装する
- [ ] 2.5 Timeout行の`enter`によるテキスト編集開始・確定・キャンセル(`esc`)を実装する
- [ ] 2.6 `View()`の`switch e.tab`にOptionsタブの分岐を追加する
- [ ] 2.7 Optionsタブの表示・トグル・入力に対するユニットテストを追加する

## 3. キー操作の3階層整理

- [ ] 3.1 `updateNormal`のタブ切替キーを`[`/`]`から`h`/`l`(`left`/`right`)に変更する(Level 0、Contentゾーンにフォーカスがある時のみ)
- [ ] 3.2 `updateNormal`のMethod変更(`h`/`l`)はインタラクションを維持したまま、視覚的ヒントの実装(タスク5)と整合させる
- [ ] 3.3 既存の`[`/`]`キーのテスト・アサーションを`h`/`l`に更新する
- [ ] 3.4 Params/HeadersタブでLevel 1(グリッドナビゲーション)にいる間の`h`/`l`(列移動)がLevel 0のタブ切替と衝突しないことを確認するテストを追加する(Level 1では`h`/`l`が列移動として機能し、`esc`でLevel 0に戻ってから`h`/`l`がタブ切替として機能することを確認)
- [ ] 3.5 Authタブでの`h`/`l`(タイプ選択)についても同様に、Level 1内でのみ機能し、タブ切替と衝突しないことを確認するテストを追加する

## 4. カーソル・セレクタの常時ハイライト

- [ ] 4.1 Params/Headersタブで`enter`によりLevel 1(KVGridの`focused`)に入った直後、追加のキー入力なしにカーソルセルがハイライト表示されることを確認・調整する(既存の`KVGrid.focused`ベースの描画ロジックを確認し、初期`cursorCol`の扱いを決める)
- [ ] 4.2 Authタブで`enter`によりLevel 1(タイプセレクタ)に入った直後、追加のキー入力なしにタイプセレクタがハイライト表示されるよう`focusAuthField()`呼び出し条件を調整する
- [ ] 4.3 上記2点の見た目をユニットテスト(`View()`の出力に対するアサーション)で確認する

## 5. 空グリッドでのenterによる新規行作成

- [ ] 5.1 `KVGrid.Update`で、行が0件かつLevel 1(`focused`)の状態で`enter`が押された場合に、`a`キーと同じ挙動(新規行作成+Key列の編集開始)になるよう分岐を追加する
- [ ] 5.2 既存の「行が0件のとき`enter`は何もしない」前提のテストがあれば、新しい挙動に合わせて更新する
- [ ] 5.3 空グリッドで`enter`を押すケースのユニットテストを追加する

## 6. Methodの視覚的ヒント

- [ ] 6.1 `Editor.View()`のMethod表示を、Level 0でMethodにフォーカスがある時のみ`◀ GET ▶`のように左右矢印付きで描画するよう変更する
- [ ] 6.2 フォーカスが無い時は矢印を出さない(既存のボーダー色による表現のみ)ことを確認する
- [ ] 6.3 見た目の変更をユニットテスト(`View()`の出力に対するアサーション)で確認する

## 7. フッターキーバインドヒントの動的化

- [ ] 7.1 `Editor`に、現在の状態(サブタブ、Level 0/1/2のどこにいるか、KVGridの行/列移動中かセル編集中か、Authのどのフィールドか)を`Shell`側から問い合わせられるgetterを追加する
- [ ] 7.2 `internal/tui/shell/view.go`の`footerHint()`を、`s.editor.Editing()`の2値分岐から、上記getterの結果に応じた多分岐に拡張する
- [ ] 7.3 Params/Headersのグリッドナビゲーション中、セル編集中、Authの各状態、Methodフォーカス中、Optionsタブでの各状態について、それぞれ適切なヒント文字列を実装する
- [ ] 7.4 `viewHelp()`(`?`ヘルプオーバーレイ)にも新しいキー操作(Optionsタブ、`h`/`l`によるタブ切替)を反映する
- [ ] 7.5 フッターヒントの各状態についてユニットテストを追加する

## 8. spec更新・最終確認

- [ ] 8.1 `openspec/specs/request-editor/spec.md`に本changeのdeltaをアーカイブ時に反映できる状態であることを確認する(`openspec validate`で本changeが有効であることを再確認)
- [ ] 8.2 `go build ./...`・`go test ./...`(Docker必須)・`go fmt ./...`が全て通ることを確認する
- [ ] 8.3 `docker compose up`でmockserverを起動し、`@stream`付きリクエストを実際にOptionsタブで有効化してから送信し、レスポンスパネルが逐次更新されることを手動確認する
- [ ] 8.4 Params/Headers/Auth/Body/Optionsの一通りの操作(タブ切替、Level 0/1/2間の遷移、Method変更、フッターヒント)を手動確認する
