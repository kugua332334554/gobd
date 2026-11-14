package layerx

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

// mustHex decodes hex text with spaces, for golden wire bytes.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(strings.Join(strings.Fields(s), ""))
	if err != nil {
		t.Fatalf("bad hex: %v", err)
	}
	return raw
}

// realMarkup returns a realistic inline keyboard with url, callback and style.
func realMarkup() *tg.ReplyInlineMarkup {
	urlBtn := tg.KeyboardInlineButton{
		Text: "Docs",
		Type: &tg.InlineButtonTypeURL{URL: "https://example.com/docs"},
	}
	var urlStyle tg.KeyboardButtonStyle
	urlStyle.SetBgPrimary(true)
	urlStyle.SetIcon(12345)
	urlBtn.SetStyle(urlStyle)

	cbBtn := tg.KeyboardInlineButton{
		Text: "OK",
		Type: &tg.InlineButtonTypeCallback{Data: []byte("ok")},
	}
	var cbStyle tg.KeyboardButtonStyle
	cbStyle.SetBgSuccess(true)
	cbBtn.SetStyle(cbStyle)

	return &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardInlineButtonRow{
			{Buttons: []tg.KeyboardInlineButton{urlBtn, cbBtn}},
		},
	}
}

// TestMarkupBridgeGolden228 checks the legacy layout byte by byte.
func TestMarkupBridgeGolden228(t *testing.T) {
	m := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardInlineButtonRow{
			{Buttons: []tg.KeyboardInlineButton{{
				Text: "a",
				Type: &tg.InlineButtonTypeDisabled{},
			}}},
		},
	}
	var b bin.Buffer
	if err := Put(&b, L228, m); err != nil {
		t.Fatalf("put: %v", err)
	}
	want := mustHex(t, `
		5402a348
		15c4b51c 01000000
		838b6077
		15c4b51c 01000000
		ff0c177d
		00000000
		01610000
	`)
	if !bytes.Equal(b.Buf, want) {
		t.Fatalf("unexpected bytes\n got %x\nwant %x", b.Buf, want)
	}

	orig := append([]byte(nil), b.Buf...)
	obj, err := Get(&b, L228)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	got, ok := obj.(*tg.ReplyInlineMarkup)
	if !ok {
		t.Fatalf("got %T", obj)
	}
	if len(got.Rows) != 1 || len(got.Rows[0].Buttons) != 1 {
		t.Fatalf("unexpected shape: %+v", got)
	}
	if got.Rows[0].Buttons[0].Text != "a" {
		t.Fatalf("text %q", got.Rows[0].Buttons[0].Text)
	}
	if _, ok := got.Rows[0].Buttons[0].Type.(*tg.InlineButtonTypeDisabled); !ok {
		t.Fatalf("type %T", got.Rows[0].Buttons[0].Type)
	}

	var b2 bin.Buffer
	if err := Put(&b2, L228, got); err != nil {
		t.Fatalf("re-put: %v", err)
	}
	if !bytes.Equal(orig, b2.Buf) {
		t.Fatalf("roundtrip mismatch\n%x\n%x", orig, b2.Buf)
	}
}

// TestMarkupBridgeDecode228 decodes legacy bytes produced by the schema.
func TestMarkupBridgeDecode228(t *testing.T) {
	// replyInlineMarkup with one row of url + callback, both styled.
	wire := mustHex(t, `
		5402a348
		15c4b51c 01000000
		838b6077
		15c4b51c 02000000
		ec250cd8
		00040000
		3034dd4f 03000000
		04 4f70656e 000000
		0c 68747470733a2f2f742e6d65 000000
		60c92be6
		01040000
		3034dd4f 04000000
		02 476f00
		02 dead00
	`)
	b := bin.Buffer{Buf: append([]byte(nil), wire...)}
	obj, err := Get(&b, L228)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(b.Buf) != 0 {
		t.Fatalf("%d bytes left", len(b.Buf))
	}
	m := obj.(*tg.ReplyInlineMarkup)
	if len(m.Rows) != 1 || len(m.Rows[0].Buttons) != 2 {
		t.Fatalf("unexpected shape: %+v", m)
	}

	urlBtn := m.Rows[0].Buttons[0]
	if urlBtn.Text != "Open" {
		t.Fatalf("text %q", urlBtn.Text)
	}
	if !urlBtn.Style.BgPrimary || !urlBtn.Style.BgDanger || urlBtn.Style.BgSuccess {
		t.Fatalf("style %+v", urlBtn.Style)
	}
	u, ok := urlBtn.Type.(*tg.InlineButtonTypeURL)
	if !ok || u.URL != "https://t.me" {
		t.Fatalf("url type %+v", urlBtn.Type)
	}

	cbBtn := m.Rows[0].Buttons[1]
	if cbBtn.Text != "Go" {
		t.Fatalf("text %q", cbBtn.Text)
	}
	if !cbBtn.Style.BgSuccess {
		t.Fatalf("style %+v", cbBtn.Style)
	}
	cb, ok := cbBtn.Type.(*tg.InlineButtonTypeCallback)
	if !ok {
		t.Fatalf("callback type %T", cbBtn.Type)
	}
	if !cb.RequiresPassword || !bytes.Equal(cb.Data, []byte{0xde, 0xad}) {
		t.Fatalf("callback %+v", cb)
	}

	var out bin.Buffer
	if err := Put(&out, L228, m); err != nil {
		t.Fatalf("re-put: %v", err)
	}
	if !bytes.Equal(wire, out.Buf) {
		t.Fatalf("roundtrip mismatch\n got %x\nwant %x", out.Buf, wire)
	}
}

