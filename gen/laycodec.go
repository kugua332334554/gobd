package gen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gotd/tl"
)

// emitUnit is one emitted codec pair.
type emitUnit struct {
	idx     int
	key     string
	goName  string
	lay     *layDef
	putName string
	getName string
	layers  []Layer
	conv    map[string]string
	masks   map[string]bool
}

// buildUnits collects custom codecs.
func (l *LayGen) buildUnits() {
	if l.custom != nil {
		return
	}
	l.layerUnits = map[Layer][]*emitUnit{}
	l.hooks = map[Layer]map[string]bool{}
	for _, spec := range l.U.Layers {
		rep := l.reports[spec.Meta.N]
		l.hooks[spec.Meta.N] = map[string]bool{}
		for _, key := range l.U.CanIdx.keys {
			kind := rep.Kinds[key]
			if kind == Hook {
				l.hooks[spec.Meta.N][key] = true
			}
			if kind != Body && kind != Relay {
				continue
			}
			lay, ok := spec.Idx.Get(key)
			if !ok {
				continue
			}
			sd, ok := l.structs[key]
			if !ok {
				continue
			}
			conv, masks := l.decisionFor(spec.Meta.N, key, lay, sd)
			dk := key + "|" + defShape(lay) + "|" + decDigest(conv, masks)
			u := l.unitOf[dk]
			if u == nil {
				u = &emitUnit{
					idx:    len(l.custom),
					key:    key,
					goName: sd.Name,
					lay:    lay,
					conv:   conv,
					masks:  masks,
				}
				u.putName = fmt.Sprintf("putZ%d", u.idx)
				u.getName = fmt.Sprintf("getZ%d", u.idx)
				l.unitOf[dk] = u
				l.custom = append(l.custom, u)
			}
			u.layers = append(u.layers, spec.Meta.N)
			l.layerUnits[spec.Meta.N] = append(l.layerUnits[spec.Meta.N], u)
		}
	}
	l.buildSkips()
}

func (l *LayGen) unitsOf(n Layer) []*emitUnit { return l.layerUnits[n] }

// decisionFor collects conversions and masks.
func (l *LayGen) decisionFor(n Layer, key string, lay *layDef, sd *structDef) (map[string]string, map[string]bool) {
	conv := map[string]string{}
	for i := range sd.Fields {
		cf := &sd.Fields[i]
		if c, ok := l.U.Dec.Conv(n, key, cf.RawName); ok {
			conv[cf.RawName] = c
		}
	}
	inLayer := map[string]bool{}
	for _, p := range lay.Def.Params {
		inLayer[p.Name] = true
	}
	masks := map[string]bool{}
	for i := range sd.Fields {
		cf := &sd.Fields[i]
		if inLayer[cf.RawName] {
			continue
		}
		if l.U.Dec.Mask(n, key, cf.RawName) {
			masks[cf.RawName] = true
		}
	}
	return conv, masks
}

// decDigest orders decisions.
func decDigest(conv map[string]string, masks map[string]bool) string {
	ck := make([]string, 0, len(conv))
	for k := range conv {
		ck = append(ck, k)
	}
	sort.Strings(ck)
	mk := make([]string, 0, len(masks))
	for k := range masks {
		mk = append(mk, k)
	}
	sort.Strings(mk)
	var b strings.Builder
	for _, k := range ck {
		b.WriteString(k + "=" + conv[k] + ";")
	}
	for _, k := range mk {
		b.WriteString("m" + k + ";")
	}
	return b.String()
}

// goType qualifies canonical Go type.
func goType(cf *fieldDef) string {
	if cf.Encoder {
		return "tg." + cf.Type
	}
	return cf.Type
}

// putFile emits encoders.
func (l *LayGen) putFile(pkg string) []byte {
	f := newFile(pkg, layImports())
	for _, u := range l.custom {
		l.emitPut(f, u)
	}
	return f.bytes()
}

// getFile emits decoders.
func (l *LayGen) getFile(pkg string) []byte {
	f := newFile(pkg, layImports())
	for _, u := range l.custom {
		l.emitGet(f, u)
	}
	return f.bytes()
}

