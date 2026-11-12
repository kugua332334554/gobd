package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gotd/tl"
)

func TestPrivTL(t *testing.T) {
	u, err := LoadUni(
		filepath.Join("..", "_schema", "layers", "meta.json"),
		filepath.Join("..", "_schema", "layers", "decide.json"),
		filepath.Join("..", "_schema", "layers", "priv.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join("..", "_schema", "layers", "priv.tl"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	s, err := tl.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[uint32]tl.Definition{}
	for _, d := range s.Definitions {
		byID[d.Definition.ID] = d.Definition
	}
	tab := u.Priv
	for _, it := range tab.Named() {
		var id uint64
		if _, err := fmt.Sscanf(it.ID, "%x", &id); err != nil {
			t.Fatal(err)
		}
		d, ok := byID[uint32(id)]
		if !ok {
			t.Errorf("%s %s missing in priv.tl", it.Name, it.ID)
			continue
		}
		if it.Sym == "" {
			continue
		}
		can, ok := u.CanonDef(it.Sym)
		if !ok {
			t.Errorf("%s: unknown sym %s", it.Name, it.Sym)
			continue
		}
		if got, want := shapeOfDef(&d), shapeOfDef(can); got != want {
			t.Errorf("%s: shape\n got %s\nwant %s", it.Name, got, want)
		}
	}
	if len(s.Definitions) != len(tab.Named()) {
		t.Errorf("defs %d, want %d", len(s.Definitions), len(tab.Named()))
	}
}

// shapeOfDef renders a definition shape.
func shapeOfDef(d *tl.Definition) string {
	var b strings.Builder
	b.WriteString("(")
	for i, p := range d.Params {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(p.Name)
		b.WriteString(":")
		if p.Flags {
			b.WriteString("#")
			continue
		}
		if p.Flag != nil {
			b.WriteString(p.Flag.Name)
			b.WriteString(".")
			b.WriteString(strconv.Itoa(p.Flag.Index))
			b.WriteString("?")
		}
		b.WriteString(p.Type.String())
	}
	b.WriteString(")=")
	b.WriteString(d.Type.String())
	return b.String()
}
