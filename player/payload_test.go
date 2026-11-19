package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func pack(stub []byte, p Payload) []byte {
	data, _ := json.Marshal(p)
	out := append([]byte{}, stub...)
	out = append(out, data...)
	n := make([]byte, 8)
	binary.LittleEndian.PutUint64(n, uint64(len(data)))
	out = append(out, n...)
	return append(out, trailerMagic...)
}

func TestReadPayload(t *testing.T) {
	b := pack([]byte("MZ fake exe body"), Payload{Title: "放課後の秘密", ID: "abc", HTML: "<p>こんにちは</p>"})
	p, err := readPayloadFrom(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "放課後の秘密" || p.HTML != "<p>こんにちは</p>" || p.Width != 1280 || p.Height != 720 {
		t.Fatalf("unexpected payload: %+v", p)
	}
}

func TestReadPayloadMissing(t *testing.T) {
	b := []byte("MZ just a stub without data....")
	if _, err := readPayloadFrom(bytes.NewReader(b), int64(len(b))); err == nil {
		t.Fatal("expected error")
	}
}

func TestStore(t *testing.T) {
	s := newStore(t.TempDir(), `悪い:名前?`, "id-1")
	if v, _ := s.Load("slot1"); v != "" {
		t.Fatal("expected empty")
	}
	if err := s.Save("slot1", `{"a":1}`); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Load("slot1"); v != `{"a":1}` {
		t.Fatal("roundtrip failed:", v)
	}
	if err := s.Delete("slot1"); err != nil {
		t.Fatal(err)
	}
	if got := folderName(`a/b:c`, "x!y"); got != "a_b_c_xy" {
		t.Fatal(got)
	}
}
