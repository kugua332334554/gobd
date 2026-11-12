package layerx

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

func TestUserGolden(t *testing.T) {
	v := &tg.User{
		ID:         123,
		AccessHash: 456,
		FirstName:  "cat",
		Username:   "mi",
		Verified:   true,
		Bot:        true,
	}
	for _, l := range Layers() {
		id, ok := Ctor(l, SymUser)
		if !ok {
			t.Fatalf("layer %d: no user", l)
		}
		var b bin.Buffer
		if err := Put(&b, l, v); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		if got := uint32(b.Buf[0]) | uint32(b.Buf[1])<<8 | uint32(b.Buf[2])<<16 | uint32(b.Buf[3])<<24; got != id {
			t.Fatalf("layer %d: id %08x, want %08x", l, got, id)
		}
		orig := append([]byte(nil), b.Buf...)
		obj, err := Get(&b, l)
		if err != nil {
			t.Fatalf("layer %d: get: %v", l, err)
		}
		var b2 bin.Buffer
		if err := Put(&b2, l, obj); err != nil {
			t.Fatalf("layer %d: re-put: %v", l, err)
		}
		if !bytes.Equal(orig, b2.Buf) {
			t.Fatalf("layer %d: unstable bytes\n%x\n%x", l, orig, b2.Buf)
		}
	}

	// Canonical layer must match the tg encoder exactly.
	var canonical, direct bin.Buffer
	if err := v.Encode(&canonical); err != nil {
		t.Fatal(err)
	}
	if err := Put(&direct, Canon, v); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonical.Buf, direct.Buf) {
		t.Fatal("canonical bytes differ")
	}
}

func TestTextStrHook(t *testing.T) {
	v := &tg.BotVerification{
		Icon:        1,
		BotID:       2,
		Description: tg.TextWithEntities{Text: "cat"},
	}
	for _, l := range Layers() {
		var b bin.Buffer
		if err := Put(&b, l, v); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		obj, err := Get(&b, l)
		if err != nil {
			t.Fatalf("layer %d: get: %v", l, err)
		}
		got := obj.(*tg.BotVerification)
		if got.Description.Text != "cat" {
			t.Fatalf("layer %d: description %q", l, got.Description.Text)
		}
	}
}

func TestRejectNewField(t *testing.T) {
	v := &tg.User{ID: 1, BotForumSidebar: true}
	var b bin.Buffer
	err := Put(&b, L228, v)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrAbsent) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChannelGolden(t *testing.T) {
	v := &tg.Channel{
		ID:         7,
		AccessHash: 8,
		Title:      "cat",
		Username:   "mi",
		Photo:      &tg.ChatPhotoEmpty{},
		Flags:      bin.Fields(0),
	}
	for _, l := range Layers() {
		var b bin.Buffer
		if err := Put(&b, l, v); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		orig := append([]byte(nil), b.Buf...)
		obj, err := Get(&b, l)
		if err != nil {
			t.Fatalf("layer %d: get: %v", l, err)
		}
		var b2 bin.Buffer
		if err := Put(&b2, l, obj); err != nil {
			t.Fatalf("layer %d: re-put: %v", l, err)
		}
		if !bytes.Equal(orig, b2.Buf) {
			t.Fatalf("layer %d: unstable bytes", l)
		}
	}
}
