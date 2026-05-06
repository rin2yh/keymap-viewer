# プロジェクト規約

このリポジトリで Claude がコード・CI・PR を編集する際に守るルール。

## コード規約

- **setter / getter 実装禁止**
  `(*T).SetX(v)` や `(*T).X() V` のメソッドを書かない。依存はコンストラクタ引数で注入する。可視フィールドへの直接代入は許容するが、メソッドでラップしない。
- **build tag 禁止**
  `//go:build foo` を使ってテストや本番コードを切り替えない。テストの自動 skip は `testing.Short` や実行時条件 例 デバイスオープン失敗 で行う。

## CI / GitHub Actions

- **`latest` 禁止**
  `runs-on: macos-latest` のような移ろう label を使わない。`macos-15` のように具体的なバージョンを固定する。
- **action は SHA pin**
  `actions/checkout@v4` のようなタグ参照ではなく、`actions/checkout@<40-char-sha> # v4.3.0` の形式で commit SHA を固定する。バージョンタグは行末コメントで人間向けに残す。
- **テストサイズで job を分割**
  small / medium / large は別ジョブで実行する。small は `-short` を付けて実機・長時間テストを早期 skip させる。

## ドキュメント / PR / 表示テキスト

- **丸かっこ禁止**
  ジョブ名、PR の summary や test plan、コミットメッセージ、表示テキスト、コード内コメントなど、人間が目にする箇所では丸かっこを使わない。Go の関数呼び出しや bash の command substitution など構文要件のあるものは除く。
