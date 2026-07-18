## Context

`internal/curlexec`の既存テスト(`executor_test.go`)は全て`Runner`インターフェースを`fakeRunner`に差し替えて検証しており、`buildArgs`(`internal/curlexec/argv.go`)が生成するargv(`-X`/`-H`/`-k`/`--max-time`/`-L`/`--data-binary`/`-D`/`-o`/`-w '%{json}'`)が実際のcurlバイナリ・実際のHTTPサーバーに対して正しく振る舞うかは一切検証されていない。

加えて、進行中の`stream-response-body`変更(`openspec/changes/stream-response-body/`, 未着手)は`@stream`プラグマによる逐次配信(`-N`、stdoutパイプ経由読み取り)を実装する予定だが、そのtask 6.1は「SSEを返す簡易テストサーバーを用いた手動確認」を前提にしている。ファイルベースの`fakeRunner`ではプロセス起動・接続維持・チャンク到着タイミング・ctrl-c打ち切り・自然終了のいずれも再現できないため、この検証には本物のHTTPサーバーと本物のcurlプロセスが要る。

このリポジトリには現状CIが無く(CLAUDE.md参照)、Docker前提のテスト層を導入しても既存の`go test ./...`のスピード・安定性を損なわない設計にする必要がある(CI整備は別途後続で行う想定)。

## Goals / Non-Goals

**Goals:**
- curl argvの各フラグ(認証ヘッダー、redirect追従、timeout、insecureは対象外〔Non-Goals参照〕)を、実curlサブプロセス経由で実地検証できるE2Eテスト層を`internal/curlexec`に追加する
- キー操作からResponseパネルの描画までを、実curl・実TUIレンダリングを通して検証するTUIレベルのE2Eテスト層を`internal/tui`に追加する
- 開発者が手元でモックサーバーを常駐起動し、TUIから実際にリクエストを送って(特に`@stream`実装時の)挙動を目視確認できるようにする
- モックサーバーの実体を自動テスト・手動確認の両方で完全に同一のものにする(Dockerイメージを共有の単位とする)
- モックサーバーの実装をlazycurl本体の依存グラフから完全に独立させる

**Non-Goals:**
- `@insecure`(TLS自己署名証明書)の実地検証は今回のスコープに含めない。素のHTTPで検証可能なフラグ群(`-X`/`-H`/`-L`/`--max-time`/`--data-binary`/`-D`/`-o`/`-w`)を優先し、TLS対応は後続の変更で扱う
- E2EテストのCI組み込み・ビルドタグによる隔離は行わない。CI整備自体が別途後続の作業
- SSEの`event:`/`data:`フレームパースなど、`stream-response-body`本体の実装は本変更のスコープ外。本変更はあくまで「その検証に使えるモックサーバーとE2Eテスト基盤」を提供する
- pty(擬似端末)経由でコンパイル済みバイナリそのものを操作する完全なブラックボックスE2E(`creack/pty`や`vhs`的なアプローチ)は採用しない。詳細は「Decisions」参照

## Decisions

### モックサーバーは`testing/mockserver/`配下に独自`go.mod`を持つ完全に独立したモジュールとして実装する

自動テスト(testcontainers-go)も手動確認(docker compose)もモックサーバーとはHTTP経由でしか通信しないため、Goコードレベルでlazycurl本体と共有する必要が無い。独自`go.mod`にすることで:
- HTTPフレームワーク(gin/echo/素の`net/http`等)を自由に選定でき、選定してもroot moduleの`go.mod`/`go.sum`を一切汚さない
- Dockerのbuild contextを`testing/mockserver/`だけに閉じられる
- root moduleの`go build ./...`の対象に入らない

代替案として「共有ハンドラpackageをhttptest.Serverとcmd/バイナリの両方から import する」構成も検討したが、コンテナ経由に倒した時点でGoレベルの共有は不要になるため不採用。

SSE的な逐次配信は`net/http.ResponseWriter`が`http.Flusher`を実装してさえいれば実現できる(Ginなら`c.Writer.Flush()`、Echoなら`c.Response().Flush()`)。フレームワーク選定に技術的制約は無いため、実装担当が使い慣れたもの(素の`net/http`を含む)を選べばよい。

### 自動テストは`testcontainers-go`でDockerfileから都度ビルドし、コンテナはテストバイナリ内で1つだけ起動して使い回す

`testcontainers-go`の`FromDockerfile`で`testing/mockserver/Dockerfile`からイメージをビルド・起動する。イメージレジストリへのpushはCI整備後の話であり、今は不要。

モックサーバーはステートレス(リクエストごとの独立した応答のみで、テスト間の状態を持たない)なため、`TestMain(m *testing.M)`でコンテナを1回だけ起動し、マップされたベースURLをパッケージ変数として全テストで共有する(singleton containerパターン)。これによりテストごとのコンテナ起動コスト(数百ms〜数秒)を回避する。テスト終了後に`TestMain`内でコンテナをterminateする。

代替案として「テストケースごとにコンテナを起動」も検討したが、モックサーバーに共有状態が無いため隔離の必要性が無く、起動コストだけが積み上がるため不採用。

`testcontainers-go`は root moduleの依存になる(`go.mod`/`go.sum`にDockerクライアント関連の依存が追加される)。最終的な`lazycurl`バイナリ(`cmd/lazycurl`)からは参照されないため実行時の影響は無いが、`go.sum`は肥大化する。これは許容する(E2Eテストを別モジュールに切り出す案も検討したが、root moduleのテストコードとして自然に書ける方を優先した)。

