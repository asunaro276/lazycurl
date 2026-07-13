## 1. Editorからのname欄除去

- [ ] 1.1 `internal/tui/form/editor.go`の`focusName`ゾーンとName textinputをフォーカス巡回(`FocusNext`/`FocusPrev`/`AtFirstFocus`)から除外する
- [ ] 1.2 `Editor.View()`からName欄の描画を削除する(Method/URLから開始するレイアウトに変更)
- [ ] 1.3 `ToRequest()`/`FromRequest()`の`Name`フィールドの扱いを、フォーム入力値ではなく外部から注入される値として整理する

## 2. 保存時名前プロンプトの実装

- [ ] 2.1 `internal/tui/shell/model.go`に新規`overlay`種別(例: `overlayRequestName`)を追加する
- [ ] 2.2 `saveFormZone()`(Collections)で、対象リクエストの`Name`が空の場合に`overlayRequestName`を開き、既存の`overlayNewCollection`同様`input`スクラッチ文字列で名前を受け取る
- [ ] 2.3 `finishAdhocSave()`または`sendFormZone()`周辺のAdhoc保存経路でも同様に名前未設定時のプロンプトを挟む
- [ ] 2.4 名前入力確定後に実際の保存処理(`.http`ファイルへの書き込み)を継続実行する
- [ ] 2.5 `internal/tui/shell/view.go`に新規オーバーレイの描画(`viewRequestName()`相当)を追加する

## 3. テスト

- [ ] 3.1 `internal/tui/form`: Name欄がフォーカス順に含まれないことを確認するテストを更新
- [ ] 3.2 `internal/tui/shell`: 未命名リクエスト保存時にプロンプトが出ること、既存名リクエストではプロンプトが出ないことを検証するテストを追加
- [ ] 3.3 `go build ./... && go test ./...`が通ることを確認する
