package gen

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
)

// PrivItem is one private definition.
type PrivItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Go   string `json:"go,omitempty"`
	Skip string `json:"skip,omitempty"`
}

// PrivTab is priv.json.
type PrivTab struct {
	Ver   int        `json:"ver"`
	Src   string     `json:"src"`
	Note  string     `json:"note"`
	Items []PrivItem `json:"items"`

	byID  map[uint32]PrivItem
	sym   map[uint32]uint32
	named []PrivItem
}

// LoadPriv reads priv.json.
func LoadPriv(path string) (*PrivTab, error) {
	p := &PrivTab{}
	if path == "" {
		return p, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, errors.Wrap(err, "priv.json")
	}
	p.byID = make(map[uint32]PrivItem, len(p.Items))
	for _, it := range p.Items {
		id, err := strconv.ParseUint(it.ID, 16, 32)
		if err != nil {
			return nil, fmt.Errorf("priv %s: bad id", it.Name)
		}
		p.byID[uint32(id)] = it
		if it.Go != "" {
			p.named = append(p.named, it)
		}
	}
	sort.Slice(p.named, func(i, j int) bool { return p.named[i].Go < p.named[j].Go })
	return p, nil
}

// Name returns item name.
func (p *PrivTab) Name(id uint32) (string, bool) {
	it, ok := p.byID[id]
	return it.Name, ok
}

// GoType returns Go type name.
func (p *PrivTab) GoType(id uint32) (string, bool) {
	it, ok := p.byID[id]
	if !ok || it.Go == "" {
		return "", false
	}
	return it.Go, true
}

// Named returns decodable items.
func (p *PrivTab) Named() []PrivItem { return p.named }

// Assign picks free syms for decodable items.
func (p *PrivTab) Assign(used map[Sym]bool) map[string]Sym {
	if p.sym != nil {
		return p.syms()
	}
	p.sym = map[uint32]uint32{}
	next := uint32(0xffffffff)
	for _, it := range p.named {
		for used[Sym(next)] {
			next--
		}
		id, _ := strconv.ParseUint(it.ID, 16, 32)
		p.sym[uint32(id)] = next
		used[Sym(next)] = true
		next--
	}
	return p.syms()
}

func (p *PrivTab) syms() map[string]Sym {
	out := map[string]Sym{}
	for _, it := range p.named {
		id, _ := strconv.ParseUint(it.ID, 16, 32)
		out[it.Go] = Sym(p.sym[uint32(id)])
	}
	return out
}

// SymOf maps private id to sym.
func (p *PrivTab) SymOf(id uint32) (Sym, bool) {
	if p.sym == nil {
		return 0, false
	}
	s, ok := p.sym[id]
	return Sym(s), ok
}

// SymConst returns sym constant name.
func SymConstPriv(goName string) string {
	return "Sym" + strings.ReplaceAll(goName, "_", "")
}

// privFile emits private tables.
func (l *LayGen) privFile(pkg string) []byte {
	f := newFile(pkg, imports(
		fmt.Sprintf("%q", "github.com/gotd/td/bin"),
	))
	p := l.U.Priv
	used := map[Sym]bool{}
	for _, k := range l.U.CanIdx.keys {
		used[Sym(l.U.CanIdx.byKey[k].Def.ID)] = true
	}
	syms := p.Assign(used)
	f.p("// private syms\n")
	f.p("const (\n")
	for _, it := range p.Named() {
		f.p("\t%s Sym = 0x%08x\n", SymConstPriv(it.Go), uint32(syms[it.Go]))
	}
	f.p(")\n\n")
	f.p("// privByID maps client ids.\n")
	f.p("var privByID = map[uint32]Sym{\n")
	for _, it := range p.Named() {
		f.p("\t0x%s: %s,\n", it.ID, SymConstPriv(it.Go))
	}
	f.p("}\n\n")
	f.p("// privName maps sym to TL name.\n")
	f.p("var privName = map[Sym]string{\n")
	for _, it := range p.Named() {
		f.p("\t%s: %q,\n", SymConstPriv(it.Go), it.Name)
	}
	f.p("}\n\n")
	f.p("// privSkip lists known unsupported ids.\n")
	f.p("var privSkip = map[uint32]string{\n")
	for _, it := range p.Items {
		if it.Skip == "" {
			continue
		}
		f.p("\t0x%s: %q,\n", it.ID, it.Name)
	}
	f.p("}\n\n")
	f.p("// privID maps sym to client id.\n")
	f.p("var privID = map[Sym]uint32{\n")
	for _, it := range p.Named() {
		f.p("\t%s: 0x%s,\n", SymConstPriv(it.Go), it.ID)
	}
	f.p("}\n\n")
	f.p("// PrivSym resolves client id.\n")
	f.p("func PrivSym(id uint32) (Sym, bool) {\n\ts, ok := privByID[id]\n\treturn s, ok\n}\n\n")
	f.p("// PrivName returns TL name.\n")
	f.p("func PrivName(s Sym) string { return privName[s] }\n\n")
	f.p("// IsPriv reports private sym.\n")
	f.p("func IsPriv(s Sym) bool {\n\t_, ok := privName[s]\n\treturn ok\n}\n\n")
	f.p("// PrivSkip reports known unsupported id.\n")
	f.p("func PrivSkip(id uint32) (string, bool) {\n\tn, ok := privSkip[id]\n\treturn n, ok\n}\n\n")
	f.p("// putPriv encodes private request.\n")
	f.p("func putPriv(b *bin.Buffer, l Layer, s Sym, v bin.Object) error {\n")
	f.p("\tswitch s {\n")
	for _, it := range p.Named() {
		f.p("\tcase %s:\n", SymConstPriv(it.Go))
		f.p("\t\tx, ok := v.(*%s)\n", it.Go)
		f.p("\t\tif !ok {\n\t\t\treturn ErrMismatch\n\t\t}\n")
		f.p("\t\tb.PutID(privID[s])\n")
		f.p("\t\treturn x.encodePriv(b, l)\n")
	}
	f.p("\t}\n\treturn ErrNoPlan\n}\n\n")
	f.p("// getPriv decodes private request.\n")
	f.p("func getPriv(b *bin.Buffer, l Layer, s Sym) (bin.Object, error) {\n")
	f.p("\tswitch s {\n")
	for _, it := range p.Named() {
		f.p("\tcase %s:\n", SymConstPriv(it.Go))
		f.p("\t\tv := new(%s)\n", it.Go)
		f.p("\t\tif err := v.decodePriv(b, l); err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
		f.p("\t\treturn v, nil\n")
	}
	f.p("\t}\n\treturn nil, ErrNoPlan\n}\n")
	return f.bytes()
}
