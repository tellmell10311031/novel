package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
)

// エディタが EXE の末尾に付け足すデータの形式:
//
//	[ペイロード(JSON, UTF-8)][ペイロード長: uint64 リトルエンディアン][マジック "NVLPKG01"]
const trailerMagic = "NVLPKG01"
const trailerSize = 8 + len(trailerMagic)

// Payload はエディタから渡されるゲーム情報です。
type Payload struct {
	Title  string `json:"title"`
	ID     string `json:"id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	HTML   string `json:"html"`
}

var errNoPayload = errors.New("ゲームデータが見つかりません")

func readPayloadFrom(r io.ReaderAt, size int64) (*Payload, error) {
	if size < int64(trailerSize) {
		return nil, errNoPayload
	}
	tail := make([]byte, trailerSize)
	if _, err := r.ReadAt(tail, size-int64(trailerSize)); err != nil {
		return nil, err
	}
	if string(tail[8:]) != trailerMagic {
		return nil, errNoPayload
	}
	n := binary.LittleEndian.Uint64(tail[:8])
	if n == 0 || n > uint64(size-int64(trailerSize)) {
		return nil, errNoPayload
	}
	buf := make([]byte, n)
	if _, err := r.ReadAt(buf, size-int64(trailerSize)-int64(n)); err != nil {
		return nil, err
	}
	p := &Payload{}
	if err := json.Unmarshal(buf, p); err != nil {
		return nil, err
	}
	if p.HTML == "" {
		return nil, errNoPayload
	}
	if p.Width <= 0 {
		p.Width = 1280
	}
	if p.Height <= 0 {
		p.Height = 720
	}
	if p.Title == "" {
		p.Title = "Novel Game"
	}
	return p, nil
}

func readPayloadFile(path string) (*Payload, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return readPayloadFrom(f, st.Size())
}
