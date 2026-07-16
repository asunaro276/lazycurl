## Why

Params/HeadersのKVGrid編集で、非編集時に選択中の列(Key/Value)がどこにも表示されない。加えて行を新規作成した直後は列カーソルがValue列に固定されたまま戻らないため、既存の行を再編集しようとすると常にValueから編集が始まり、Keyを編集できないように見える(実際は`h`/`left`/`shift+tab`で列移動できるが、その手がかりが画面上に一切ない)。

## What Changes

- KVGridの非編集時、選択中の行だけでなく選択中の**列**(Key or Value)が視覚的に分かるようにする
- 行を新規作成しValueの編集を確定した直後も、以降その行/他の行に対して迷わずKey列に戻れる状態にする(列カーソルの可視化により解消。列カーソル自体のリセットは行わない)

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `request-editor`: 「フォームベースのリクエスト編集」要求のKey/Valueグリッド編集シナリオに、選択中セルの可視性に関する記述を追加する

## Impact

- `internal/tui/form/kvgrid.go`: `View()`のセルハイライトロジックを行単位から列(セル)単位に変更する
