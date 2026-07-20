## 1. プラグマ往復バグの修正

- [x] 1.1 `internal/tui/form/editor.go`の`Editor`構造体に`Pragmas.Stream`用フィールド(`pragStream bool`)を追加する
- [x] 1.2 `FromRequest`で`req.Pragmas.Stream`を`e.pragStream`に読み込む
- [x] 1.3 `ToRequest`で`e.pragStream`を`Pragmas.Stream`に書き戻す
- [x] 1.4 `internal/tui/form`に、`FromRequest` → (何らかのキー操作を経て) → `ToRequest`の往復で`Pragmas.Stream`が保持されることを確認するユニットテストを追加する(`TestEditorPragmaRoundTrip`)
- [x] 1.5 `internal/tui/shell`に、`@stream`付きリクエストをロードしフォームを操作(ナビゲーションのみ)してから`ctrl+r`すると`beginStreamingSend`経路に入ることを確認するテストを追加する(`TestShellFormNavigationPreservesStreamPragma`)

## 2. Optionsタブの追加

- [x] 2.1 `internal/tui/form/editor.go`の`tab`定数に`TabOptions`を追加し、`tabLabels`に`"Options"`を追加する
- [x] 2.2 Stream/Insecure/NoRedirect/Timeoutの4行を表示する専用ビュー(`viewOptions()`)を実装する。4項目とも同じ見た目の行とし、Stream/Insecure/NoRedirectはチェックボックス風、Timeoutは同じ行の見た目でテキスト入力欄を持つ
- [x] 2.3 Options内部のカーソル移動(`j`/`k`で4行の間を移動)と選択中行の常時ハイライトを実装する
- [x] 2.4 チェックボックス行(Stream/Insecure/NoRedirect)の`enter`/`space`によるトグルを実装する
- [x] 2.5 Timeout行の`enter`によるテキスト編集開始・確定・キャンセル(`esc`)を実装する
- [x] 2.6 `View()`の`switch e.tab`にOptionsタブの分岐を追加する
- [x] 2.7 Optionsタブの表示・トグル・入力に対するユニットテストを追加する(`TestEditorOptionsTogglesPragmas`)

## 3. キー操作の3階層整理

- [x] 3.1 `updateNormal`のタブ切替キーを`[`/`]`から`h`/`l`(`left`/`right`)に変更する(Level 0、Contentゾーンにフォーカスがある時のみ)
- [x] 3.2 `updateNormal`のMethod変更(`h`/`l`)はインタラクションを維持したまま、視覚的ヒントの実装(タスク6)と整合させる
- [x] 3.3 既存の`[`/`]`キーのテスト・アサーションを`h`/`l`に更新する(`TestEditorTabSwitchOnlyInNormalState`, `TestEditorTabSwitchWrapsThroughAllFiveTabs`)
- [x] 3.4 Params/HeadersタブでLevel 1(グリッドナビゲーション)にいる間の`h`/`l`(列移動)がLevel 0のタブ切替と衝突しないことを確認するテストを追加する(`TestEditorGridColumnMoveDoesNotLeakIntoTabSwitch`)
- [x] 3.5 Authタブでの`h`/`l`(タイプ選択)についても同様に、Level 1内でのみ機能し、タブ切替と衝突しないことを確認するテストを追加する(`TestEditorAuthTypeSelectDoesNotLeakIntoTabSwitch`)

## 4. カーソル・セレクタの常時ハイライト

- [x] 4.1 Params/Headersタブで`enter`によりLevel 1(KVGridの`focused`)に入った直後、追加のキー入力なしにカーソルセルがハイライト表示されることを確認・調整する。既存の`KVGrid.focused`ベースの描画ロジックはそのまま活用し、`NewKVGrid`の初期`cursorCol`を`colEnabled`から`colKey`に変更した(Level 1に入って最初に触れるのがチェックボックスではなくKey入力になるように)
- [x] 4.2 Authタブで`enter`によりLevel 1(タイプセレクタ)に入った直後、追加のキー入力なしにタイプセレクタがハイライト表示されることを確認した。`viewAuth()`のハイライト条件(`authField==0 && focus==focusContent`)は既にLevel1進入時点で真になっており、追加の実装変更は不要だった
- [x] 4.3 上記2点の見た目をユニットテスト(`View()`の出力に対するアサーション)で確認する(`TestEditorLevel1HighlightsImmediatelyOnEntry`)

