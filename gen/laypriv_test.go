package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotd/tl"
)

func TestPrivTL(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "_schema", "layers", "priv.tl"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	s, err := tl.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[uint32]string{}
	for _, d := range s.Definitions {
		byID[d.Definition.ID] = d.Definition.Name
	}
	tab, err := LoadPriv(filepath.Join("..", "_schema", "layers", "priv.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range tab.Named() {
		var id uint64
		if _, err := fmt.Sscanf(it.ID, "%x", &id); err != nil {
			t.Fatal(err)
		}
		if _, ok := byID[uint32(id)]; !ok {
			t.Errorf("%s %s missing in priv.tl", it.Name, it.ID)
		}
	}
	if len(s.Definitions) != len(tab.Named()) {
		t.Errorf("defs %d, want %d", len(s.Definitions), len(tab.Named()))
	}
}
