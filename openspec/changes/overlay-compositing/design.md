## Context

`internal/tui/shell/view.go`の`Shell.View()`は次の形になっている。

```go
func (s *Shell) View() string {
	main := lipgloss.JoinVertical(lipgloss.Left, s.viewGrid(), s.viewStatusBar())

	switch s.overlay {
	case overlayHelp:
		return overlayBox.Render(s.viewHelp())
	...
	}
	return main
}
```

`s.overlay != overlayNone`の間、`main`(4パネル+ステータスバー)は一切参照されず、`overlayBox.Render(...)`だけが返る。つまり現状のoverlayは「モーダル」ではなく「画面全体の一時的な差し替え」であり、`unified-shell-navigation`が撤廃したはずの「モードによる画面専有」が、overlay表示中に限って復活してしまっている。

lipgloss(v1.1.0)自体には、既に色付けされた文字列同士を座標指定で重ね合わせる機能はない。一方、lipglossの間接依存である`charmbracelet/x/ansi`には、表示幅ベースでANSIエスケープを壊さずに文字列を切り出せる`ansi.Cut(s string, left, right int) string`が存在し、これを使えば背景行の任意の列範囲を安全に置き換えられる。

## Goals / Non-Goals

**Goals:**
- overlay表示中も背景の4パネルを(dimmedスタイルで)描画し続け、overlayをその上に中央配置で重ねる
- overlay自体が現在の端末サイズに収まらない場合は、そのoverlayに限り現状同様のフルスクリーン表示にフォールバックする
- `overlay` enumの種類・`handleOverlayKey`によるキー処理・状態遷移(`s.overlay`の値の変化)は一切変更しない。描画層のみの修正に閉じる

**Non-Goals:**
- overlayの内容(ヘルプ一覧のテキスト、保存先一覧のUIなど)自体の変更は行わない
- モーダル表示のアニメーションやトランジション効果は扱わない
- `overlay` enumの追加・削除(新しいoverlay種別の追加)は本changeの対象外

## Decisions

### 1. 合成方式: `ansi.Cut`による行単位の差し込み

背景(`viewGrid()`の出力)を`\n`で行分割し、overlayボックス(`overlayBox.Render(...)`の出力)も行分割する。overlayの各行について、対応する背景行を`ansi.Cut(bgLine, 0, offsetX)` / `ansi.Cut(bgLine, offsetX+overlayWidth, bgWidth)`で左右に分割し、`左 + overlay行 + 右`として結合する。`offsetX`/`offsetY`はoverlayを画面中央に置くための計算値。

**代替案として検討したもの:**
- 独自にANSIエスケープをパースする合成ロジックを新規実装する → `x/ansi`が既に依存関係に含まれており、車輪の再発明になるため不採用
- overlay表示中は背景を単色の空白で塗りつぶす(パネル内容を出さない) → 「パネルを常に表示する」という前提(今回のexploreの出発点)を満たさないため不採用

### 2. dimmedスタイル: 再描画によるスタイル切り替え

overlay表示中、背景パネルを暗く見せる方法として、「既に描画済みのANSI文字列に`lipgloss.NewStyle().Faint(true)`等を事後適用する」方式と、「`renderPanel`/`viewGrid`に`dimmed bool`を渡し、ボーダー色・テキスト色を別のグレー系スタイルセットに切り替えて再描画する」方式を比較した。

前者は、パネル内部で既にmethodバッジやステータスバッジなどが個別の前景色エスケープシーケンスを持っており、その上から`faint`(SGR 2)を重ねても多くの端末で視覚的にほぼ変化しない、あるいは端末によって挙動が異なる。後者は常に同じ結果になり、既存の`panelBorderFocused`/`panelBorderUnfocused`のような「スタイルバリアントを切り替える」既存パターン(`renderPanel`が`s.focus == p`で切り替えているのと同じ考え方)にも合う。よって後者(再描画方式)を採用する。

具体的には、`panelBorderFocused`/`panelBorderUnfocused`と対になる`panelBorderDimmed`のようなグレー系スタイルを追加し、`viewGrid(dimmed bool)`/`renderPanel(p Panel, w, h int, content string, dimmed bool)`のように、overlay有無を引数として渡す形にする。

### 3. フォールバック判定: overlay種類ごとの実サイズ比較(コンテンツ依存方式)

固定の端末サイズ閾値(例: 80x24未満なら常にフルスクリーン)ではなく、各overlayが実際にレンダリングされた幅・高さ(`lipgloss.Width`/`lipgloss.Height`相当、パディング・ボーダー込み)を計算し、現在の端末幅・高さと比較する。収まらない場合はそのoverlayに限りフルスクリーン表示(現状の`overlayBox.Render(...)`のみを返す既存挙動)にフォールバックする。

これにより、内容が長いoverlay(例: helpの一覧)は比較的広い端末でもフルスクリーン化しやすく、内容が短いoverlay(例: confirmDelete)はより狭い端末でも中央合成表示を維持できる。overlay種類ごとに固定の必要サイズを持たせず、実際にレンダリングした結果のサイズで都度判定するため、overlayの文言を将来変更してもこの判定ロジック自体の変更は不要。

## Risks / Trade-offs

- [ANSIを含む文字列の列範囲切り出しは、全角文字(日本語のUI文字列を含む)を含む場合に表示幅の計算を誤りやすい] → `ansi.Cut`は表示幅(セル幅)ベースで動作するため、`ansi.CutWc`系の使い分けも含め、日本語テキスト(ヘルプ文言・ステータスバッジ等)を実際にレンダリングして確認する
- [背景パネルを再描画する分、overlay表示中の`View()`の呼び出しコストがわずかに増える] → TUIの描画頻度・パネル4枚程度のサイズでは無視できる範囲と判断し、最適化は行わない
- [dimmedスタイルの色選定によっては、フォーカス中パネルとの区別が視認しづらくなる可能性] → 既存の`panelBorderFocused`(色コード62)・`panelBorderUnfocused`(色コード240)から、さらに彩度を落とした色を選定し、実際にターミナルで見た目を確認する

## Migration Plan

破壊的なデータ移行は発生しない(表示ロジックのみの変更)。既存のテスト(`shell_test.go`)は`s.overlay`の値のみを検証しているため影響を受けないが、新規に「overlay表示中もパネルの内容が(dimmedで)出力に含まれること」「フォールバック時はパネル内容が出力に含まれないこと」を検証するテストを追加する。

## Open Questions

- dimmedスタイルの具体的な色コードは実装時に実ターミナルで確認しながら決定する
