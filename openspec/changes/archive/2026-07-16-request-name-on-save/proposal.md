## Why

現在`form.Editor`はName欄を常時表示・編集可能なフィールドとして持っており、Collections/Adhocいずれのモードでも編集中から名前入力を求められる。しかしAdhocモードでは保存するまでリクエストに名前は本質的に不要であり、Collectionsモードでも「編集中は仮の状態、保存する瞬間に初めて名前を確定する」方が自然な体験になる。

## What Changes

- リクエストのName入力を、フォーム常設フィールドから廃止し、保存操作(`ctrl+s`)時のプロンプトに移す **BREAKING**(既存のフォームフォーカス順からName欄が消える)
- 名前が空のまま保存しようとした場合、名前入力を促すオーバーレイを表示してから保存を完了する
- 既に名前が付いている既存リクエストを保存する場合は、従来通りプロンプトなしでそのまま保存する(名前を毎回聞き直さない)
- Adhocモードでは保存しない限り名前は一切要求されず、無名のまま編集・送信が完結する

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `request-editor`: リクエストのName入力手段を「フォーム常設欄」から「保存時プロンプト」へ変更する要求を追加・既存要求を修正する

## Impact

- `internal/tui/form/editor.go`: Name textinputおよびフォーカス順(`focusName`起点のFocusNext/FocusPrev)からName欄を除外
- `internal/tui/shell/update.go`: `saveFormZone()`(Collections)・`finishAdhocSave()`(Adhoc)の保存経路に、名前が空の場合の名前入力オーバーレイ(新規`overlay`種別)を追加
- `internal/tui/shell/view.go`: 新規オーバーレイの描画、フォーム内Name欄表示の削除
