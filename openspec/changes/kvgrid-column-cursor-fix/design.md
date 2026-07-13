## Context

`internal/tui/form/kvgrid.go`の`KVGrid.View()`は非編集時、選択行全体を`styleSelected.Render(line)`で反転表示するのみで、`cursorCol`(colEnabled/colKey/colValue)がどこを指しているかは描画に反映されていない。`cursorCol`は行を跨いでも保持され続け、行作成直後の自動遷移(`colKey`→`colValue`)によって以降ずっとValue列に留まりやすい。

## Goals / Non-Goals

**Goals:**
- 非編集時に選択中のセル(Key or Value)を視覚的に区別できるようにする

**Non-Goals:**
- `cursorCol`の自動リセット等、カーソル移動ロジック自体の変更(可視化のみで十分に問題が解消するため)
- Enabledチェックボックス列のセル単位ハイライト変更(現状のcheckbox表示で十分視認できるため対象外)

## Decisions

- **行全体の反転ハイライトはやめ、Key/Valueそれぞれのテキスト部分だけを個別にハイライトする**: `pad(key, 20)`と`value`をそれぞれ独立した文字列として構築し、`cursorCol`が指す方だけに`styleSelected`を適用する。Enabledチェックボックス部分は現状の表示のまま(反転させない)。
- **`cursorCol`のリセットは行わない**: 列カーソルは可視化されればユーザーは`h`/`l`で移動できることが分かるため、行移動時に強制的にKey列へ戻す変更は不要。挙動を変えず表示だけ直すことでリグレッションリスクを抑える。

## Risks / Trade-offs

- [Risk] セル単位ハイライトへの変更で行全体のハイライトに依存する既存テストが壊れる可能性 → Mitigation: `kvgrid_test.go`の`View()`出力アサーションを確認し、必要なら期待値を更新する
