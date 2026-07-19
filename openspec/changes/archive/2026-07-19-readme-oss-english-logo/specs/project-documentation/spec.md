## ADDED Requirements

### Requirement: README is written in English with an ASCII wordmark
リポジトリの `README.md` はSHALL全文英語で記述され、日本語版(`README.ja.md`等)は提供しない。`README.md` の冒頭にはSHALLプロジェクト名(`lazycurl`)の`figlet`(`standard`フォント)によるASCIIワードマークをコードブロックとして含める。

#### Scenario: A first-time visitor reads the README
- **WHEN** GitHub上または`cat README.md`で `README.md` を開く
- **THEN** 全セクション(インストール、使い方、キーバインド、コレクションの保存形式、pragma一覧、environmentと変数展開)が英語で表示され、冒頭に`lazycurl`のASCIIワードマークが表示される

### Requirement: Supplementary documentation is written in English
`docs/claude-code-remote-setup.md` はSHALL全文英語で記述され、日本語版は提供しない。

#### Scenario: A contributor reads the remote setup doc
- **WHEN** `docs/claude-code-remote-setup.md` を開く
- **THEN** 内容がすべて英語で表示される

### Requirement: Repository includes an OSS license
リポジトリルートにSHALL `LICENSE` ファイル(MIT License、著作権表記付き)が存在し、`README.md` からSHALLライセンスへの言及(セクションまたはバッジ)がある。

#### Scenario: A visitor checks the repository's license
- **WHEN** GitHub上でリポジトリのライセンス表示、または`LICENSE`ファイルを直接確認する
- **THEN** MIT Licenseの全文と著作権表記が確認でき、`README.md`にもライセンスへの言及がある