func layImports() string {
	return imports(
		fmt.Sprintf("%q", "fmt"),
		"",
		fmt.Sprintf("%q", "github.com/gotd/td/bin"),
		"",
		fmt.Sprintf("%q", "github.com/gotd/td/tg"),
	) + "\nvar (\n\t_ = fmt.Stringer(nil)\n\t_ = bin.Buffer{}\n\t_ = tg.User{}\n)\n"
}

// fieldSet indexes canonical fields by raw name.
func fieldSet(sd *structDef) map[string]*fieldDef {
	m := make(map[string]*fieldDef, len(sd.Fields))
	for i := range sd.Fields {
		f := &sd.Fields[i]
		m[f.RawName] = f
	}
	return m
}

func hasCond(sd *structDef) bool {
	for _, f := range sd.Fields {
		if f.Conditional {
			return true
		}
	}
	return false
}

// flagsVars lists marker names in order.
func flagsVars(d *layDef) []string {
	var names []string
	seen := map[string]bool{}
	for _, p := range d.Def.Params {
		if p.Flag == nil || seen[p.Flag.Name] {
			continue
		}
		seen[p.Flag.Name] = true
		names = append(names, p.Flag.Name)
	}
	return names
}

func flagVar(names []string, name string) string {
	for i, n := range names {
		if n == name {
			return fmt.Sprintf("fl%d", i)
		}
	}
	return "fl0"
}

func flagName(p tl.Parameter) string {
	if p.Flag == nil {
		return ""
	}
	return p.Flag.Name
}

// unencodable reports a required field missing from layer.
func unencodable(lay *layDef, sd *structDef) (string, bool) {
	inLayer := map[string]bool{}
	for _, p := range lay.Def.Params {
		inLayer[p.Name] = true
	}
	for i := range sd.Fields {
		cf := &sd.Fields[i]
		if cf.Type == "bin.Fields" {
			continue
		}
		if inLayer[cf.RawName] {
			continue
		}
		if !cf.Conditional {
			return cf.RawName, true
		}
	}
	return "", false
}

// emitPut writes one encoder.
func (l *LayGen) emitPut(f *file, u *emitUnit) {
	sd := l.structs[u.key]
	cf := fieldSet(sd)
	names := flagsVars(u.lay)
	f.p("// %s 编码\n", u.putName)
	if field, ok := unencodable(u.lay, sd); ok {
		f.p("func %s(b *bin.Buffer, l Layer, v *tg.%s) error {\n", u.putName, sd.Name)
		f.p("\treturn fmt.Errorf(\"%%w: field %%q\", ErrAbsent, %q)\n", field)
		f.p("}\n\n")
		return
	}
	f.p("func %s(b *bin.Buffer, l Layer, v *tg.%s) error {\n", u.putName, sd.Name)
	f.p("\tif v == nil {\n\t\treturn ErrNil\n\t}\n")
	if hasCond(sd) {
		f.p("\tv.SetFlags()\n")
	}
	for i := range names {
		f.p("\tvar fl%d uint32\n", i)
	}
	for _, p := range u.lay.Def.Params {
		if p.Flag == nil {
			continue
		}
		c := cf[p.Name]
		if c == nil {
			continue
		}
		v := flagVar(names, p.Flag.Name)
		switch {
		case c.ConditionalBool:
			f.p("\tif v.%s {\n\t\t%s |= 1 << %d\n\t}\n", c.Name, v, p.Flag.Index)
		case c.Conditional:
			f.p("\tif v.%s.Has(%d) {\n\t\t%s |= 1 << %d\n\t}\n",
				c.ConditionalField, c.ConditionalIndex, v, p.Flag.Index)
		default:
			f.p("\t%s |= 1 << %d\n", v, p.Flag.Index)
		}
	}
	for _, p := range u.lay.Def.Params {
		if p.Flags {
			f.p("\tb.PutUint32(%s)\n", flagVar(names, p.Name))
			continue
		}
		c := cf[p.Name]
		if c == nil {
			if p.Flag == nil {
				f.p("\t\treturn fmt.Errorf(\"%%w: field %%q\", ErrNoPlan, %q)\n", p.Name)
			}
			continue
		}
		v := flagVar(names, flagName(p))
		switch {
		case p.Flag != nil && c.ConditionalBool:
			// Bit carries the value.
		case p.Flag != nil:
			f.p("\tif %s&(1<<%d) != 0 {\n", v, p.Flag.Index)
			l.emitPutValue(f, c, "\t\t", u.conv[c.RawName])
			f.p("\t}\n")
		case c.ConditionalBool:
			f.p("\tb.PutBool(v.%s)\n", c.Name)
		case c.Conditional:
			f.p("\tif !v.%s.Has(%d) {\n\t\treturn fmt.Errorf(\"%%w: field %%q\", ErrAbsent, %q)\n\t}\n",
				c.ConditionalField, c.ConditionalIndex, p.Name)
			l.emitPutValue(f, c, "\t", u.conv[c.RawName])
		default:
			l.emitPutValue(f, c, "\t", u.conv[c.RawName])
		}
	}
	// Reject canonical-only fields that are set.
	inLayer := map[string]bool{}
	for _, p := range u.lay.Def.Params {
		inLayer[p.Name] = true
	}
	for i := range sd.Fields {
		c := &sd.Fields[i]
		if inLayer[c.RawName] || !c.Conditional || u.masks[c.RawName] {
			continue
		}
		f.p("\tif v.%s.Has(%d) {\n\t\treturn fmt.Errorf(\"%%w: field %%q\", ErrAbsent, %q)\n\t}\n",
			c.ConditionalField, c.ConditionalIndex, c.RawName)
	}
	f.p("\treturn nil\n}\n\n")
}

