package gen

import (
	"fmt"

	"github.com/gotd/tl"
)

// wrappers lists generic envelopes.
func (l *LayGen) wrappers() []*layDef {
	var out []*layDef
	for _, key := range l.U.CanIdx.keys {
		d := l.U.CanIdx.byKey[key]
		if !d.IsFunc() || len(d.Def.GenericParams) != 1 {
			continue
		}
		if d.Def.Type.Name != d.Def.GenericParams[0] {
			continue
		}
		generic := false
		for _, p := range d.Def.Params {
			if p.Type.GenericRef {
				generic = true
				break
			}
		}
		if generic {
			out = append(out, d)
		}
	}
	return out
}

// wrapFile emits envelope tables.
func (l *LayGen) wrapFile(pkg string) []byte {
	f := newFile(pkg, layImports())
	ws := l.wrappers()
	f.p("// wrapMax bounds envelope depth.\n")
	f.p("const wrapMax = 16\n\n")
	f.p("// wrapOf maps sym to envelope index.\n")
	f.p("func wrapOf(s Sym) int {\n\tswitch s {\n")
	for i, w := range ws {
		f.p("\tcase %s:\n\t\treturn %d\n", l.SymConst(w.Key), i)
	}
	f.p("\t}\n\treturn -1\n}\n\n")
	f.p("// wrapSkip skips envelope prefix.\n")
	f.p("func wrapSkip(i int, b *bin.Buffer, l Layer) (int, bool, error) {\n\tswitch i {\n")
	for i := range ws {
		f.p("\tcase %d:\n\t\treturn wrapZ%d(b, l)\n", i, i)
	}
	f.p("\t}\n\treturn 0, false, ErrNoPlan\n}\n\n")
	f.p("// wrapName returns TL name.\n")
	f.p("func wrapName(i int) string {\n\tswitch i {\n")
	for i := range ws {
		f.p("\tcase %d:\n\t\treturn %q\n", i, ws[i].Key)
	}
	f.p("\t}\n\treturn \"\"\n}\n\n")
	for i, w := range ws {
		l.emitWrap(f, i, w)
	}
	return f.bytes()
}

// emitWrap emits one envelope prefix reader.
func (l *LayGen) emitWrap(f *file, idx int, d *layDef) {
	f.p("// wrapZ%d 解包裹\n", idx)
	f.p("func wrapZ%d(b *bin.Buffer, l Layer) (int, bool, error) {\n", idx)
	f.p("\tvar got int\n\tvar has bool\n")
	names := flagsVars(d)
	for i := range names {
		f.p("\tvar fl%d uint32\n", i)
	}
	for _, p := range d.Def.Params {
		if p.Type.GenericRef {
			continue
		}
		if p.Flags {
			f.p("\t{\n\t\tx, err := b.Uint32()\n\t\tif err != nil {\n\t\t\treturn 0, false, err\n\t\t}\n\t\t%s = x\n\t}\n",
				flagVar(names, p.Name))
			continue
		}
		open := false
		if p.Flag != nil {
			f.p("\tif %s&(1<<%d) != 0 {\n", flagVar(names, p.Flag.Name), p.Flag.Index)
			open = true
		}
		if p.Name == "layer" && p.Type.Name == "int" && p.Flag == nil {
			f.p("\tx, err := b.Int()\n")
			f.p("\tif err != nil {\n\t\treturn 0, false, err\n\t}\n")
			f.p("\tgot = x\n\thas = true\n")
		} else {
			l.emitWrapSkip(f, p.Type, "\t")
		}
		if open {
			f.p("\t}\n")
		}
	}
	f.p("\treturn got, has, nil\n}\n\n")
}

// emitWrapSkip skips one value.
func (l *LayGen) emitWrapSkip(f *file, t tl.Type, ind string) {
	if t.GenericArg != nil && (t.Name == "vector" || t.Name == "Vector") {
		if t.Name == "vector" {
			f.p("%sn, err := b.Int()\n", ind)
		} else {
			f.p("%sn, err := b.VectorHeader()\n", ind)
		}
		f.p("%sif err != nil {\n%s\treturn 0, false, err\n%s}\n", ind, ind, ind)
		f.p("%sfor i := 0; i < n; i++ {\n", ind)
		l.emitWrapSkip(f, *t.GenericArg, ind+"\t")
		f.p("%s}\n", ind)
		return
	}
	if isClassRef(t) {
		f.p("%sif err := skipObj(b, l); err != nil {\n%s\treturn 0, false, err\n%s}\n", ind, ind, ind)
		return
	}
	name := t.String()
	if t.Percent {
		name = name[1:]
	}
	f.p("%sif err := skipPrim(b, %q); err != nil {\n%s\treturn 0, false, err\n%s}\n",
		ind, name, ind, ind)
}

// wrapperNames helps audit.
func wrapperNames(ws []*layDef) []string {
	out := make([]string, 0, len(ws))
	for _, w := range ws {
		out = append(out, w.Key)
	}
	return out
}

var _ = fmt.Sprintf
