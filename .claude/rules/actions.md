---
paths:
  - ".github/workflows/**"
---

# GitHub Actions / CI 規約

- **`latest` 禁止**
  `runs-on: macos-latest` のような移ろう label を使わない。`macos-15` のように具体的なバージョンを固定する。
- **action は SHA pin**
  `actions/checkout@v4` のようなタグ参照ではなく、`actions/checkout@<40-char-sha> # v4.3.0` の形式で commit SHA を固定する。バージョンタグは行末コメントで人間向けに残す。
