## 1. 前提

- [ ] 1.1 別change `request-name-on-save` が実装・マージ済みであることを確認する(保存フローの単純化を前提に本changeを進めるため)

## 2. モデル層の整理(internal/tui/shell/model.go)

- [ ] 2.1 `Mode`型・`ModeAdhoc`/`ModeCollections`定数・`mode`フィールドを削除する
- [ ] 2.2 `Panel`の並びを`PanelRequest`(旧`PanelEditor`) / `PanelResponse` / `PanelCollections` / `PanelHistory`の4つ、番号`0`-`3`に固定する
- [ ] 2.3 `requestZone`(`zoneList`/`zoneForm`)を廃止し、`form.Editor`が持つnormal/insert状態に置き換える
- [ ] 2.4 `adhocRequest`関連のフィールド・保存関連フィールド(`saveIdx`/`savingAdhoc`)を、モードに依存しない「collectionに属さないRequest」の状態として整理する

## 3. form.Editorへのnormal/insert状態導入(internal/tui/form/editor.go)

- [ ] 3.1 `Editor`に`editing bool`相当の状態を追加し、`KVGrid`と同じ考え方(非editing時は移動、`enter`でinsert開始、`esc`でinsert終了)をName/Method/URL/タブ内容に適用する
- [ ] 3.2 `Editor`に、ツリー全体(Editor自身+内包する`KVGrid`)のどこかがinsert状態かどうかを返すアクセサ(例: `Editing() bool`)を追加する
- [ ] 3.3 Params/HeadersタブでKVGrid行への`enter`が、Editorのinsert→KVGridのediting、という2階層のネストとして正しく機能することを確認する
- [ ] 3.4 `esc`が直近1階層だけ戻す(KVGridのediting中ならKVGridのediting終了のみ、Editor全体のnormalには戻らない)ことを確認する

## 4. キーハンドリングの書き換え(internal/tui/shell/update.go)

- [ ] 4.1 `handleKey`から`inFormZone()`による全キー奪取を廃止し、`form.Editor.Editing()`がfalseの場合のみ`0`-`3`キーをパネル切替として処理する
- [ ] 4.2 `toggleMode`/`panelsForMode`を削除し、パネル集合を`[PanelRequest, PanelResponse, PanelCollections, PanelHistory]`固定にする
- [ ] 4.3 `[2] Collections`パネル: カーソル移動でコレクション行の展開/畳みを切り替え、展開中のリクエスト行で`enter`を押すと`[0] Request`へロードしフォーカスを移すロジックを実装する
- [ ] 4.4 `[3] History`パネル: カーソル移動でエントリの展開/畳みプレビューを切り替え、`enter`で`viewingIdx`を確定し`[1] Response`へ反映するロジックを実装する
- [ ] 4.5 保存フロー(`saveFormZone`相当)を、モードに依存しない「collectionに属しているか」の分岐に整理する

## 5. 描画の書き換え(internal/tui/shell/view.go)

- [ ] 5.1 `viewAdhocLayout`/`viewCollectionsLayout`/`viewModeTabs`を廃止し、常時2x2グリッド(`[0][1]`上段、`[2][3]`下段)を描画する単一のレイアウト関数に統合する
- [ ] 5.2 `[2] Collections`パネルのアコーディオン描画(選択中コレクションのみリクエスト一覧展開)を実装する。既存の`viewRequestsAccordion`をパターンとして参考にする
- [ ] 5.3 `[3] History`パネルのアコーディオン描画(選択中エントリのみ展開)を実装する
- [ ] 5.4 フッターのキーバインドヒントを、フォーカスパネル・`Editor`のnormal/insert状態・アクティブタブから動的に導出する関数へ置き換える

## 6. テスト

- [ ] 6.1 `internal/tui/form`: Editorのnormal/insert遷移(enter/escの往復、ネスト時のescの粒度)を検証するテストを追加する
- [ ] 6.2 `internal/tui/shell`: `0`-`3`キーがRequestパネルのnormal状態・insert状態それぞれでどう振る舞うかを検証するテストを追加する
- [ ] 6.3 `internal/tui/shell`: Collectionsパネルのドリルダウン(コレクション選択→リクエスト選択→`[0]`へロード)を検証するテストを追加する
- [ ] 6.4 `internal/tui/shell`: Historyパネルのプレビュー展開と`enter`確定によるResponse反映を検証するテストを追加する
- [ ] 6.5 モード関連の既存テスト(`toggleMode`、`panelsForMode`等)を新しいパネル構成に合わせて更新・削除する
- [ ] 6.6 `go build ./... && go test ./...`が通ることを確認する

## 7. openspec archive時の後処理

- [ ] 7.1 archive時に`openspec/specs/adhoc-mode/spec.md`が空になることを確認し、必要であれば`openspec/specs/adhoc-mode/`ディレクトリ自体の扱いをopenspec運用ルールに従って整理する
