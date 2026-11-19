// mkres は novel-player.exe に埋め込む Windows リソース（アイコン・マニフェスト）を生成します。
//
// アイコンは 256x256 の PNG を「固定サイズの枠（スロット）」に入れて埋め込みます。
// エディタは EXE 出力時にこの枠の中身を利用者の画像に置き換えるため、
// EXE のヘッダーやリソース表を書き換える必要がありません。
//
//	go run ./tools/mkres -o rsrc_windows_amd64.syso
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/tc-hib/winres"
)

// エディタ（editor.html）と同じ値にすること
const (
	IconSlotSize   = 300000
	IconSlotMarker = "NOVELMAKER-ICON-SLOT-V1"
)

func main() {
	out := flag.String("o", "rsrc_windows_amd64.syso", "output .syso file")
	flag.Parse()

	slot := padPNG(encode(defaultIcon()), IconSlotSize)

	rs := winres.ResourceSet{}
	// RT_ICON #1 = 256x256 PNG（スロット）
	if err := rs.Set(winres.RT_ICON, winres.ID(1), winres.LCIDNeutral, slot); err != nil {
		log.Fatal(err)
	}
	// RT_GROUP_ICON #1
	grp := new(bytes.Buffer)
	binary.Write(grp, binary.LittleEndian, []uint16{0, 1, 1}) // reserved, type=icon, count
	grp.Write([]byte{0, 0, 0, 0})                            // width=256, height=256, colors, reserved
	binary.Write(grp, binary.LittleEndian, []uint16{1, 32})  // planes, bitcount
	binary.Write(grp, binary.LittleEndian, uint32(len(slot)))
	binary.Write(grp, binary.LittleEndian, uint16(1)) // RT_ICON id
	if err := rs.Set(winres.RT_GROUP_ICON, winres.ID(1), winres.LCIDNeutral, grp.Bytes()); err != nil {
		log.Fatal(err)
	}
	rs.SetManifest(winres.AppManifest{
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
	})

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := rs.WriteObject(f, winres.ArchAMD64); err != nil {
		log.Fatal(err)
	}
}

func encode(img image.Image) []byte {
	b := new(bytes.Buffer)
	if err := png.Encode(b, img); err != nil {
		log.Fatal(err)
	}
	return b.Bytes()
}

// padPNG は PNG の IEND の直前に私用チャンク "npAd"（目印＋ゼロ埋め）を入れ、全体を size バイトにします。
func padPNG(p []byte, size int) []byte {
	body := p[:len(p)-12] // IEND を除く
	padLen := size - len(body) - 12 - 12
	if padLen < len(IconSlotMarker) {
		log.Fatalf("icon too large: %d", len(p))
	}
	data := make([]byte, padLen)
	copy(data, IconSlotMarker)
	out := append([]byte{}, body...)
	out = appendChunk(out, "npAd", data)
	out = appendChunk(out, "IEND", nil)
	if len(out) != size {
		log.Fatalf("slot size mismatch %d", len(out))
	}
	return out
}

func appendChunk(b []byte, typ string, data []byte) []byte {
	n := make([]byte, 4)
	binary.BigEndian.PutUint32(n, uint32(len(data)))
	b = append(b, n...)
	td := append([]byte(typ), data...)
	b = append(b, td...)
	binary.BigEndian.PutUint32(n, crc32.ChecksumIEEE(td))
	return append(b, n...)
}

// 標準アイコン：紫のグラデーションに白い本
func defaultIcon() image.Image {
	const S = 256
	img := image.NewNRGBA(image.Rect(0, 0, S, S))
	inRounded := func(x, y, x0, y0, x1, y1, r int) bool {
		if x < x0 || x >= x1 || y < y0 || y >= y1 {
			return false
		}
		cx, cy := x, y
		if x < x0+r {
			cx = x0 + r
		} else if x >= x1-r {
			cx = x1 - r - 1
		}
		if y < y0+r {
			cy = y0 + r
		} else if y >= y1-r {
			cy = y1 - r - 1
		}
		dx, dy := x-cx, y-cy
		return dx*dx+dy*dy <= r*r
	}
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			if !inRounded(x, y, 8, 8, S-8, S-8, 44) {
				continue
			}
			t := float64(x+y) / float64(2*S)
			c := color.NRGBA{uint8(79 + t*60), uint8(107 - t*15), uint8(237 + t*3), 255}
			// 本（左右のページ）
			left := inRounded(x, y, 52, 70, 124, 190, 10)
			right := inRounded(x, y, 132, 70, 204, 190, 10)
			if left || right {
				c = color.NRGBA{255, 255, 255, 255}
				// 行の線
				for _, ly := range []int{100, 122, 144, 166} {
					if y >= ly && y < ly+5 && ((x > 64 && x < 112) || (x > 144 && x < 192)) {
						c = color.NRGBA{190, 196, 240, 255}
					}
				}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}
