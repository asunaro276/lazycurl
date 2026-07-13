## Context

現状の`internal/tui/shell`は`Mode`(`ModeAdhoc`/`ModeCollections`)を持ち、`[`/`]`で切り替える。各モードは`panelsForMode()`が返す異なるパネル集合を持ち、`View()`は`viewAdhocLayout()`/`viewCollectionsLayout()`のどちらかを描画する。Collectionsモードでは`Requests`パネルがコレクション内リクエスト一覧と埋め込みフォーム(`zoneList`/`zoneForm`)を兼ねる。Adhocモードでは`Editor`パネルが常に`zoneForm`固定で存在する。

`handleKey()`は`inFormZone()`が真の間、`ctrl+c`以外の全キーを`handleFormZoneKey()`に委譲する。この結果、Editorパネルにフォーカスしている間はパネル番号キー(`1`-`4`)が機能しない。これがユーザーの主要な不満点であり、本changeの出発点である。

一方`internal/tui/form/kvgrid.go`の`KVGrid`は、非editing時は`j/k/h/l`でセル移動、`enter`で対象セルのediting開始、`esc`でediting終了、という2段階のフォーカス状態を既に実装している。本designはこのパターンをフォーム全体(Name/Method/URL/タブ切替)に一段引き上げることで、Requestパネル全体を同じ考え方で統一する。

## Goals / Non-Goals

**Goals:**
- モード切り替えを廃止し、4パネル(Request/Response/Collections/History)を常時同時表示する単一画面にする
- パネル番号キー(`0`-`3`)が、Requestパネルにフォーカスしている場合を含め常に機能するようにする(文字入力を奪っている間を除く)
- Collections/Historyパネルは、選択中の1項目だけをアコーディオン展開し、他は1行summaryに畳んで画面を圧迫しない
- フッターのキーバインドヒントを、フォーカス状態から動的に導出する

**Non-Goals:**
- Bodyタブの`$EDITOR`外部エディタ連携(`ctrl-e`)自体の変更。外部エディタ起動中の扱いは既存のまま(プロセス終了までブロッキング)とし、本changeでは「Bodyのテキストエリアにinsertで入る」までを扱う
- Authタブのラジオ選択(Bearer/Basic等)のUI自体の変更。normal/insert状態機械への組み込み方は`tasks.md`実装時に既存のAuthタブ構造を踏襲する
- リクエスト名の保存時プロンプト化(別change `request-name-on-save` で扱う。本changeはそれが完了済みである前提で設計する)

## Decisions

### 1. パネル構成は2x2固定グリッド、モードなし
`[0] Request` `[1] Response`を上段、`[2] Collections` `[3] History`を下段に固定配置し、常時4パネルとも表示する。lazygitのような「フォーカスしたパネルが拡大する」方式や、フォーカスしていないパネルを完全に隠す方式も検討したが、いずれもレイアウトが頻繁に大きく変わり、送信結果を見ながらリクエストを編集するという主動線の邪魔になるため採用しない。

### 2. Collections/Historyの「畳み」はレイアウト単位ではなく行単位のアコーディオン
既存の`viewRequestsAccordion()`(選択行のみ埋め込みフォームへ展開、他は1行summary)と同じパターンを、Collectionsパネル(コレクション行→選択中のみリクエスト一覧を展開)とHistoryパネル(選択中のエントリのみ詳細展開)に適用する。パネル自体の表示/非表示ではなく、パネル内部の行の展開/畳みで「情報量を絞る」という要件を満たす。

### 3. Collectionsは2段ドリルダウン
コレクション行にカーソルがあり`enter`を押すとそのコレクションのリクエスト一覧が展開される(パネルは`[2]`のまま)。展開されたリクエスト行でさらに`enter`を押すと、そのリクエストが`[0] Request`パネルへロードされ、フォーカスも`[0]`へ自動的に移る。

### 4. Historyは「カーソル移動=展開表示」「enter=Responseへの反映確定」の2段階
既存の`viewingIdx`(-1=ライブレスポンス、>=0=履歴参照)をそのまま踏襲する。Historyパネル内で`j/k`によりカーソルを移動すると、その行がパネル内で展開表示される(プレビュー)。`enter`を押した時点で`viewingIdx`が更新され、`[1] Response`パネルの表示にも反映される。