ビルドタグ(`//go:build e2e`等)による実行対象の分離は今回は行わない。`go test ./...`にそのまま含める。Dockerが無い環境での実行方法(スキップ等)は今回のスコープでは扱わず、CI整備時に合わせて検討する。

### 手動確認は同一Dockerfileを参照する`docker-compose.yml`で固定ポート公開する

`docker compose up`で起動し、固定のホストポートでモックサーバーを公開する。開発者がlazycurlのTUIから毎回同じURLの`.http`リクエストを叩けるようにするため、動的ポート割り当てではなく固定ポートとする。

### TUIのE2Eは`teatest`を使い、pty経由の完全E2Eは採用しない

`internal/tui/shell/shell_test.go`の既存テストは全て`s.Update(msg)`を直接呼ぶモデル単体テストであり、`tea.Program`の起動・`View()`の実レンダリング・非同期`tea.Cmd`(送信結果の`sendResultMsg`等)の実行タイミングは一切検証されていない。

TUIレベルのE2Eには以下の3段階を検討した:

1. 現状の`Update()`直叩き(最も軽いが、実`tea.Program`のライフサイクルを通らない)
2. `teatest`(`github.com/charmbracelet/x/exp/teatest`, Bubble Tea公式のテストヘルパー): `tea.Program`を`io.Pipe`ベースでin-process起動し、`tm.Send(tea.KeyMsg{...})`でキー入力を送り込み、`teatest.WaitFor`で特定の出力(ANSI込みの生端末出力)が現れるまで待つ、あるいは`tm.FinalOutput()`をgolden fileと突き合わせる
3. pty(擬似端末)経由でコンパイル済みバイナリを実際の端末として操作する完全なブラックボックスE2E(`creack/pty`等)

2の`teatest`を採用する。理由は、pty無しでも本物の`tea.Program`・`Update`/`View`ループ・非同期`tea.Cmd`実行を通しで検証でき、`internal/curlexec`のE2Eテストと同様に`tui.New(...)`へ`curlexec.NewExecutor()`(実curl)を渡すことで、モックサーバーコンテナに対する実際のリクエスト送信からResponseパネルの描画までを一気通貫で検証できるため。3(pty経由)は最も"本物"に近いが、実端末のリサイズ・色・タイミング依存でフレーキーになりやすく、このプロジェクトの規模に対しては過剰と判断し不採用とした。

`internal/tui`のE2Eテストも`internal/curlexec`と同様に、`TestMain`で`testing/mockserver`コンテナをパッケージ内で1回だけ起動して共有する(Goのテストバイナリはパッケージ単位でビルドされるため、`internal/curlexec`と`internal/tui`はそれぞれ別々にコンテナを1つ起動することになるが、いずれも起動は1回のみで、テストケースごとの再起動は発生しない)。

### モックサーバーが提供するエンドポイント

`buildArgs`が生成するcurlフラグの実地検証、および`stream-response-body`のtask 6.1(逐次表示・ctrl-c打ち切り・自然終了の確認)をカバーする最小セットとする:

| エンドポイント | 検証対象 |
|---|---|
| `/echo`(メソッド・ヘッダー・bodyをJSONで返す) | `-X`, `-H`, `--data-binary` |
| `/status/{code}` | `-w '%{json}'`によるstatus code取得、`-D`ヘッダーファイル |
| `/redirect/{n}` | `-L`(follow)、`@no-redirect`時の非follow |
| `/delay/{sec}` | `--max-time`(`@timeout`)、ctrl-c打ち切り |
| `/stream` | `@stream`時の`-N`・stdoutパイプ経由の逐次配信(chunkを時間差で送出) |
| `/auth/basic`, `/auth/bearer` | `Authorization`ヘッダーの導出(Basic/Bearer) |

## Risks / Trade-offs

- [Docker Desktop/daemonが無い開発環境ではE2Eテストが失敗する] → CI未整備の現時点では許容。README/CLAUDE.mdに前提として明記する
- [`testing/mockserver/`が独自`go.mod`を持つことで、依存更新(`go mod tidy`)を2箇所で個別に行う必要がある] → モックサーバーは依存の少ない単純なHTTPサーバーに留める想定であり、更新頻度は低いと見込む
- [ビルドタグ隔離をしないため、Docker環境が無いと`go test ./...`全体が失敗する] → 早期に問題として顕在化させる狙いで許容する。開発体験上の問題が大きいと判明した場合は後続でタグ隔離を検討する
- [固定ポートのdocker composeは、同一ホストで他のサービスとポート競合する可能性がある] → 競合時は`docker-compose.yml`のポートマッピングを手動で変更してもらう運用とする
- [`teatest`ベースのTUI E2Eは、実curl実行+コンテナ通信+非同期`tea.Cmd`のタイミングが絡むため、`Update()`直叩きの既存テストより実行時間が長く、フレーキーになりうる] → `teatest.WaitFor`のタイムアウトを十分に長く取る。恒常的に不安定な場合はテストケース数を絞り、代表的なgolden pathのみをTUI E2Eで担保し、詳細な分岐は既存の`Update()`直叩きテストに任せる

## Open Questions

- `@insecure`(TLS)の実地検証を将来的に追加する場合、`httptest.NewTLSServer`相当の自己署名証明書対応をモックサーバー自体に持たせるか、別エンドポイントとして分離するかは未決
- モックサーバーのエンドポイント一覧は実装時に過不足が見つかる可能性があり、tasks側で調整の余地を残す