## 5. 空グリッドでのenterによる新規行作成

- [x] 5.1 `KVGrid.Update`で、行が0件の状態で`enter`が押された場合に、`a`キーと同じ挙動(新規行作成+Key列の編集開始)になるよう分岐を追加する(共通の`addRow()`ヘルパーとして`a`とenterの両方から呼ぶ形に整理)
- [x] 5.2 既存の「行が0件のとき`enter`は何もしない」前提のテストがあれば、新しい挙動に合わせて更新する(該当する既存テストは無かったことをgrepで確認済み)
- [x] 5.3 空グリッドで`enter`を押すケースのユニットテストを追加する(`TestKVGridEnterOnEmptyGridAddsRow`)

## 6. Methodの視覚的ヒント

- [x] 6.1 `Editor.View()`のMethod表示を、Level 0でMethodにフォーカスがある時のみ`◀ GET ▶`のように左右矢印付きで描画するよう変更する
- [x] 6.2 フォーカスが無い時は矢印を出さない(既存のボーダー色による表現のみ)ことを確認する
- [x] 6.3 見た目の変更をユニットテスト(`View()`の出力に対するアサーション)で確認する(`TestEditorMethodShowsArrowsWhenFocused`)

## 7. フッターキーバインドヒントの動的化

- [x] 7.1〜7.3 `Editor`に、現在の状態(サブタブ、Level 0/1/2、KVGridの行/列移動中かセル編集中か、Authのどのフィールドか)に応じたヒント文字列そのものを返す`FooterHint()`メソッドを実装した(design.mdで示した2案のうち「Editor自身にヒント文字列生成ロジックを持たせる」案を採用し、個別のgetter群は追加していない)
- [x] 7.2 `internal/tui/shell/view.go`の`footerHint()`を`s.editor.FooterHint()`への委譲に置き換えた
- [x] 7.4 `viewHelp()`(`?`ヘルプオーバーレイ)に新しいキー操作(Level 0/1の説明、Options込み5タブ、`h`/`l`によるタブ切替)を反映する
- [x] 7.5 フッターヒントの各状態についてユニットテストを追加する(`TestEditorFooterHintReflectsLevelAndTab`)

## 8. spec更新・最終確認

- [x] 8.1 `openspec/specs/request-editor/spec.md`に本changeのdeltaをアーカイブ時に反映できる状態であることを確認する(`openspec validate`で本changeが有効であることを再確認済み)
- [x] 8.2 `go build ./...`・`go vet ./...`・`gofmt -l`は全て通過を確認済み。`go test ./...`は`internal/tui/form`・`internal/tui/shell`を含む非Docker依存パッケージは全てパス。`internal/curlexec`・`internal/tui`のE2Eスイートは、本サンドボックス環境にDockerデーモンが実際には起動していないため実行不可(CLAUDE.mdに記載の既知の受容済みギャップで、本changeによる regression ではない)
- [ ] 8.3 `docker compose up`でmockserverを起動し、`@stream`付きリクエストを実際にOptionsタブで有効化してから送信し、レスポンスパネルが逐次更新されることを手動確認する(Docker利用可能な環境での確認が必要、本セッションでは未実施)
- [ ] 8.4 Params/Headers/Auth/Body/Optionsの一通りの操作(タブ切替、Level 0/1/2間の遷移、Method変更、フッターヒント)を手動確認する(TUIを実際に操作できる環境での確認が必要、本セッションでは未実施)
