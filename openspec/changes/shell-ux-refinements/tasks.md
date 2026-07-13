## 1. ステータスバーの自動クリア

- [x] 1.1 `internal/tui/shell/model.go` の `Shell` に `statusGen int` フィールドを追加する
- [x] 1.2 `internal/tui/shell/update.go` に `setStatus(msg string) tea.Cmd` ヘルパーを追加する。`statusMsg`/`statusGen` を更新し、`tea.Tick(5*time.Second, ...)` で現在の世代を積んだ `clearStatusMsg` を返すコマンドを生成する
- [x] 1.3 `update.go` 内の既存の `s.statusMsg = ...` という直接代入箇所(`sendResultMsg` 処理、`handleCollectionsKey`、`handleRequestsKey`、`handleOverlayKey`、`sendCurrent`、`sendAdhocCurrent`、`finishAdhocSave` など)をすべて `s.setStatus(...)` 経由に置き換え、戻り値の `tea.Cmd` を呼び出し元の戻り値まで伝播させる(必要に応じて `tea.Batch` で他のコマンドと合成する)
- [x] 1.4 `Shell.Update` に `clearStatusMsg` メッセージのケースを追加し、`msg.gen == s.statusGen` の場合のみ `statusMsg` を空文字にクリアする
- [x] 1.5 単体テストを追加する: エラー表示から5秒後に自動でクリアされること、5秒以内に新しいメッセージが来た場合は古いタイマーでクリアされないこと

## 2. フォームのサブセクション切り替えキーの変更

- [x] 2.1 `internal/tui/form/editor.go` の `Update` 内、`case "1", "2", "3", "4"` によるタブ切り替えを `case "[", "]"` による前後切り替えに変更する
- [x] 2.2 `View()` のタブラベル表示・ヒント文言を新しいキーに合わせて更新する(ヘルプ文言はGroup 6でまとめて更新)
- [x] 2.3 既存のテスト(`form/convert_test.go` など)で `1`-`4` キー送信を前提にしている箇所を `[`/`]` に更新する(該当箇所なし、変更不要と確認)

## 3. Shellへのフォーム埋め込みとフォーカスチェーン拡張

- [x] 3.1 `internal/tui/shell/model.go` の `Shell` に `form.Editor` のインスタンスと編集対象の状態(Collectionsモードで編集中のリクエストindex、Adhocモードでは常に1件)を追加する
- [x] 3.2 Collectionsモードの `Requests` パネルに「リストゾーン」(既存の一覧ナビゲーション)と「フォームゾーン」(埋め込みフォーム)の内部フォーカス状態を追加する
- [x] 3.3 Adhocモードの `Editor` パネルは、パネルにフォーカスが移った時点で常にフォームゾーン(Nameフィールド)にフォーカスが当たるようにする
- [x] 3.4 `tab`/`shift+tab` が「リストゾーン ⇔ フォームゾーンの各フィールド ⇔ 前後のShellパネル」を一続きのチェーンとして移動するようにフォーカス制御を実装する
- [x] 3.5 フォームゾーンでのキー入力のたびに `form.Editor.ToRequest()` を呼び出し、`requests[idx]`(Collections)または `adhocRequest`(Adhoc)へ即座に反映する
- [x] 3.6 `Requests`/`Editor` パネルの `View` 実装を、パネル幅(特にCollectionsモードの狭い列)に収まるよう調整する。非選択行はサマリー表示のまま、選択中の行のみフォームを展開表示する

## 4. グローバルキーの迂回と保存キーの付け替え

- [x] 4.1 `internal/tui/shell/update.go` の `handleKey` の先頭で、現在フォーカスがフォームゾーン内かどうかを判定する分岐を追加する
- [x] 4.2 フォームゾーンにフォーカスがある場合、`q`/`s`/`[`/`]`/`1`-`4`/`enter` のグローバルショートカット解釈をスキップし、キーをフォームへ直接ディスパッチする(`ctrl+c` は常にグローバルとして扱う)。送信は `ctrl+r` に一本化する
- [x] 4.3 フォームゾーンでの `ctrl+s` を実装する: Collectionsモードでは `colStore.SaveRequests` で現在の `requests` を書き込み、Adhocモードでは `overlaySaveAdhoc` を起動する
- [x] 4.4 Adhocモードの単独 `s` キーバインドを削除する(`ctrl+s` に統合)
- [x] 4.5 `ctrl+q`(編集破棄)のロジックを削除する
- [x] 4.6 `ctrl+c` がフォームゾーン内でも送信中キャンセル/終了として従来通り機能することを確認する(テスト追加済み)

## 5. `App.modeEditor` / `OpenEditorMsg` の削除

- [x] 5.1 `internal/tui/app.go` から `mode`(`modeEditor`)、`editor`、`editingCol`、`editingIndex`、`statusOverride` フィールドと関連ロジック(`updateEditor`、`saveEditor`、`View`内のmodeEditor分岐)を削除する
- [x] 5.2 `internal/tui/shell` から `OpenEditorMsg` 型と発行箇所(`handleEditorPanelKey`、`handleRequestsKey`の`e`ケース)を削除する
- [x] 5.3 `e` キーによるパネル起動ロジックを削除し、フォーカスチェーン(タスク3)経由でフォームゾーンに到達する導線に一本化する
- [x] 5.4 `internal/tui/app_test.go` のうち `modeEditor`/`OpenEditorMsg` に依存するテストを、新しいインライン編集の導線に合わせて書き換える

## 6. ヘルプ・ステータスバー文言の更新

- [x] 6.1 `internal/tui/shell/view.go` の `viewStatusBar()` のキーバインドヒント文言を新しいキー割り当て(`[`/`]`のセクション切替、`ctrl+s`保存など)に更新する
- [x] 6.2 `viewHelp()` の内容を新しいキー割り当てに更新する
- [x] 6.3 `CLAUDE.md` の `App`/`form.Editor` に関する記述を、実装後のアーキテクチャ(全画面モード廃止、Shellへのフォーム埋め込み)に合わせて更新する

## 7. テストと検証

- [x] 7.1 `go build -o lazycurl ./cmd/lazycurl` が通ることを確認する
- [x] 7.2 `go test ./...` が通ることを確認する
- [x] 7.3 `./lazycurl` を実際に起動し、Adhoc/Collections双方で以下を手動確認する: フォーカス直後の直接編集、`[`/`]`によるセクション切替、`ctrl+s`保存、ステータスバーの5秒自動クリア、フォームゾーン外での`?`ヘルプ表示(tmuxでpty駆動して確認。5秒自動クリアは単体テストで検証済みのため実時間待機はスキップ)
- [x] 7.4 `go fmt ./...` を実行する
