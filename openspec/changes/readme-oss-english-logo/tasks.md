## 1. ライセンス追加

- [ ] 1.1 リポジトリルートに `LICENSE` ファイルを追加する(MIT License、`Copyright (c) 2026 asunaro276`)

## 2. ロゴ生成

- [ ] 2.1 `figlet -f standard lazycurl` でASCIIワードマークを生成する

## 3. README英訳・置き換え

- [ ] 3.1 `README.md` 冒頭に2.1で生成したASCIIワードマークをコードブロックとして追加する
- [ ] 3.2 `README.md` の全セクション(概要、インストール、`curl`依存、使い方、Adhoc/Collectionsモード、キーバインド、コレクションの保存形式、pragma一覧、`@stream`の説明、environmentと変数展開)を英訳し、日本語原文を置き換える
- [ ] 3.3 `README.md` にライセンスへの言及(MITライセンスセクションまたはバッジ)を追加する

## 4. docs英訳・置き換え

- [ ] 4.1 `docs/claude-code-remote-setup.md` を全文英訳し、日本語原文を置き換える

## 5. 確認

- [ ] 5.1 `go build ./...` と `go test ./...` が既存どおり通ることを確認する(ドキュメントのみの変更でありコードへの影響がないことの確認)
- [ ] 5.2 README・docsに日本語が残っていないことを目視で確認する
