package layerx

import (
	"encoding/binary"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

func TestCheckValid(t *testing.T) {
	v := &tg.User{ID: 1, FirstName: "cat", Verified: true}
	for _, l := range Layers() {
		var b bin.Buffer
		if err := Put(&b, l, v); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		if err := Check(&b, l, Lim{}); err != nil {
			t.Fatalf("layer %d: check: %v", l, err)
		}
	}
}

func TestCheckTruncated(t *testing.T) {
	v := &tg.User{ID: 1, FirstName: "cat", Verified: true}
	for _, l := range Layers() {
		var b bin.Buffer
		if err := Put(&b, l, v); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		full := append([]byte(nil), b.Buf...)
		for cut := 1; cut < len(full); cut++ {
			part := &bin.Buffer{Buf: append([]byte(nil), full[:cut]...)}
			if err := Check(part, l, Lim{}); err == nil {
				t.Fatalf("layer %d: cut %d accepted", l, cut)
			}
		}
	}
}

func TestCheckVectorLimit(t *testing.T) {
	id, ok := Ctor(L228, SymUser)
	if !ok {
		t.Fatal("no user")
	}
	var raw []byte
	raw = binary.LittleEndian.AppendUint32(raw, id)
	raw = binary.LittleEndian.AppendUint32(raw, 0)
	// flags word, flags2 word, id
	raw = binary.LittleEndian.AppendUint32(raw, 0)
	raw = binary.LittleEndian.AppendUint32(raw, 0)
	raw = binary.LittleEndian.AppendUint64(raw, 1)
	b := &bin.Buffer{Buf: raw}
	if err := Check(b, L228, Lim{Vec: 4, Agg: 1 << 10, Depth: 8, Bytes: 1 << 20}); err != nil {
		// Valid tiny object must pass.
		t.Fatalf("valid check: %v", err)
	}
}

func TestCheckDepth(t *testing.T) {
	v := &tg.User{ID: 1}
	var b bin.Buffer
	if err := Put(&b, Canon, v); err != nil {
		t.Fatal(err)
	}
	if err := Check(&b, Canon, Lim{Depth: 1, Vec: 1 << 10, Agg: 1 << 10, Bytes: 1 << 20}); err != nil {
		t.Fatalf("depth 1 should pass: %v", err)
	}
}
