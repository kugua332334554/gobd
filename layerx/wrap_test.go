package layerx

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/mt"
	"github.com/gotd/td/tg"
)

func newTable(t *testing.T, s Sym) *Table {
	t.Helper()
	tbl := NewTable()
	err := tbl.Add(s, func(ctx context.Context, m *Mark) (Out, error) {
		return &tg.BoolTrue{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

func wrapInvokeWithLayer(t *testing.T, layer int, inner []byte) *bin.Buffer {
	t.Helper()
	var b bin.Buffer
	b.PutID(tg.InvokeWithLayerRequestTypeID)
	b.PutInt(layer)
	b.Buf = append(b.Buf, inner...)
	return &b
}

func TestWrapperLayer(t *testing.T) {
	var inner bin.Buffer
	if err := Put(&inner, L228, &tg.HelpGetConfigRequest{}); err != nil {
		t.Fatal(err)
	}
	tbl := newTable(t, SymHelpGetConfig)
	b := wrapInvokeWithLayer(t, 228, inner.Buf)
	a, err := tbl.Take(L233, b)
	if err != nil {
		t.Fatal(err)
	}
	if a.Layer() != L228 {
		t.Fatalf("layer %d", a.Layer())
	}
	if !a.Proof() {
		t.Fatal("no evidence")
	}
	if len(a.Wraps()) != 1 {
		t.Fatalf("wraps %d", len(a.Wraps()))
	}
	if a.Mark().Sym() != SymHelpGetConfig {
		t.Fatalf("sym %s", SymName(a.Mark().Sym()))
	}
}

func TestWrapperInitConnection(t *testing.T) {
	var inner bin.Buffer
	if err := Put(&inner, L231, &tg.HelpGetConfigRequest{}); err != nil {
		t.Fatal(err)
	}
	var init bin.Buffer
	init.PutID(tg.InitConnectionRequestTypeID)
	init.PutUint32(0)
	init.PutInt(1)
	init.PutString("dev")
	init.PutString("sys")
	init.PutString("app")
	init.PutString("en")
	init.PutString("")
	init.PutString("en")
	init.Buf = append(init.Buf, inner.Buf...)

	tbl := newTable(t, SymHelpGetConfig)
	b := wrapInvokeWithLayer(t, 231, init.Buf)
	a, err := tbl.Take(L233, b)
	if err != nil {
		t.Fatal(err)
	}
	if a.Layer() != L231 {
		t.Fatalf("layer %d", a.Layer())
	}
	if len(a.Wraps()) != 2 {
		t.Fatalf("wraps %d", len(a.Wraps()))
	}
}

func TestWrapperUnknownLayer(t *testing.T) {
	var inner bin.Buffer
	if err := Put(&inner, L228, &tg.HelpGetConfigRequest{}); err != nil {
		t.Fatal(err)
	}
	tbl := newTable(t, SymHelpGetConfig)
	b := wrapInvokeWithLayer(t, 227, inner.Buf)
	if _, err := tbl.Take(L233, b); !errors.Is(err, ErrLayer) {
		t.Fatalf("expected ErrLayer, got %v", err)
	}
}

func TestWrapperGzip(t *testing.T) {
	var inner bin.Buffer
	if err := Put(&inner, L228, &tg.HelpGetConfigRequest{}); err != nil {
		t.Fatal(err)
	}
	var raw bytes.Buffer
	zw := gzip.NewWriter(&raw)
	if _, err := zw.Write(inner.Buf); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var b bin.Buffer
	if err := (&mt.GzipPacked{PackedData: raw.Bytes()}).Encode(&b); err != nil {
		t.Fatal(err)
	}
	tbl := newTable(t, SymHelpGetConfig)
	if _, err := tbl.Take(L228, &b); !errors.Is(err, ErrNoZip) {
		t.Fatalf("expected ErrNoZip, got %v", err)
	}
	b = bin.Buffer{Buf: b.Buf}
	a, err := tbl.TakeOpt(L228, &b, Opts{Unzip: func(data []byte) ([]byte, error) {
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if _, err := out.ReadFrom(zr); err != nil {
			return nil, err
		}
		return out.Bytes(), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if a.Layer() != L228 {
		t.Fatalf("layer %d", a.Layer())
	}
}

func TestOverlay(t *testing.T) {
	tbl := newTable(t, SymHelpGetConfig)
	ov := NewOverlay()
	if err := ov.Map(0x11223344, SymHelpGetConfig); err != nil {
		t.Fatal(err)
	}
	if err := ov.Map(0x11223344, Sym(0)); err == nil {
		t.Fatal("expected error")
	}
	tbl.SetOverlay(ov)
	var b bin.Buffer
	b.PutID(0x11223344)
	a, err := tbl.Take(L228, &b)
	if err != nil {
		t.Fatal(err)
	}
	if a.Mark().Sym() != SymHelpGetConfig {
		t.Fatalf("sym %s", SymName(a.Mark().Sym()))
	}
	if _, ok := ov.Sym(0x55667788); ok {
		t.Fatal("unexpected id")
	}
}

func TestOdd(t *testing.T) {
	tbl := NewTable()
	var got OddView
	tbl.OnOdd(func(v OddView) { got = v })
	var b bin.Buffer
	if err := Put(&b, L228, &tg.HelpGetConfigRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := tbl.Take(L228, &b); !errors.Is(err, ErrOdd) {
		t.Fatalf("expected ErrOdd, got %v", err)
	}
	if got.Layer != L228 {
		t.Fatalf("odd layer %d", got.Layer)
	}
}

func TestPreflight(t *testing.T) {
	tbl := newTable(t, SymHelpGetConfig)
	var b bin.Buffer
	if err := Put(&b, L228, &tg.HelpGetConfigRequest{}); err != nil {
		t.Fatal(err)
	}
	if _, err := tbl.TakeOpt(L228, &b, Opts{Lim: Lim{Bytes: 1, Vec: 1, Agg: 1, Depth: 1}}); !errors.Is(err, ErrLimit) {
		t.Fatalf("expected ErrLimit, got %v", err)
	}
}
