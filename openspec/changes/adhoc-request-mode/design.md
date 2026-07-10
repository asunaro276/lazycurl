## Context

現在のlazycurlは、起動すると常に`Collections`/`Requests`/`Response`/`History`の4パネルレイアウトが表示され、リクエストの新規作成は必ず既存のcollectionを前提とする(`internal/tui/shell/update.go`の`handleRequestsKey`は、collectionが選択されていない場合に新規リクエスト作成を拒否する)。

一方、想定される主な利用シーンは「まずcurlリクエストをその場で組み立てて送ってみたい」であり、保存(collection化)やenvironment変数の利用は、再利用したくなった時点で初めて必要になる二次的な機能である。この優先順位の違いを、起動直後の操作フローに反映する。

既存の`request-editor`(Method/URL/Params/Headers/Auth用フォーム + Bodyテキストエリア)は、すでに「1リクエストのエディタ」として`collection`から独立した部品になっている(`OpenEditorMsg`経由でApp側から呼ばれる)ため、これをAdhocモードでもそのまま再利用する。

## Goals / Non-Goals

**Goals:**
- collectionを作らずに、起動直後からMethod/URL/Headers/Bodyを組み立てて送信できる`Adhoc`モードを追加する
- `Adhoc`モードと既存の4パネルレイアウト(`Collections`モード)を、`[`/`]`キーでいつでも切り替えられるようにする
- `Adhoc`モードのリクエストは、ユーザーが明示的に保存するまでメモリ内にのみ存在させる
- 保存時は既存collectionへの追記、または新規collection作成のいずれかを選べるようにし、保存完了後は`Collections`モードへ自動的に切り替える
- 実行履歴(History)は両モードで共有する

**Non-Goals:**
- OpenAPIスキーマからのcollection一括生成(import機能)。これは別スレッドの将来課題であり、本changeのスコープ外
- Adhocリクエストのディスク永続化(セッションを跨いだ下書き保存)。保存しない限りメモリ内限定のままとする
- `Adhoc`モードでの`{{variable}}`展開。environment機能はcollectionに紐づく既存の設計のまま変更しない

## Decisions

### D1: モードはUIの表示状態として持ち、データモデルは変えない

`Adhoc`/`Collections`は`Shell`構造体が持つ表示モードのフラグ(例: `mode Mode`という2値のenum)として実装し、`collection.Store`や`.http`パーサ、`environment.Store`などの既存データ層には一切手を入れない。Adhocリクエストは`Shell`が保持する単一の`httpfile.Request`(未保存のスクラッチバッファ)として扱う。

代替案として検討したが不採用:
- Adhocリクエストを裏で隠しcollection(例: `__scratch__.http`)として自動保存する → 「保存するまでディスクに残らない」というユーザーの期待(実装済みの決定)に反し、ディスク上に見えないファイルが増えて混乱を招く

### D2: `Collections`モードの既存要件は変更せず、適用範囲だけ狭める

`tui-shell`の「パネルベースのレイアウト」要件(常時4パネル表示)は、`Collections`モードにいる間の要件として存置する。パネル自体の構成・キーバインド(`tab`/`hjkl`/`?`)は一切変更しない。変わるのは「そのレイアウトが常に画面に出ているか、`Adhoc`モードとの切り替え対象になるか」という点のみ。

### D3: 履歴は既存の`HistoryEntry`をそのまま流用する

`HistoryEntry`は既に`CollectionName string`フィールドを持つため、Adhocでの送信時はこれを空文字のまま記録すればよく、データ構造の変更は不要。`Collections`モードのHistoryパネルは全件(collection名が空のものも含む)をそのまま表示する。

### D4: 保存フローは既存の`overlayNewCollection`を再利用する

Adhocリクエストの保存(`s`キー)では、既存collectionの一覧から選ぶ画面と、`overlayNewCollection`(名前入力→`colStore.CreateCollection`)を組み合わせる。新規collectionの場合は作成直後にAdhocリクエストを`###`ブロックとして追記し、既存collectionの場合は選択したcollectionのリクエスト一覧に追記する。保存後は`focus`/`mode`を`Collections`・該当collection・該当リクエストへ更新する。

### D5: `[`/`]`はモード切り替え専用とし、`tab`/`shift+tab`とは独立させる

`tab`/`shift+tab`は各モード内のパネル間移動(既存のまま)。`[`/`]`はモードそのものの切り替えという、上位レイヤーの操作として新設する。現行コードに`[`/`]`の割り当ては存在しないため、キーバインドの衝突はない。

## Risks / Trade-offs

- [Risk] Adhocモードで保存せずに終了すると入力内容が消える → Mitigation: ステータスバー等で「未保存」であることを明示し、終了確認やヒント表示を検討する(詳細はtasksで具体化)
- [Risk] `Collections`モードの起動時レイアウト要件が「常時表示」から「モード切り替え時のみ表示」に変わることで、既存のtui-shell仕様を読むユーザー・実装者に誤解を与えうる → Mitigation: specの delta で明示的に scope を`Collections`モードに限定する文言に変更済み
- [Risk] 保存導線(`s`キー)が`Collections`モードの既存キー(`send`は`enter`、削除は`d/x`など)と将来的に衝突する可能性 → Mitigation: 現行キーマップに`s`は未使用であることを実装時に確認する

## Open Questions

- 保存時にcollectionを選ぶUIは、既存の`Collections`パネルのリスト選択を流用するか、専用のオーバーレイにするか
- Adhocモードで「未保存」であることをどう視覚的に示すか(ステータスバー文言、タブのマーカーなど)
