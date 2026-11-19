//go:build !windows

package main

import (
	"fmt"
	"os"
)

// ランタイムは Windows 専用です。Windows 以外では、EXE 内のゲームデータを確認する用途にだけ使えます。
//
//	go run . <ゲーム.exe>   … 埋め込まれたゲームの情報を表示
func main() {
	if len(os.Args) < 2 {
		fmt.Println("このプログラムは Windows 用です。build.sh で novel-player.exe をビルドしてください。")
		return
	}
	p, err := readPayloadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("title=%s id=%s size=%dx%d html=%d bytes\n", p.Title, p.ID, p.Width, p.Height, len(p.HTML))
}
