package gen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gotd/tl"
)

// DiffKind is a per-layer plan class.
type DiffKind uint8

const (
	// Same wire id and shape.
	Same DiffKind = iota
	// Id differs, shape same.
	Id
	// Body fields projectable.
	Body
	// Relay nested changes.
	Relay
	// None absent in layer.
	None
	// Hook needs decision.
	Hook
	// Gone absent in canon.
	Gone
)

func (k DiffKind) String() string {
	switch k {
	case Same:
		return "same"
	case Id:
		return "id"
	case Body:
		return "body"
	case Relay:
		return "relay"
	case None:
		return "none"
	case Hook:
		return "hook"
	case Gone:
		return "gone"
	}
	return "?"
}

// typeSig is canonical type text.
func typeSig(t tl.Type) string {
	s := t.String()
	if t.Bare {
		s = "b" + s
	}
	if t.GenericRef {
		s = "!" + s
	}
	return s
}

// defShape is wire layout text.
func defShape(d *layDef) string {
	var b strings.Builder
	b.WriteString(d.Cat.String())
	b.WriteByte('|')
	if d.Def.Base {
		b.WriteString("base|")
	}
	if len(d.Def.GenericParams) > 0 {
		b.WriteString("g{")
		b.WriteString(strings.Join(d.Def.GenericParams, ","))
		b.WriteString("}|")
	}
	b.WriteString("=")
	b.WriteString(d.Def.Type.String())
	b.WriteByte('|')
	for _, p := range d.Def.Params {
		if p.Flags {
			b.WriteString("#;")
			continue
		}
		if p.Flag != nil {
			b.WriteString(fmt.Sprintf("%s.%d?", p.Flag.Name, p.Flag.Index))
		}
		if p.Type.GenericRef {
			b.WriteString("!")
		}
		b.WriteString(typeSig(p.Type))
		b.WriteByte(';')
	}
	return b.String()
}

// paramMap indexes params by raw name.
func paramMap(d *layDef) map[string]tl.Parameter {
	m := make(map[string]tl.Parameter, len(d.Def.Params))
	for _, p := range d.Def.Params {
		if p.Flags || p.Name == "" {
			continue
		}
		m[p.Name] = p
	}
	return m
}

// directKind compares two definitions without closure.
func directKind(can, lay *layDef, conv func(string) (string, bool)) DiffKind {
	if defShape(can) == defShape(lay) {
		if can.Def.ID == lay.Def.ID {
			return Same
		}
		return Id
	}
	if can.IsFunc() != lay.IsFunc() {
		return Hook
	}
	if len(can.Def.GenericParams) > 0 || len(lay.Def.GenericParams) > 0 {
		return Hook
	}
	cm, lm := paramMap(can), paramMap(lay)
	for name, cp := range cm {
		lp, ok := lm[name]
		if !ok {
			// Missing required fields are encode-rejected later.
			continue
		}
		if typeSig(cp.Type) != typeSig(lp.Type) {
			if conv != nil {
				if _, ok := conv(name); ok {
					continue
				}
			}
			return Hook
		}
	}
	return Body
}

// ClassReport is a classification result.
type ClassReport struct {
	Layer Layer
	Kinds map[string]DiffKind
	Refs  map[string][]string
}

// Count counts kinds.
func (r *ClassReport) Count() map[DiffKind]int {
	c := map[DiffKind]int{}
	for _, k := range r.Kinds {
		c[k]++
	}
	return c
}

// KindsOf returns sorted keys by kind.
func (r *ClassReport) KindsOf(k DiffKind) []string {
	var out []string
	for key, kind := range r.Kinds {
		if kind == k {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

// Classify builds plan kinds for one layer.
func (u *Uni) Classify(l *LayerSpec) *ClassReport {
	r := &ClassReport{
		Layer: l.Meta.N,
		Kinds: make(map[string]DiffKind, len(u.CanIdx.keys)),
		Refs:  make(map[string][]string, len(u.CanIdx.keys)),
	}
	for _, key := range u.CanIdx.keys {
		can := u.CanIdx.byKey[key]
		if u.Dec.Skip(l.Meta.N, key) {
			r.Kinds[key] = None
			continue
		}
		lay, ok := l.Idx.Get(key)
		if !ok {
			r.Kinds[key] = None
			continue
		}
		conv := func(field string) (string, bool) {
			return u.Dec.Conv(l.Meta.N, key, field)
		}
		r.Kinds[key] = directKind(can, lay, conv)
	}
	// Transitively downgrade parents of changed nodes.
	for _, key := range u.CanIdx.keys {
		can := u.CanIdx.byKey[key]
		refs := l.Idx.refsFrom(can)
		r.Refs[key] = refs
	}
	for {
		changed := false
		for _, key := range u.CanIdx.keys {
			k := r.Kinds[key]
			if k != Same && k != Id {
				continue
			}
			for _, ref := range r.Refs[key] {
				rk, ok := r.Kinds[ref]
				if !ok {
					// Referent not in canon: treat as changed.
					rk = None
				}
				if rk != Same {
					r.Kinds[key] = Relay
					changed = true
					break
				}
			}
		}
		if !changed {
			break
		}
	}
	return r
}

// refsFrom resolves canonical refs via this index.
func (i *layIdx) refsFrom(can *layDef) []string {
	d := &layDef{Key: can.Key, Def: can.Def, Cat: can.Cat}
	return i.refs(d)
}
