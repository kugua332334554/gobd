package layerx

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/gotd/td/bin"
	"github.com/gotd/td/tg"
)

func TestFreezeFanout(t *testing.T) {
	v := &tg.User{ID: 9, FirstName: "cat", Verified: true}
	f, err := Freeze(v)
	if err != nil {
		t.Fatal(err)
	}
	if f.Sym() != SymUser {
		t.Fatalf("sym %s", SymName(f.Sym()))
	}
	for _, l := range Layers() {
		var direct, frozen bin.Buffer
		if err := Put(&direct, l, v); err != nil {
			t.Fatalf("layer %d: put: %v", l, err)
		}
		if err := f.Write(&frozen, l); err != nil {
			t.Fatalf("layer %d: write: %v", l, err)
		}
		if !bytes.Equal(direct.Buf, frozen.Buf) {
			t.Fatalf("layer %d: bytes differ", l)
		}
		n, err := f.Size(l)
		if err != nil {
			t.Fatal(err)
		}
		if n != len(frozen.Buf) {
			t.Fatalf("layer %d: size %d != %d", l, n, len(frozen.Buf))
		}
	}
}

func TestPutResClass(t *testing.T) {
	tbl := NewTable()
	if err := tbl.Add(SymHelpGetConfig, func(ctx context.Context, m *Mark) (Out, error) {
		return &tg.Config{Date: 1, Expires: 2}, nil
	}); err != nil {
		t.Fatal(err)
	}
	var b bin.Buffer
	if err := Put(&b, L228, &tg.HelpGetConfigRequest{}); err != nil {
		t.Fatal(err)
	}
	a, err := tbl.Take(L228, &b)
	if err != nil {
		t.Fatal(err)
	}
	var out bin.Buffer
	if err := a.Mark().PutRes(&out, &tg.User{ID: 1}); !errors.Is(err, ErrMismatch) {
		t.Fatalf("expected ErrMismatch, got %v", err)
	}
	out.Reset()
	if err := a.Mark().PutRes(&out, &tg.Config{Date: 1, Expires: 2}); err != nil {
		t.Fatal(err)
	}
}

func TestServe(t *testing.T) {
	tbl := NewTable()
	if err := tbl.Add(SymHelpGetConfig, func(ctx context.Context, m *Mark) (Out, error) {
		return &tg.Config{Date: 1, Expires: 2}, nil
	}); err != nil {
		t.Fatal(err)
	}
	var b bin.Buffer
	if err := Put(&b, L228, &tg.HelpGetConfigRequest{}); err != nil {
		t.Fatal(err)
	}
	a, err := tbl.Take(L228, &b)
	if err != nil {
		t.Fatal(err)
	}
	out, err := tbl.Serve(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(*tg.Config); !ok {
		t.Fatalf("unexpected %T", out)
	}
}
