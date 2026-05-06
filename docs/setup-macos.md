# macOS セットアップ

## Input Monitoring 権限設定

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
