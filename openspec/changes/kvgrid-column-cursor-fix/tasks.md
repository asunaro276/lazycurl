## 1. セル単位ハイライトの実装

- [ ] 1.1 `internal/tui/form/kvgrid.go`の`View()`を、行全体の反転ではなくKey/Valueそれぞれのテキストを個別にレンダリングするよう変更する
- [ ] 1.2 `cursorCol == colKey`のとき`pad(key, 20)`部分のみ、`cursorCol == colValue`のとき`value`部分のみを`styleSelected`でハイライトする
- [ ] 1.3 `colEnabled`選択時の見た目(現状のcheckbox表示)は変更しない

## 2. テスト

- [ ] 2.1 `internal/tui/form/kvgrid_test.go`に、非編集時のKey列選択/Value列選択それぞれで対応するセルのみがハイライトされることを検証するテストを追加
- [ ] 2.2 既存の`View()`関連テストが新しい出力形式に合わせて通ることを確認する
- [ ] 2.3 `go build ./... && go test ./...`が通ることを確認する
