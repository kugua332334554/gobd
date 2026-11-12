package gen

import (
	"sort"

	"github.com/gotd/tl"
)

// layDef is an indexed definition.
type layDef struct {
	Key string
	Def tl.Definition
	Cat tl.Category
}

// IsFunc reports method definition.
func (d *layDef) IsFunc() bool { return d.Cat == tl.CategoryFunction }

// layIdx is a schema index.
type layIdx struct {
	sch     *tl.Schema
	byKey   map[string]*layDef
	byID    map[uint32]*layDef
	keys    []string
	members map[string][]*layDef
}

func newIdx(s *tl.Schema) *layIdx {
	i := &layIdx{
		sch:     s,
		byKey:   make(map[string]*layDef, len(s.Definitions)),
		byID:    make(map[uint32]*layDef, len(s.Definitions)),
		members: map[string][]*layDef{},
	}
	for _, sd := range s.Definitions {
		d := &layDef{
			Key: defKey(sd.Definition),
			Def: sd.Definition,
			Cat: sd.Category,
		}
		i.byKey[d.Key] = d
		i.byID[d.Def.ID] = d
		i.keys = append(i.keys, d.Key)
		if d.Cat == tl.CategoryType {
			cls := d.Def.Type.String()
			i.members[cls] = append(i.members[cls], d)
		}
	}
	sort.Strings(i.keys)
	return i
}

// Get returns definition by key.
func (i *layIdx) Get(key string) (*layDef, bool) {
	d, ok := i.byKey[key]
	return d, ok
}

// Members returns class constructors.
func (i *layIdx) Members(class string) []*layDef {
	return i.members[class]
}

// isClassRef reports boxed class reference.
func isClassRef(t tl.Type) bool {
	if t.GenericRef || t.Percent {
		return false
	}
	if t.Name == "" {
		return false
	}
	return t.Name[0] >= 'A' && t.Name[0] <= 'Z'
}

// deref strips nested vectors.
func deref(t tl.Type) tl.Type {
	for t.GenericArg != nil && (t.Name == "vector" || t.Name == "Vector") {
		t = *t.GenericArg
	}
	return t
}

// refs collects referenced definition keys.
func (i *layIdx) refs(d *layDef) []string {
	set := map[string]struct{}{}
	for _, p := range d.Def.Params {
		if p.Flags {
			continue
		}
		t := deref(p.Type)
		if t.GenericRef {
			continue
		}
		if isClassRef(t) {
			for _, m := range i.Members(t.String()) {
				set[m.Key] = struct{}{}
			}
			continue
		}
		name := t.String()
		if t.Percent {
			name = name[1:]
		}
		if m, ok := i.byKey[name]; ok {
			set[m.Key] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
