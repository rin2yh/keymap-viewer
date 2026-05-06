# 開発者ガイド

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
