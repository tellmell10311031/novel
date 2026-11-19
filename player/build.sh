#!/usr/bin/env bash
# novel-player.exe（Windows 用ゲームランタイム）をビルドし、
# エディタが読み込む ../runtime/player-stub.js を生成します。
set -euo pipefail
cd "$(dirname "$0")"

go mod tidy
go test ./...
go run ./tools/mkres -o rsrc_windows_amd64.syso
mkdir -p build
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H windowsgui" -o build/novel-player.exe .

# 同梱するライセンス表記
lic=build/THIRD_PARTY_LICENSES.txt
{
  echo "このソフトウェアは以下のオープンソースソフトウェアを利用しています。"
  echo "This software uses the following open source software."
  echo
  print_license() {
    echo "================================================================"
    echo "$1"
    echo "================================================================"
    cat "$2"
    echo
  }
  print_license "The Go Programming Language" "$(go env GOROOT)/LICENSE"
  for m in github.com/jchv/go-webview2 github.com/jchv/go-winloader golang.org/x/sys; do
    d=$(go list -m -f '{{.Dir}}' "$m")
    print_license "$m" "$(ls "$d"/LICENSE* | head -1)"
  done
} > "$lic"

mkdir -p ../runtime
node -e '
const fs = require("fs");
const exe = fs.readFileSync("build/novel-player.exe").toString("base64");
const lic = fs.readFileSync(process.argv[1], "utf8");
fs.writeFileSync("../runtime/player-stub.js",
  "// 自動生成ファイル（player/build.sh）。編集しないでください。\n" +
  "window.NOVEL_PLAYER = {\n" +
  "  iconSlotSize: 300000,\n" +
  "  iconSlotMarker: \"NOVELMAKER-ICON-SLOT-V1\",\n" +
  "  licenses: " + JSON.stringify(lic) + ",\n" +
  "  exe: \"" + exe + "\"\n};\n");
' "$lic"
ls -l build/novel-player.exe ../runtime/player-stub.js