// emitPutValue writes field value at indent.
func (l *LayGen) emitPutValue(f *file, cf *fieldDef, ind, conv string) {
	expr := "v." + cf.Name
	if cf.Vector {
		if cf.BareVector {
			f.p("%sb.PutInt(len(%s))\n", ind, expr)
		} else {
			f.p("%sb.PutVectorHeader(len(%s))\n", ind, expr)
		}
		f.p("%sfor _, x := range %s {\n", ind, expr)
		l.emitPutElem(f, cf, "x", ind+"\t", conv)
		f.p("%s}\n", ind)
		return
	}
	l.emitPutElem(f, cf, expr, ind, conv)
}

// emitPutElem writes one element.
func (l *LayGen) emitPutElem(f *file, cf *fieldDef, expr, ind, conv string) {
	switch {
	case conv != "":
		f.p("%sif err := put%s(b, %s); err != nil {\n", ind, conv, expr)
		f.p("%s\treturn fmt.Errorf(\"encode: %%w\", err)\n", ind)
		f.p("%s}\n", ind)
	case cf.Encoder && cf.Interface != "":
		f.p("%sif err := putObj(b, l, %s); err != nil {\n", ind, expr)
		f.p("%s\treturn fmt.Errorf(\"encode: %%w\", err)\n", ind)
		f.p("%s}\n", ind)
	case cf.Encoder:
		f.p("%sif err := putObj(b, l, &%s); err != nil {\n", ind, expr)
		f.p("%s\treturn fmt.Errorf(\"encode: %%w\", err)\n", ind)
		f.p("%s}\n", ind)
	default:
		f.p("%sb.Put%s(%s)\n", ind, cf.Func, expr)
	}
}

// emitGet writes one decoder.
func (l *LayGen) emitGet(f *file, u *emitUnit) {
	sd := l.structs[u.key]
	cf := fieldSet(sd)
	names := flagsVars(u.lay)
	f.p("// %s 解码\n", u.getName)
	f.p("func %s(b *bin.Buffer, l Layer, v *tg.%s) error {\n", u.getName, sd.Name)
	for i := range names {
		f.p("\tvar fl%d uint32\n", i)
	}
	for _, p := range u.lay.Def.Params {
		if p.Flags {
			f.p("\t{\n\t\tx, err := b.Uint32()\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\t%s = x\n\t}\n",
				flagVar(names, p.Name))
			continue
		}
		c := cf[p.Name]
		v := flagVar(names, flagName(p))
		if p.Flag != nil {
			f.p("\tif %s&(1<<%d) != 0 {\n", v, p.Flag.Index)
		} else {
			f.p("\t{\n")
		}
		switch {
		case c == nil:
			l.emitSkipType(f, p.Type, "\t\t")
		case p.Flag != nil && c.ConditionalBool:
			f.p("\t\tv.%s = true\n", c.Name)
		default:
			l.emitGetValue(f, c, "\t\t", u.conv[c.RawName])
		}
		f.p("\t}\n")
	}
	f.p("\treturn nil\n}\n\n")
}

