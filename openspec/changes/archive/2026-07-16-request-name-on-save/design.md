## Context

`internal/tui/form/editor.go`の`Editor`は`Name`(`textinput.Model`)を`focusName`という最初のフォーカスゾーンとして常時保持しており、`FocusNext`/`FocusPrev`のフォーカス巡回に組み込まれている。`internal/tui/shell/update.go`の`saveFormZone()`(Collections)・`finishAdhocSave()`(Adhoc)は、保存対象のリクエストが常に名前を持っている前提で動いている。

## Goals / Non-Goals

**Goals:**
- Name欄をフォームの常設フィールドから外し、保存時にのみ名前を確定させる
- 既存リクエスト(既に名前がある)の再保存時はプロンプトを挟まない

**Non-Goals:**
- 保存フロー自体の大規模な再設計(パネル統合やモード撤廃は別change `unified-shell-navigation` で扱う)
- リクエスト名のバリデーションルール変更(重複チェック等は現状維持)

## Decisions

- **保存時プロンプトは新規`overlay`種別として実装する**: 既存の`overlayNewCollection`(コレクション名入力)と同じ「テキスト入力オーバーレイ」のパターンを再利用する。理由: `Shell`は既に`input string`という汎用スクラッチ入力を持っており、同じ仕組みに乗せるのが最小変更。
- **名前が空の場合のみプロンプトを出す**: 毎回名前を聞くと既存リクエストの再保存(頻出操作)が煩雑になるため、`Request.Name == ""`の時だけ割り込む。
- **Name欄はEditorから完全に削除し、Shell側の状態(`requests[idx].Name`/`adhocRequest.Name`)としてのみ保持する**: Editorは「保存対象の名前を知らない」状態になる。代替案として「Editor内にName欄を残しつつ非表示/読み取り専用にする」も検討したが、`inFormZone`のフォーカス管理が複雑化するため採用しない。

## Risks / Trade-offs

- [Risk] 保存のたびに名前チェックが増えることで`ctrl+s`の挙動が「即保存」から「条件分岐」に変わり、ユーザーが送信(`ctrl+r`)と混同する可能性 → Mitigation: プロンプトは名前が空の場合のみ発火するため、命名済みリクエストの操作感は変化しない
- [Risk] `unified-shell-navigation` change が保存フロー(`saveFormZone`/`overlaySaveAdhoc`)を大きく触るため、実装順序を誤るとコンフリクトする → Mitigation: 本changeを先に実装・マージし、`unified-shell-navigation`の設計はName欄が既に存在しない前提で進める
