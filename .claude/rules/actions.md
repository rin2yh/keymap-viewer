---
paths:
  - ".github/workflows/**"
---

# GitHub Actions / CI 規約

- **`latest` 禁止**
  `runs-on: macos-latest` のような移ろう label を使わない。`macos-15` のように具体的なバージョンを固定する。
- **action は SHA pin**
  `actions/checkout@v4` のようなタグ参照ではなく、`actions/checkout@<40-char-sha> # v4.3.0` の形式で commit SHA を固定する。バージョンタグは行末コメントで人間向けに残す。
- **テストサイズで job を分割**
  small / medium / large は別ジョブで実行する。small ジョブは `-short` を付けて、実機テストや長時間テストを早期 skip させる。
- **ジョブ名・ステップ名で丸かっこ禁止**
  `test (small)` ではなく `test small` と書く。詳細は `text.md` を参照。