// emitGetValue reads field value.
func (l *LayGen) emitGetValue(f *file, cf *fieldDef, ind, conv string) {
	dst := "v." + cf.Name
	if cf.Vector {
		f.p("%s{\n", ind)
		if cf.BareVector {
			f.p("%s\tn, err := b.Int()\n", ind)
		} else {
			f.p("%s\tn, err := b.VectorHeader()\n", ind)
		}
		f.p("%s\tif err != nil {\n%s\t\treturn err\n%s\t}\n", ind, ind, ind)
		f.p("%s\tif n > 0 {\n%s\t\t%s = make([]%s, 0, n%%bin.PreallocateLimit)\n%s\t}\n",
			ind, ind, dst, goType(cf), ind)
		f.p("%s\tfor i := 0; i < n; i++ {\n", ind)
		l.emitGetElem(f, cf, dst, ind+"\t\t", conv)
		f.p("%s\t}\n%s}\n", ind, ind)
		return
	}
	l.emitGetElem(f, cf, dst, ind, conv)
}

// emitGetElem reads one element into dst.
func (l *LayGen) emitGetElem(f *file, cf *fieldDef, dst, ind, conv string) {
	switch {
	case conv != "":
		f.p("%sx, err := get%s(b)\n", ind, conv)
		f.p("%sif err != nil {\n%s\treturn err\n%s}\n", ind, ind, ind)
		if cf.Vector {
			f.p("%s%s = append(%s, x)\n", ind, dst, dst)
		} else {
			f.p("%s%s = x\n", ind, dst)
		}
	case cf.Encoder && cf.Interface != "":
		f.p("%sx, err := getObj(b, l)\n", ind)
		f.p("%sif err != nil {\n%s\treturn err\n%s}\n", ind, ind, ind)
		f.p("%sy, ok := x.(tg.%s)\n", ind, cf.Interface)
		f.p("%sif !ok {\n%s\treturn ErrMismatch\n%s}\n", ind, ind, ind)
		if cf.Vector {
			f.p("%s%s = append(%s, y)\n", ind, dst, dst)
		} else {
			f.p("%s%s = y\n", ind, dst)
		}
	case cf.Encoder:
		f.p("%sx, err := getObj(b, l)\n", ind)
		f.p("%sif err != nil {\n%s\treturn err\n%s}\n", ind, ind, ind)
		f.p("%sy, ok := x.(*tg.%s)\n", ind, cf.Type)
		f.p("%sif !ok {\n%s\treturn ErrMismatch\n%s}\n", ind, ind, ind)
		if cf.Vector {
			f.p("%s%s = append(%s, *y)\n", ind, dst, dst)
		} else {
			f.p("%s%s = *y\n", ind, dst)
		}
	default:
		f.p("%sx, err := b.%s()\n", ind, cf.Func)
		f.p("%sif err != nil {\n%s\treturn err\n%s}\n", ind, ind, ind)
		if cf.Vector {
			f.p("%s%s = append(%s, x)\n", ind, dst, dst)
		} else {
			f.p("%s%s = x\n", ind, dst)
		}
	}
}

