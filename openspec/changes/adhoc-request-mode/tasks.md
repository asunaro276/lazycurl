## 1. モード基盤

- [ ] 1.1 `internal/tui/shell/model.go`に`Mode`型(`ModeAdhoc`/`ModeCollections`)を追加し、`Shell`にモード状態を持たせる
- [ ] 1.2 起動時のデフォルトモードを`ModeAdhoc`にする
- [ ] 1.3 `internal/tui/shell/update.go`に`[`/`]`キーのハンドリングを追加し、`tab`/`shift+tab`(パネル内移動)とは独立してモードを切り替える
- [ ] 1.4 `internal/tui/shell/view.go`でアクティブモードをハイライト表示するモードタブ(または相当する表示)を描画する

## 2. Adhocモードのレイアウトと編集

- [ ] 2.1 `Adhoc`モード用に、編集フォーム+Response+Historyの3ペインレイアウトを`view.go`に実装する
- [ ] 2.2 `Shell`にAdhoc用の未保存スクラッチリクエスト(`httpfile.Request`1件)を保持するフィールドを追加する
- [ ] 2.3 `Adhoc`モードでcollection未作成・未選択でもリクエスト編集フォーム(`OpenEditorMsg`)を開けるようにする
- [ ] 2.4 `internal/tui/app.go`の`OpenEditorMsg`処理で、空のcollection名(Adhoc由来)を扱えるようにする(既存の`LoadRequests`/`SaveRequests`呼び出しをAdhoc時はスキップし、Shell側のスクラッチリクエストを更新する)

## 3. Adhocの一時性とenvironment非対応

- [ ] 3.1 Adhocのスクラッチリクエストがアプリ終了時に(保存されない限り)破棄されることを確認する(永続化コードを書かないことで自然に満たす)
- [ ] 3.2 Adhoc送信時に`environment.ExpandRequest`等の変数展開を呼ばず、生の値でcurlを実行する経路を実装する

## 4. 履歴の共有

- [ ] 4.1 Adhocでの送信結果を、既存の`HistoryEntry`(`CollectionName`を空文字)として`s.history`に追記する
- [ ] 4.2 `Collections`モードのHistoryパネルがAdhoc由来のエントリ(collection名が空)も表示できることを確認する

## 5. collectionへの保存導線

- [ ] 5.1 `Adhoc`モードで`s`キーのハンドリングを追加する
- [ ] 5.2 保存時に既存collection一覧からの選択、または新規collection名の入力(既存の`overlayNewCollection`フローを再利用)を行うオーバーレイを実装する
- [ ] 5.3 既存collectionを選んだ場合はそのcollectionの`.http`ファイルへ、新規collectionの場合は作成後のファイルへ、スクラッチリクエストを`###`ブロックとして追記する(`colStore.SaveRequests`等の既存APIを利用)
- [ ] 5.4 保存完了後に自動的に`Collections`モードへ切り替え、保存先collectionと保存したリクエストにフォーカス・選択状態を合わせる

## 6. ヘルプ・ドキュメント更新

- [ ] 6.1 `viewHelp()`・ステータスバーに`[`/`]`(モード切替)・`s`(Adhocからの保存)のキー説明を追加する
- [ ] 6.2 README.mdのキーバインド表・パネル説明にAdhocモードの導線を追記する

## 7. テスト

- [ ] 7.1 `internal/tui/shell/shell_test.go`にモード切り替え(`[`/`]`)のテストケースを追加する
- [ ] 7.2 collection未作成状態でのAdhocリクエスト編集・送信のテストケースを追加する
- [ ] 7.3 Adhocから既存collectionへの保存・新規collection作成による保存のテストケースを追加する
- [ ] 7.4 保存後に`Collections`モードへ自動切り替わり、対象が選択されることを検証するテストケースを追加する
- [ ] 7.5 `internal/tui/app_test.go`にAdhocの`OpenEditorMsg`(空collection名)処理のテストケースを追加する
