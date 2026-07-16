## Why

`internal/tui/shell`の`overlay`(help/envSelect/newCollection/saveTo/requestName/confirmDelete)は、名称上は「パネルの上に重なるモーダル」だが、`View()`の実装は`s.viewGrid()`が返す4パネルの描画を丸ごと捨てて`overlayBox`の内容だけを返している。結果として、overlay表示中は4パネルが画面から完全に消え、実質的にモード切り替え時代と同じ「画面全体の差し替え」が起きている。これは「一切ページ切り替えをせず、パネル4つを常に表示する」というシェルの前提と矛盾しており、`View()`のコメント(「with any active overlay drawn on top」)が主張する挙動とも食い違っている。

## What Changes

- 背景の4パネル(`viewGrid()`の出力)を行単位で分解し、`charmbracelet/x/ansi`の`ansi.Cut`(既存のindirect依存。新規追加不要)を使って各行にoverlayの内容を差し込む合成描画ヘルパーを追加する
- overlayは常に画面中央に配置する
- overlay表示中、背景の4パネルは彩度を落とした`dimmed`スタイル(ボーダー色・テキスト色の別バリアント)で再描画する。既に描画済みのANSI文字列へ事後的に`faint`等を適用する方式は端末依存で信頼できないため採用しない
- overlay自体の実レンダリングサイズ(内容量に応じて可変)が現在の端末サイズに収まらない場合は、そのoverlayに限り現状と同じ「4パネルを消してoverlayのみ表示する」フルスクリーン表示にフォールバックする
- `overlay` enum(`overlayHelp`/`overlayEnvSelect`/`overlayNewCollection`/`overlayConfirmDelete`/`overlaySaveTo`/`overlayRequestName`)、および`handleOverlayKey`によるキー処理ロジックは変更しない。修正は描画(`view.go`)側に閉じる

## Capabilities

### New Capabilities

(なし)

### Modified Capabilities

- `tui-shell`: overlay表示時のレイアウト要求として、「4パネルを背景に残したまま中央に重ねて表示する」「背景はdimmedスタイルで表示する」「端末が狭くoverlayが収まらない場合はフルスクリーンにフォールバックする」という新しい要求を追加する

## Impact

- `internal/tui/shell/view.go`: `View()`のoverlay分岐、`renderPanel`/`viewGrid`(dimmedスタイル対応)、新規の合成描画ヘルパー関数
- `internal/tui/shell/model.go`: 合成描画に必要な状態(dimmedスタイル定義など)を追加する可能性がある。`overlay` enumやその他の状態フィールドは変更しない
- 依存関係の変更なし(`charmbracelet/x/ansi`は既存のindirect依存を直接importに格上げするのみ)
- 関連する既知の課題(参考情報。本changeのスコープ外): 別change `unified-shell-navigation`は既にmainへマージ済みだが、archive後処理が未実施で`openspec/specs/tui-shell/spec.md`が「モード」概念前提の古い記述のままになっている。本changeの仕様差分はこの点と競合しないよう、既存要求の文言を書き換えず新規要求の追加のみで構成する
