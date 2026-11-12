package gen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-faster/errors"

	"github.com/gotd/tl"
)

// Layer is TL layer number.
type Layer uint16

// Sym is canonical type id.
type Sym uint32

// LayerMeta is one meta.json entry.
type LayerMeta struct {
	N     Layer  `json:"n"`
	File  string `json:"file"`
	Repo  string `json:"repo"`
	Ref   string `json:"ref"`
	Date  string `json:"date"`
	SHA   string `json:"sha"`
	Bytes int    `json:"bytes"`
}

// LayerIndex is meta.json.
type LayerIndex struct {
	Canon     Layer       `json:"canon"`
	CanonFile string      `json:"canonFile"`
	Legacy    string      `json:"legacy"`
	SrcPath   string      `json:"srcPath"`
	Layers    []LayerMeta `json:"layers"`
}

// LayerSpec is parsed layer schema.
type LayerSpec struct {
	Meta LayerMeta
	Sch  *tl.Schema
	Idx  *layIdx
}

// Uni is the multi-layer schema universe.
type Uni struct {
	Meta   LayerIndex
	Canon  Layer
	CanSch *tl.Schema
	CanIdx *layIdx
	Layers []*LayerSpec
	Dec    *Decide
	Priv   *PrivTab
}

// defKey is qualified definition name.
func defKey(d tl.Definition) string {
	return definitionType(d)
}

func loadSchema(path string) (*tl.Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := tl.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrapf(err, "parse %s", path)
	}
	return s, nil
}

// mergeSchema copies definitions missing in target.
func mergeSchema(target, src *tl.Schema) {
Defs:
	for _, d := range src.Definitions {
		for _, td := range target.Definitions {
			if td.Definition.ID == d.Definition.ID {
				continue Defs
			}
		}
		target.Definitions = append(target.Definitions, d)
	}
Classes:
	for _, c := range src.Classes {
		for _, tc := range target.Classes {
			if tc.Name == c.Name {
				continue Classes
			}
		}
		target.Classes = append(target.Classes, c)
	}
}

// LoadUni reads meta.json and all schemas.
func LoadUni(metaPath, decidePath, privPath string) (*Uni, error) {
	dec, err := LoadDecide(decidePath)
	if err != nil {
		return nil, err
	}
	if err := dec.unknown(); err != nil {
		return nil, err
	}
	priv, err := LoadPriv(privPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	var idx LayerIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, errors.Wrap(err, "meta.json")
	}
	base := filepath.Dir(metaPath)

	canonPath := filepath.Join(base, idx.CanonFile)
	canSch, err := loadSchema(canonPath)
	if err != nil {
		return nil, errors.Wrap(err, "canonical")
	}
	var legacy *tl.Schema
	if idx.Legacy != "" {
		legacy, err = loadSchema(filepath.Join(base, idx.Legacy))
		if err != nil {
			return nil, errors.Wrap(err, "legacy")
		}
	}

	u := &Uni{
		Meta:   idx,
		Canon:  idx.Canon,
		CanSch: canSch,
		CanIdx: newIdx(canSch),
		Dec:    dec,
		Priv:   priv,
	}
	sort.Slice(idx.Layers, func(i, j int) bool { return idx.Layers[i].N < idx.Layers[j].N })
	for _, m := range idx.Layers {
		p := filepath.Join(base, m.File)
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(raw)
		if got := hex.EncodeToString(sum[:]); m.SHA != "" && got != m.SHA {
			return nil, errors.Errorf("%s: sha mismatch %s != %s", m.File, got, m.SHA)
		}
		s, err := tl.Parse(bytes.NewReader(raw))
		if err != nil {
			return nil, errors.Wrapf(err, "parse %s", m.File)
		}
		if legacy != nil {
			mergeSchema(s, legacy)
		}
		s.Layer = int(m.N)
		u.Layers = append(u.Layers, &LayerSpec{
			Meta: m,
			Sch:  s,
			Idx:  newIdx(s),
		})
	}
	if len(u.Layers) == 0 {
		return nil, errors.New("no layers")
	}
	if err := u.check(); err != nil {
		return nil, err
	}
	return u, nil
}

// check verifies canonical parity and id collisions.
func (u *Uni) check() error {
	last := u.Layers[len(u.Layers)-1]
	if last.Meta.N != u.Canon {
		return errors.Errorf("last layer %d != canon %d", last.Meta.N, u.Canon)
	}
	for _, k := range u.CanIdx.keys {
		a := u.CanIdx.byKey[k]
		b, ok := last.Idx.byKey[k]
		if !ok {
			return errors.Errorf("canon has %s, last layer does not", k)
		}
		if a.Def.ID != b.Def.ID {
			return errors.Errorf("%s: canon id %x, layer id %x", k, a.Def.ID, b.Def.ID)
		}
	}
	for i := 1; i < len(u.Layers); i++ {
		if u.Layers[i].Meta.N <= u.Layers[i-1].Meta.N {
			return errors.Errorf("layer order: %d after %d", u.Layers[i].Meta.N, u.Layers[i-1].Meta.N)
		}
	}
	return nil
}

// LayerOf returns layer spec by number.
func (u *Uni) LayerOf(n Layer) *LayerSpec {
	for _, l := range u.Layers {
		if l.Meta.N == n {
			return l
		}
	}
	return nil
}

// CanonKeys returns sorted canonical definition keys.
func (u *Uni) CanonKeys() []string { return u.CanIdx.keys }

// CanonDef returns canonical definition.
func (u *Uni) CanonDef(key string) (*tl.Definition, bool) {
	d, ok := u.CanIdx.Get(key)
	if !ok {
		return nil, false
	}
	out := d.Def
	return &out, true
}

// LayerDef returns layer definition.
func (u *Uni) LayerDef(n Layer, key string) (*tl.Definition, bool) {
	spec := u.LayerOf(n)
	if spec == nil {
		return nil, false
	}
	d, ok := spec.Idx.Get(key)
	if !ok {
		return nil, false
	}
	out := d.Def
	return &out, true
}

// LayerKeys returns sorted layer keys.
func (u *Uni) LayerKeys(n Layer) []string {
	spec := u.LayerOf(n)
	if spec == nil {
		return nil
	}
	return spec.Idx.keys
}

func (u *Uni) String() string {
	return fmt.Sprintf("canon=%d layers=%d defs=%d", u.Canon, len(u.Layers), len(u.CanIdx.keys))
}