### 5. Requestパネルのnormal/insert状態は`KVGrid.editing`パターンの昇格
`form.Editor`に`editing bool`相当の状態を追加する:
- normal: `j/k`(または`tab`/`shift+tab`)でName/Method/URL/アクティブタブの内容間を移動。`enter`でフォーカス中フィールドのinsertへ。`[`/`]`でタブ切替
- insert: 対象フィールド(textinput/textarea/KVGridセル)が文字入力を占有。`esc`でnormalへ戻る
- Params/Headersタブでは、normalからKVGrid行へ`enter`すると、KVGrid自体の`editing`状態(セル編集)にネストする。この場合も「ツリーのどこかがinsert/editing」という単一の判定に集約される

Shellは`Panel`のフォーカスに関わらず「現在insert中かどうか」を`Editor`から取得できるアクセサ(例: `Editing() bool`)を通じて把握し、insert中でなければ`0`-`3`キーを常にパネル切替として処理する。この判定を`inFormZone()`ベースの全キー奪取から、「insert中のみ最小限のキーを奪う」方式に置き換えるのが本changeの核心的な変更である。

### 6. フッターヒントは状態から導出する関数として実装する
`Shell`のフォーカスパネル・Requestパネルのnormal/insert・アクティブタブから、表示すべきヒント文字列を決定する関数を用意する(静的な3パターンのハードコードをやめる)。

### 7. `adhoc-mode`capabilityの要求は`tui-shell`へ移管する
「モード切替」「Adhocモードのレイアウト」要求はモード概念ごと不要になるため削除する。「Adhocリクエストの一時性」「collectionへの保存」は、「Requestパネルがcollectionに属していない状態」の性質として`tui-shell`へ移す。「履歴の共有」要求は、Historyパネルがそもそも1つしか存在しなくなるため独立した要求としては不要になる(自明に満たされる)。

## Risks / Trade-offs

- [Risk] normal/insertの状態がRequestパネル内でネストする(Editorレベル→KVGridレベル)ため、`esc`を1回押した時にどの階層まで戻るべきかが曖昧になりうる → Mitigation: 「1回の`esc`は直近1階層だけ戻す」という一貫ルールを設け、`tasks.md`実装時にKVGridの`cancelEdit`と同じ粒度で統一する
- [Risk] Collections/Historyのアコーディオン展開ロジックが、既存の`viewRequestsAccordion`と類似コードの重複を生む可能性 → Mitigation: 実装時に共通のアコーディオン描画ヘルパーへ切り出すことを検討する(本designでは必須としない)
- [Risk] `request-name-on-save`と保存フロー(`saveFormZone`/`overlaySaveAdhoc`)のコードが重なるため、先に着手した側の変更が他方のブランチと衝突する → Mitigation: `request-name-on-save`を先にマージしてから本changeに着手する(`proposal.md`のImpactに明記済み)
- [Trade-off] 4パネル常時表示は、狭い端末幅では各パネルの表示領域が小さくなる → 許容する(既存のCollectionsモードも同様に4分割していたため、既存の制約を超えるものではない)

## Migration Plan

1. `request-name-on-save`を先に実装・マージする
2. `form.Editor`にnormal/insert状態を追加し、既存の`tab`ベースのフォーカス巡回をnormal状態のナビゲーションとして再利用する
3. `Shell`から`Mode`/`toggleMode`/`panelsForMode`のモード分岐を削除し、常時4パネルの`View()`に統合する
4. Collections/Historyパネルにアコーディオン描画とドリルダウン/確定操作を実装する
5. フッターヒントを状態導出関数に置き換える
6. `openspec/specs/adhoc-mode/`を空にする変更をarchive時に反映する

## Open Questions

- Bodyタブのテキストエリアがinsert状態にある間、`ctrl-e`(外部エディタ)はinsert状態を維持したまま実行するか、一旦normalに戻すか(実装時に既存の`ctrl-e`ハンドリングを見て決める)
- History/Collectionsパネルの表示行数が画面高さを超える場合のスクロール方式(既存のリスト表示にスクロールが実装されているか要確認)