// TestMarkupBridgeInline checks L229-L233 encode/decode stability.
func TestMarkupBridgeInline(t *testing.T) {
	m := &tg.ReplyInlineMarkup{Rows: realMarkup().Rows}
	m.SetForceReply(true)

	for _, l := range []Layer{L229, L230, L231, L232, L233} {
		var b bin.Buffer
		if err := Put(&b, l, m); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		if id := uint32(b.Buf[0]) | uint32(b.Buf[1])<<8 | uint32(b.Buf[2])<<16 | uint32(b.Buf[3])<<24; id != 0xb2b15770 {
			t.Fatalf("layer %d: id %08x", l, id)
		}
		orig := append([]byte(nil), b.Buf...)

		obj, err := Get(&b, l)
		if err != nil {
			t.Fatalf("layer %d: get: %v", l, err)
		}
		if len(b.Buf) != 0 {
			t.Fatalf("layer %d: %d bytes left", l, len(b.Buf))
		}
		got := obj.(*tg.ReplyInlineMarkup)
		if !got.ForceReply {
			t.Fatalf("layer %d: force_reply lost", l)
		}
		if len(got.Rows) != 1 || len(got.Rows[0].Buttons) != 2 {
			t.Fatalf("layer %d: unexpected shape: %+v", l, got)
		}

		var b2 bin.Buffer
		if err := Put(&b2, l, got); err != nil {
			t.Fatalf("layer %d: re-put: %v", l, err)
		}
		if !bytes.Equal(orig, b2.Buf) {
			t.Fatalf("layer %d: roundtrip mismatch\n%x\n%x", l, orig, b2.Buf)
		}
	}
}

// TestMarkupBridgeRealKeyboard checks a real keyboard across both layouts.
func TestMarkupBridgeRealKeyboard(t *testing.T) {
	check := func(t *testing.T, m *tg.ReplyInlineMarkup) {
		t.Helper()
		if len(m.Rows) != 1 || len(m.Rows[0].Buttons) != 2 {
			t.Fatalf("unexpected shape: %+v", m)
		}
		urlBtn := m.Rows[0].Buttons[0]
		if urlBtn.Text != "Docs" {
			t.Fatalf("url text %q", urlBtn.Text)
		}
		if !urlBtn.Style.BgPrimary || urlBtn.Style.Icon != 12345 {
			t.Fatalf("url style %+v", urlBtn.Style)
		}
		u, ok := urlBtn.Type.(*tg.InlineButtonTypeURL)
		if !ok || u.URL != "https://example.com/docs" {
			t.Fatalf("url type %+v", urlBtn.Type)
		}

		cbBtn := m.Rows[0].Buttons[1]
		if cbBtn.Text != "OK" || !cbBtn.Style.BgSuccess {
			t.Fatalf("callback button %+v", cbBtn)
		}
		cb, ok := cbBtn.Type.(*tg.InlineButtonTypeCallback)
		if !ok || !bytes.Equal(cb.Data, []byte("ok")) {
			t.Fatalf("callback type %+v", cbBtn.Type)
		}
	}

	m := realMarkup()
	for _, l := range []Layer{L228, L233} {
		var b bin.Buffer
		if err := Put(&b, l, m); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		orig := append([]byte(nil), b.Buf...)
		obj, err := Get(&b, l)
		if err != nil {
			t.Fatalf("layer %d: get: %v", l, err)
		}
		got := obj.(*tg.ReplyInlineMarkup)
		check(t, got)

		var b2 bin.Buffer
		if err := Put(&b2, l, got); err != nil {
			t.Fatalf("layer %d: re-put: %v", l, err)
		}
		if !bytes.Equal(orig, b2.Buf) {
			t.Fatalf("layer %d: roundtrip mismatch", l)
		}
	}

	// 228 -> 233 -> 228 keeps fields and legacy bytes.
	var legacy bin.Buffer
	if err := Put(&legacy, L228, m); err != nil {
		t.Fatalf("put 228: %v", err)
	}
	orig := append([]byte(nil), legacy.Buf...)

	obj, err := Get(&legacy, L228)
	if err != nil {
		t.Fatalf("get 228: %v", err)
	}
	var modern bin.Buffer
	if err := Put(&modern, L233, obj); err != nil {
		t.Fatalf("put 233: %v", err)
	}
	obj2, err := Get(&modern, L233)
	if err != nil {
		t.Fatalf("get 233: %v", err)
	}
	check(t, obj2.(*tg.ReplyInlineMarkup))

	var back bin.Buffer
	if err := Put(&back, L228, obj2); err != nil {
		t.Fatalf("put back 228: %v", err)
	}
	if !bytes.Equal(orig, back.Buf) {
		t.Fatalf("cross-layer roundtrip mismatch\n%x\n%x", orig, back.Buf)
	}
}

