package gen

import (
	"sort"
)

// resPairs builds method result membership.
func (l *LayGen) resPairs() ([]uint64, []uint32) {
	var out []uint64
	seen := map[uint32]bool{}
	for _, key := range l.U.CanIdx.keys {
		d := l.U.CanIdx.byKey[key]
		if !d.IsFunc() {
			continue
		}
		class := d.Def.Type
		for class.GenericArg != nil && (class.Name == "vector" || class.Name == "Vector") {
			class = *class.GenericArg
		}
		if class.GenericRef || !isClassRef(class) {
			continue
		}
		members := l.U.CanIdx.Members(class.String())
		if len(members) == 0 {
			continue
		}
		m := uint64(d.Def.ID)
		seen[d.Def.ID] = true
		for _, mem := range members {
			out = append(out, m<<32|uint64(mem.Def.ID))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	methods := make([]uint32, 0, len(seen))
	for id := range seen {
		methods = append(methods, id)
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i] < methods[j] })
	return out, methods
}

// resFile emits result checks.
func (l *LayGen) resFile(pkg string) []byte {
	f := newFile(pkg, imports("\"sort\""))
	pairs, methods := l.resPairs()
	f.p("// resPairs lists method results.\n")
	f.p("var resPairs = []uint64{\n")
	for _, p := range pairs {
		f.p("\t0x%016x,\n", p)
	}
	f.p("}\n\n")
	f.p("// resMethods lists checked methods.\n")
	f.p("var resMethods = []uint32{\n")
	for _, m := range methods {
		f.p("\t0x%08x,\n", m)
	}
	f.p("}\n\n")
	f.p("// resAny reports unchecked method.\n")
	f.p("func resAny(m Sym) bool {\n")
	f.p("\tn := sort.Search(len(resMethods), func(i int) bool { return resMethods[i] >= uint32(m) })\n")
	f.p("\treturn !(n < len(resMethods) && resMethods[n] == uint32(m))\n}\n\n")
	f.p("// resOK checks result class.\n")
	f.p("func resOK(m Sym, id uint32) bool {\n")
	f.p("\tv := uint64(m)<<32 | uint64(id)\n")
	f.p("\tn := sort.Search(len(resPairs), func(i int) bool { return resPairs[i] >= v })\n")
	f.p("\treturn n < len(resPairs) && resPairs[n] == v\n}\n")
	return f.bytes()
}