// skipFile emits generic skippers.
func (l *LayGen) skipFile(pkg string) []byte {
	f := newFile(pkg, layImports())
	f.p("// skipObj skips one object.\n")
	f.p("func skipObj(b *bin.Buffer, l Layer) error {\n")
	f.p("\tid, err := b.Uint32()\n")
	f.p("\tif err != nil { return err }\n")
	f.p("\ts, ok := SymOf(l, id)\n")
	f.p("\tif !ok { return skipRaw(b, l, id) }\n")
	f.p("\tif _, err := getAt(b, l, s); err != nil { return err }\n")
	f.p("\treturn nil\n}\n\n")
	f.p("// skipRaw skips layer-only values.\n")
	f.p("func skipRaw(b *bin.Buffer, l Layer, id uint32) error {\n")
	f.p("\tswitch l {\n")
	for _, spec := range l.U.Layers {
		defs := l.skipDefs[spec.Meta.N]
		if len(defs) == 0 {
			continue
		}
		f.p("\tcase L%d:\n\t\tswitch id {\n", spec.Meta.N)
		for _, d := range defs {
			f.p("\t\tcase 0x%08x:\n\t\t\treturn skipQ%d(b, l)\n", d.Def.ID, l.skipIdx[d.Key])
		}
		f.p("\t\t}\n")
	}
	f.raw("\t}\n\treturn fmt.Errorf(\"%w: id %08x\", ErrNoPlan, id)\n}\n\n")
	for _, spec := range l.U.Layers {
		for _, d := range l.skipDefs[spec.Meta.N] {
			idx := l.skipIdx[d.Key]
			if l.skipDone[d.Key] {
				continue
			}
			l.skipDone[d.Key] = true
			l.emitSkipDef(f, idx, d)
		}
	}
	return f.bytes()
}

// emitSkipDef emits one skip body.
func (l *LayGen) emitSkipDef(f *file, idx int, d *layDef) {
	f.p("// skipQ%d\n", idx)
	f.p("func skipQ%d(b *bin.Buffer, l Layer) error {\n", idx)
	names := flagsVars(d)
	for i := range names {
		f.p("\tvar fl%d uint32\n", i)
	}
	for _, p := range d.Def.Params {
		if p.Flags {
			f.p("\t{\n\t\tx, err := b.Uint32()\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\t%s = x\n\t}\n",
				flagVar(names, p.Name))
			continue
		}
		open := false
		if p.Flag != nil {
			f.p("\tif %s&(1<<%d) != 0 {\n", flagVar(names, p.Flag.Name), p.Flag.Index)
			open = true
		}
		l.emitSkipType(f, p.Type, "\t\t")
		if open {
			f.p("\t}\n")
		}
	}
	f.p("\treturn nil\n}\n\n")
}

// emitSkipType skips one typed value.
func (l *LayGen) emitSkipType(f *file, t tl.Type, ind string) {
	if t.GenericArg != nil && (t.Name == "vector" || t.Name == "Vector") {
		if t.Name == "vector" {
			f.p("%sn, err := b.Int()\n", ind)
		} else {
			f.p("%sn, err := b.VectorHeader()\n", ind)
		}
		f.p("%sif err != nil {\n%s\treturn err\n%s}\n", ind, ind, ind)
		f.p("%sfor i := 0; i < n; i++ {\n", ind)
		l.emitSkipType(f, *t.GenericArg, ind+"\t")
		f.p("%s}\n", ind)
		return
	}
	switch {
	case t.GenericRef:
		f.p("%sreturn ErrNoPlan\n", ind)
	case isClassRef(t):
		f.p("%sif err := skipObj(b, l); err != nil {\n%s\treturn err\n%s}\n", ind, ind, ind)
	default:
		name := t.String()
		if t.Percent {
			name = name[1:]
		}
		f.p("%sif err := skipPrim(b, %q); err != nil {\n%s\treturn err\n%s}\n", ind, name, ind, ind)
	}
}

// buildSkips collects layer-only definitions.
func (l *LayGen) buildSkips() {
	l.skipDefs = map[Layer][]*layDef{}
	l.skipIdx = map[string]int{}
	l.skipDone = map[string]bool{}
	for _, spec := range l.U.Layers {
		var defs []*layDef
		for _, key := range spec.Idx.keys {
			if _, ok := l.U.CanIdx.Get(key); ok {
				continue
			}
			d := spec.Idx.byKey[key]
			if d.IsFunc() {
				continue
			}
			defs = append(defs, d)
			if _, ok := l.skipIdx[d.Key]; !ok {
				l.skipIdx[d.Key] = len(l.skipIdx)
			}
		}
		l.skipDefs[spec.Meta.N] = defs
	}
}

// sortedKeys sorts bool map keys.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
