package gen

import (
	"fmt"

	"github.com/gotd/tl"
)

// Scan kinds.
const (
	scPrim = iota
	scFlags
	scObj
	scVecPrim
	scVecObj
	scVecVecPrim
	scVecVecObj
)

// Prim kinds.
const (
	pNone = iota
	pInt
	pLong
	pInt53
	pInt128
	pInt256
	pDouble
	pString
	pBytes
	pBool
)

// scanField is one wire slot.
type scanField struct {
	kind uint8
	prim uint8
	bit  int8
	word uint8
	sym  uint32
}

// primKind maps TL name.
func primKind(name string) uint8 {
	switch name {
	case "int", "int32":
		return pInt
	case "long", "int64":
		return pLong
	case "int53":
		return pInt53
	case "int128":
		return pInt128
	case "int256":
		return pInt256
	case "double":
		return pDouble
	case "string":
		return pString
	case "bytes":
		return pBytes
	case "Bool", "bool", "true", "false":
		return pBool
	}
	return pNone
}

// scanSlot resolves one parameter.
func (l *LayGen) scanSlot(t tl.Type) (kind uint8, prim uint8, sym uint32, ok bool) {
	depth := 0
	for t.GenericArg != nil && (t.Name == "vector" || t.Name == "Vector") {
		depth++
		t = *t.GenericArg
	}
	if t.GenericRef {
		return 0, 0, 0, false
	}
	if isClassRef(t) {
		switch depth {
		case 0:
			return scObj, 0, 0, true
		case 1:
			return scVecObj, 0, 0, true
		case 2:
			return scVecVecObj, 0, 0, true
		}
		return 0, 0, 0, false
	}
	name := t.String()
	if t.Percent {
		name = name[1:]
	}
	if p := primKind(name); p != pNone {
		switch depth {
		case 0:
			return scPrim, p, 0, true
		case 1:
			return scVecPrim, p, 0, true
		case 2:
			return scVecVecPrim, p, 0, true
		}
		return 0, 0, 0, false
	}
	if d, ok := l.U.CanIdx.Get(name); ok {
		switch depth {
		case 0:
			return scObj, 0, d.Def.ID, true
		case 1:
			return scVecObj, 0, d.Def.ID, true
		}
	}
	return 0, 0, 0, false
}

// scanFields builds wire slots.
func (l *LayGen) scanFields(d *layDef) ([]scanField, error) {
	words := map[string]uint8{}
	for _, p := range d.Def.Params {
		if p.Flags {
			if _, ok := words[p.Name]; !ok {
				words[p.Name] = uint8(len(words))
			}
		}
	}
	var out []scanField
	for _, p := range d.Def.Params {
		if p.Flags {
			out = append(out, scanField{kind: scFlags, prim: pNone, bit: -1, word: words[p.Name]})
			continue
		}
		t := p.Type
		if t.GenericArg == nil && t.Name == "true" {
			continue
		}
		kind, prim, sym, ok := l.scanSlot(t)
		if !ok {
			return nil, fmt.Errorf("scan %s: field %s type %s", d.Key, p.Name, t.String())
		}
		f := scanField{kind: kind, prim: prim, sym: sym, bit: -1}
		if p.Flag != nil {
			f.bit = int8(p.Flag.Index)
			f.word = words[p.Flag.Name]
		}
		out = append(out, f)
	}
	return out, nil
}

// buildScan prepares scan tables.
func (l *LayGen) buildScan() error {
	if l.scanCanon != nil {
		return nil
	}
	l.scanCanon = map[Sym][]scanField{}
	for _, key := range l.U.CanIdx.keys {
		d := l.U.CanIdx.byKey[key]
		if len(d.Def.GenericParams) > 0 {
			continue
		}
		fields, err := l.scanFields(d)
		if err != nil {
			return err
		}
		l.scanCanon[Sym(d.Def.ID)] = fields
	}
	l.scanLayers = map[Layer]map[Sym][]scanField{}
	for _, spec := range l.U.Layers {
		rep := l.reports[spec.Meta.N]
		m := map[Sym][]scanField{}
		for _, key := range l.U.CanIdx.keys {
			k := rep.Kinds[key]
			if k != Body && k != Relay {
				continue
			}
			d, ok := spec.Idx.Get(key)
			if !ok {
				continue
			}
			if len(d.Def.GenericParams) > 0 {
				continue
			}
			fields, err := l.scanFields(d)
			if err != nil {
				return err
			}
			m[Sym(l.U.CanIdx.byKey[key].Def.ID)] = fields
		}
		l.scanLayers[spec.Meta.N] = m
	}
	return nil
}

// scanFile emits scan tables.
func (l *LayGen) scanFile(pkg string) []byte {
	f := newFile(pkg, "")
	f.p("// scanCanon is canonical wire table.\n")
	f.p("var scanCanon = map[Sym][]scanField{\n")
	for _, key := range l.U.CanIdx.keys {
		d := l.U.CanIdx.byKey[key]
		f.p("\t%s: {", l.SymConst(key))
		writeScanFields(f, l.scanCanon[Sym(d.Def.ID)])
		f.p("},\n")
	}
	f.p("}\n\n")

	for _, spec := range l.U.Layers {
		m := l.scanLayers[spec.Meta.N]
		if len(m) == 0 {
			continue
		}
		f.p("// scanL%d overrides layer %d.\n", spec.Meta.N, spec.Meta.N)
		f.p("var scanL%d = map[Sym][]scanField{\n", spec.Meta.N)
		for _, key := range l.U.CanIdx.keys {
			d := l.U.CanIdx.byKey[key]
			fields, ok := m[Sym(d.Def.ID)]
			if !ok {
				continue
			}
			f.p("\t%s: {", l.SymConst(key))
			writeScanFields(f, fields)
			f.p("},\n")
		}
		f.p("}\n\n")
	}

	f.p("// scanFor returns wire slots.\n")
	f.p("func scanFor(l Layer, s Sym) []scanField {\n")
	f.p("\tswitch l {\n")
	for _, spec := range l.U.Layers {
		if len(l.scanLayers[spec.Meta.N]) == 0 {
			continue
		}
		f.p("\tcase L%d:\n\t\tif f, ok := scanL%d[s]; ok {\n\t\t\treturn f\n\t\t}\n",
			spec.Meta.N, spec.Meta.N)
	}
	f.p("\t}\n\treturn scanCanon[s]\n}\n")
	return f.bytes()
}

// writeScanFields emits literal fields.
func writeScanFields(f *file, fields []scanField) {
	for i, s := range fields {
		if i > 0 {
			f.p(", ")
		}
		f.p("{%d, %d, %d, %d, 0x%08x}", s.kind, s.prim, s.bit, s.word, s.sym)
	}
}