// TestMarkupBridgeLegacyButtons covers switch_inline and degraded types.
func TestMarkupBridgeLegacyButtons(t *testing.T) {
	// switch_inline with same_peer and one peer type.
	wire := mustHex(t, `
		5402a348
		15c4b51c 01000000
		838b6077
		15c4b51c 01000000
		fc991399
		03000000
		02 537700
		01 710000
		15c4b51c 01000000
		ac0f3c83
	`)
	b := bin.Buffer{Buf: append([]byte(nil), wire...)}
	obj, err := Get(&b, L228)
	if err != nil {
		t.Fatalf("get switch_inline: %v", err)
	}
	if len(b.Buf) != 0 {
		t.Fatalf("switch_inline: %d bytes left", len(b.Buf))
	}
	m := obj.(*tg.ReplyInlineMarkup)
	sw, ok := m.Rows[0].Buttons[0].Type.(*tg.InlineButtonTypeSwitchInline)
	if !ok {
		t.Fatalf("type %T", m.Rows[0].Buttons[0].Type)
	}
	if !sw.SamePeer || sw.Query != "q" || len(sw.PeerTypes) != 1 {
		t.Fatalf("switch_inline %+v", sw)
	}
	if _, ok := sw.PeerTypes[0].(*tg.InlineQueryPeerTypePM); !ok {
		t.Fatalf("peer type %T", sw.PeerTypes[0])
	}

	var out bin.Buffer
	if err := Put(&out, L228, m); err != nil {
		t.Fatalf("re-put switch_inline: %v", err)
	}
	if !bytes.Equal(wire, out.Buf) {
		t.Fatalf("switch_inline mismatch\n got %x\nwant %x", out.Buf, wire)
	}

	// request_peer is reply-only, decode must consume it and degrade.
	peer := mustHex(t, `
		5402a348
		15c4b51c 01000000
		838b6077
		15c4b51c 01000000
		f5150f5b
		00000000
		03 526571
		07000000
		008a3b5f 00000000
		01000000
	`)
	b = bin.Buffer{Buf: append([]byte(nil), peer...)}
	obj, err = Get(&b, L228)
	if err != nil {
		t.Fatalf("get request_peer: %v", err)
	}
	if len(b.Buf) != 0 {
		t.Fatalf("request_peer: %d bytes left", len(b.Buf))
	}
	m = obj.(*tg.ReplyInlineMarkup)
	if _, ok := m.Rows[0].Buttons[0].Type.(*tg.InlineButtonTypeDisabled); !ok {
		t.Fatalf("request_peer type %T", m.Rows[0].Buttons[0].Type)
	}
}

// TestMarkupBridgeErrors checks unsupported buttons return wrapped errors.
func TestMarkupBridgeErrors(t *testing.T) {
	bad := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardInlineButtonRow{
			{Buttons: []tg.KeyboardInlineButton{{
				Text: "x",
				Type: &tg.InputInlineButtonTypeURLAuth{URL: "https://x"},
			}}},
		},
	}
	var b bin.Buffer
	err := Put(&b, L228, bad)
	if !errors.Is(err, ErrNoPlan) {
		t.Fatalf("unexpected error: %v", err)
	}

	nilType := &tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardInlineButtonRow{
			{Buttons: []tg.KeyboardInlineButton{{Text: "x"}}},
		},
	}
	b.Reset()
	if err := Put(&b, L228, nilType); !errors.Is(err, ErrNil) {
		t.Fatalf("unexpected error: %v", err)
	}
}
