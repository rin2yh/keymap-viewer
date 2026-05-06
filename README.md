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

## macOS の Input Monitoring 権限設定

macOS では HID デバイスへアクセスするアプリケーションごとに **Input Monitoring** 権限を付与する必要があります。これを設定しないと `--list-hid` でも何も表示されません。

1. **System Settings → Privacy & Security → Input Monitoring** を開く
2. 左下のロックを解除し、`+` ボタンから以下を追加する:
   - `Terminal.app` (CLI から `go run` / バイナリを実行する場合)
   - 使用している IDE (VS Code, GoLand, Cursor など)
   - ビルド済みの keymap-viewer バイナリ (Finder で `/Users/yuuki/workspace/keymap-viewer/keymap-viewer` を選択して追加)
3. 追加後、各アプリを **完全に終了** してから再起動する (Cmd+Q で終了。ウィンドウを閉じるだけでは反映されない)
4. それでも `--list-hid` で何も表示されない場合は、USB ケーブルを差し直す。可能なら別のポートやハブを介さず直結で試す

## トラブルシューティング

- **`no VIA Raw HID interface found` が出る**
  - Input Monitoring 権限が当該プロセスに付与されているか確認する
  - USB ケーブル・ポートを変えて再接続する
  - `--list-hid` の出力に `usage_page=0xFF60` を持つエントリがあるか確認する。無い場合は VIA 対応ファームウェアが書き込まれていない可能性が高い
- **読み取った数値が VIA Web 版 ([usevia.app](https://usevia.app)) と食い違う**
  - ファームウェア側の VIA 互換性を確認する。古い QMK ビルドでは keycode マッピングがズレていることがある
  - VIA JSON 定義ファイルのバージョンも合わせて確認する

## テスト・lint

サンドボックス環境では `GOCACHE` の書き込み先を明示する必要があります。

```sh
# テスト (race 検出付き)
GOCACHE=$TMPDIR/go-cache go test ./... -race

# フォーマット差分の確認
gofmt -l ./...

# 静的解析
go vet ./...
```

## 読み取り専用保証

`internal/via/command.go` には読み取り系の 4 個の `CommandID` 定数しか宣言されていません。書き込み系 (`0x05`, `0x13`, `0x06`, `0x0A`, `0x0B` など) は定数すら存在せず、低レベルの `writeReport` には許可済み CommandID のホワイトリストガードがあり、未知の値は panic で落ちます。公開 API である `*ReadOnlyClient` のメソッドは `ProtocolVersion` / `LayerCount` / `Keycode` / `KeymapBuffer` / `DeviceInfo` / `Close` のみで、setter は一切ありません。これらの不変条件は `internal/via/client_test.go` でテストされています。

## ライセンス

Apache-2.0 (予定 / 未定)

## 参考リンク

- VIA 仕様: <https://www.caniusevia.com/docs/specification>
- Corne 用 VIA 定義: <https://github.com/the-via/keyboards/blob/master/v3/crkbd/crkbd.json>
- guigui (GUI フレームワーク): <https://github.com/guigui-gui/guigui>
- go-hid (HID バインディング): <https://github.com/sstallion/go-hid>
