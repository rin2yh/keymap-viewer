# keymap-viewer

## 概要

keymap-viewer は QMK/VIA 対応キーボードのキーマップを VIA プロトコル経由で読み取り、[Remap](https://remap-keys.app/) ライクな UI で表示する読み取り専用デスクトップビューアです。書き込み系コマンドは型レベルで禁止されており、現時点では Corne v4 Chocolate (VID `0x4653`, PID `0x0001`) のみを対象としています。

## 必要環境

- Go 1.26
- macOS (動作確認済み)
- libhidapi (`go-hid` に同梱の C ソースを Cgo でビルドするため、追加インストールは不要だが Xcode Command Line Tools が必要)
- Corne v4 Chocolate ファームウェア (VIA 対応ビルドであること)

## ビルド・実行

依存関係は `go mod` が解決します。Cgo を有効にした状態でビルドしてください。

```sh
# ビルド
go build .

# 通常起動 (GUI ビューア)
go run .

# VIA プロトコルバージョンの確認
go run . --probe

# 接続中の HID デバイスを列挙 (デバッグ用途)
go run . --list-hid

# 全レイヤーのキーコードを stdout にダンプ
go run . --dump
```

## ドキュメント

- macOS セットアップ手順とトラブルシューティング: [docs/setup-macos.md](docs/setup-macos.md)
- 開発者向け情報 (テスト・lint・読み取り専用保証): [docs/development.md](docs/development.md)

## ライセンス

Apache-2.0 (予定 / 未定)

## 参考リンク

- VIA 仕様: <https://www.caniusevia.com/docs/specification>
- Corne 用 VIA 定義: <https://github.com/the-via/keyboards/blob/master/v3/crkbd/crkbd.json>
- guigui (GUI フレームワーク): <https://github.com/guigui-gui/guigui>
- go-hid (HID バインディング): <https://github.com/sstallion/go-hid>
