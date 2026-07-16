## 1. 合成描画ヘルパーの実装(internal/tui/shell/view.go)

- [x] 1.1 `github.com/charmbracelet/x/ansi`を直接importに追加し、`go.mod`を`go mod tidy`で同期する
- [x] 1.2 背景文字列とオーバーレイ文字列を受け取り、オーバーレイを中央配置で合成した文字列を返すヘルパー関数(例: `compositeOverlay(background string, overlay string, termWidth, termHeight int) string`)を実装する。行単位に分解し、各行を`ansi.Cut`で左右に分割してオーバーレイ行を差し込む
- [x] 1.3 オーバーレイの実レンダリングサイズ(幅・高さ、パディング/ボーダー込み)が`termWidth`/`termHeight`に収まるかを判定するロジックを実装し、収まらない場合はオーバーレイ単体の文字列をそのまま返す(現状のフルスクリーン表示)ようにする
- [x] 1.4 全角文字(日本語のUI文言)を含む行での列幅計算が正しいことを確認する(必要に応じて`ansi.CutWc`系を使い分ける)

## 2. 背景パネルのdimmedスタイル対応

- [x] 2.1 `panelBorderFocused`/`panelBorderUnfocused`と対になる、彩度を落とした`panelBorderDimmed`相当のスタイル(ボーダー色・テキスト色)を追加する
- [x] 2.2 `renderPanel`/`viewGrid`に`dimmed bool`引数を追加し、trueの場合はdimmedスタイルで描画するようにする
- [x] 2.3 既存の呼び出し箇所(`View()`内の`main`生成部分)を、オーバーレイ非表示時は`dimmed=false`で呼び出すよう更新する

## 3. View()の書き換え

- [x] 3.1 `View()`のオーバーレイ分岐を、`main`を捨てる現状の実装から、「`dimmed=true`で再描画した背景」+「オーバーレイ内容」を1.2のヘルパーで合成する実装に置き換える
- [x] 3.2 オーバーレイがフルスクリーンにフォールバックするケース(1.3の判定結果)では、既存同様オーバーレイ内容のみを返すようにする
- [x] 3.3 `overlay` enumの値や`handleOverlayKey`など、キー処理・状態遷移ロジックには一切手を入れないことを確認する

## 4. テスト

- [x] 4.1 `internal/tui/shell`: オーバーレイ表示中の`View()`出力に、背景4パネルの内容(dimmedスタイル適用後の断片を含む)とオーバーレイ内容の両方が含まれることを検証するテストを追加する
- [x] 4.2 `internal/tui/shell`: 端末サイズを縮小した状態でオーバーレイを表示し、フルスクリーンフォールバックが発生する(背景パネルの内容が出力に含まれない)ことを検証するテストを追加する
- [x] 4.3 `internal/tui/shell`: 既存のオーバーレイ関連テスト(`s.overlay`の値の遷移を検証するテスト)が引き続き通ることを確認する
- [x] 4.4 `go build ./... && go test ./...`が通ることを確認する

## 5. 手動確認

- [x] 5.1 実ターミナルでヘルプ/新規collection作成/削除確認の各オーバーレイを表示し、背景パネルがdimmedで見えること、日本語文言が正しい位置に表示されることを目視確認する(tmux capture-paneで確認。残りのenvSelect/saveTo/requestNameオーバーレイも同じ`currentOverlayBody()`/`compositeOverlay`経路を通るため、描画ロジック自体は共通で検証済み)
- [x] 5.2 ターミナルを意図的に縮小し、フルスクリーンフォールバックが発生することを目視確認する
