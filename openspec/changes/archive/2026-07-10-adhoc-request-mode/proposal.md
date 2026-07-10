## Why

現状のlazycurlは、リクエストを1件送るだけでもまずcollectionを作成しなければならない(collectionが無いと新規リクエスト作成すらできない)。しかし実際の使われ方としては、まず手元でcurlリクエストを組み立てて送ってみたい、というのが最初のニーズであり、保存・整理(collection化)やenvironment変数の利用は、その後「再利用したくなったら」初めて必要になる機能である。この優先順位のズレが、起動直後の操作を煩雑にしている。

## What Changes

- TUIに `Adhoc` / `Collections` の2モードを導入し、`[`/`]`キーで切り替えられるようにする。アクティブなモードはハイライト表示する
- `Adhoc`モードでは、collectionを作成・選択しなくても、その場でMethod/URL/Headers/Body編集フォーム(既存のrequest-editorを再利用)を開いてリクエストを組み立て・送信できる
- `Adhoc`モードのレイアウトは 編集フォーム + Response + History の3ペインとする
- `Adhoc`モードで組み立てたリクエストは、保存されるまでメモリ内にのみ存在し、ディスクへは書き込まれない(environment変数展開もcollectionに紐づく機能のため、Adhoc中は行わない)
- `Adhoc`モードから任意のタイミングで`s`キーにより保存を実行できる。保存時は既存collectionの選択、または新規collection作成(既存の`overlayNewCollection`フローを再利用)を行う
- 保存完了後は自動的に`Collections`モードへ切り替わり、保存先collection/リクエストが選択された状態になる
- 実行履歴(History)は`Adhoc`/`Collections`間で共有する(既存の`HistoryEntry.CollectionName`が空文字の場合はAdhocでの実行として扱う)
- `Collections`モードは既存の4パネルレイアウト(Collections/Requests/Response/History)をそのまま維持する。起動時のデフォルトモードは`Adhoc`に変更する

## Capabilities

### New Capabilities
- `adhoc-mode`: collectionを介さずにリクエストを組み立て・送信できるAdhocモード。モード切り替えの仕組み(`[`/`]`、ハイライト表示)、Adhoc画面のレイアウト、メモリ内限定の状態管理、collectionへの保存導線を含む

### Modified Capabilities
- `tui-shell`: 起動時のパネルレイアウト要件を`Collections`モード限定の要件として再定義し、モード切り替え(`[`/`]`)とデフォルト起動モードに関する要件を追加する
- `request-editor`: リクエストの新規作成・編集がcollectionに属さない状態(Adhoc)でも行えるようにし、collectionへの紐づけを保存時まで遅延できるようにする

## Impact

- `internal/tui/shell/`(model.go/update.go/view.go): モード状態の追加、パネルレイアウトの出し分け、`[`/`]`キーハンドリング、保存導線の実装
- `internal/tui/app.go`: `OpenEditorMsg`が空のcollection名(Adhoc由来)を受け取れるようにする対応
- `collection-storage`・`environment-variables`・`curl-execution`の各capabilityには変更なし(既存のCreateCollection/SaveRequests、curl実行フローをそのまま再利用)
