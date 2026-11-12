package layerx

import (
	"reflect"
	"testing"

	"github.com/gotd/td/bin"
)

func TestTables(t *testing.T) {
	for _, l := range Layers() {
		var n int
		for s := range symNames {
			id, ok := Ctor(l, s)
			if !ok {
				continue
			}
			n++
			got, ok := SymOf(l, id)
			if !ok {
				t.Fatalf("layer %d: id %08x not found", l, id)
			}
			if got != s {
				t.Fatalf("layer %d: sym %s -> id %08x -> %s", l, SymName(s), id, SymName(got))
			}
		}
		t.Logf("layer %d: %d ctors", l, n)
	}
}

// tryPut encodes zero value, skipping invalid ones.
func tryPut(b *bin.Buffer, l Layer, v bin.Object) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return Put(b, l, v) == nil
}

func TestZeroRoundTrip(t *testing.T) {
	var (
		ok, skipped, failed int
	)
	for _, l := range Layers() {
		for s := range symNames {
			if _, ok := Ctor(l, s); !ok {
				continue
			}
			v := newSym(s)
			if v == nil {
				skipped++
				continue
			}
			var b bin.Buffer
			if !tryPut(&b, l, v) {
				skipped++
				continue
			}
			got, err := Get(&b, l)
			if err != nil {
				t.Errorf("layer %d %s: get: %v", l, SymName(s), err)
				failed++
				continue
			}
			if len(b.Buf) != 0 {
				t.Errorf("layer %d %s: %d bytes left", l, SymName(s), len(b.Buf))
				failed++
				continue
			}
			if !reflect.DeepEqual(v, got) {
				t.Errorf("layer %d %s: mismatch", l, SymName(s))
				failed++
				continue
			}
			ok++
			if failed > 10 {
				t.Fatal("too many failures")
			}
		}
	}
	t.Logf("ok=%d skipped=%d failed=%d", ok, skipped, failed)
}
