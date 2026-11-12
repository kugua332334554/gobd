// Binary layandroid compares DrKLO schemas with official layers.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gotd/td/gen"
	"github.com/gotd/tl"
)

// jParam is one JSON parameter.
type jParam struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// jDef is one JSON definition.
type jDef struct {
	ID     string   `json:"id"`
	Pred   string   `json:"predicate"`
	Method string   `json:"method"`
	Params []jParam `json:"params"`
	Type   string   `json:"type"`
}

// name returns definition name.
func (d jDef) name() string {
	if d.Method != "" {
		return d.Method
	}
	return d.Pred
}

// jSchema is one layer file.
type jSchema struct {
	Constructors []jDef `json:"constructors"`
	Methods      []jDef `json:"methods"`
}

// defs returns all definitions.
func (s jSchema) defs() []jDef {
	out := make([]jDef, 0, len(s.Constructors)+len(s.Methods))
	out = append(out, s.Constructors...)
	out = append(out, s.Methods...)
	return out
}

// id parses decimal id.
func (d jDef) id() (uint32, error) {
	n, err := strconv.ParseInt(d.ID, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint32(int32(n)), nil
}

// shapeJSON renders wire shape.
func shapeJSON(d jDef) string {
	var b strings.Builder
	b.WriteString("(")
	for i, p := range d.Params {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(p.Name)
		b.WriteString(":")
		b.WriteString(p.Type)
	}
	b.WriteString(")=")
	b.WriteString(d.Type)
	return b.String()
}

// shapeTL renders wire shape.
func shapeTL(d *tl.Definition) string {
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

// item is one overlay entry.
type item struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Sym  string `json:"sym,omitempty"`
}

// layerOut is one layer report.
type layerOut struct {
	N     int    `json:"n"`
	Items []item `json:"items"`
}

// report is the output document.
type report struct {
	Ver    int        `json:"ver"`
	Layers []layerOut `json:"layers"`
}

func main() {
	src := flag.String("src", "", "DrKLO tlscheme dir")
	meta := flag.String("meta", "_schema/layers/meta.json", "layers meta")
	decide := flag.String("decide", "_schema/layers/decide.json", "decisions")
	out := flag.String("out", "", "write priv.json")
	java := flag.String("java", "", "java tgnet source dir")
	tlOut := flag.String("tl", "", "write priv.tl")
	goOut := flag.String("go", "", "write priv_gen.go")
	privFile := flag.String("priv", "_schema/layers/priv.json", "priv.json")
	limit := flag.Int("limit", 12, "list lines")
	flag.Parse()

	if *java != "" && *tlOut != "" {
		u, err := gen.LoadUni(*meta, *decide, *privFile)
		if err != nil {
			panic(err)
		}
		if err := extractTL(u, *java, *src, *privFile, *tlOut, *goOut); err != nil {
			panic(err)
		}
		return
	}

	u, err := gen.LoadUni(*meta, *decide, *privFile)
	if err != nil {
		panic(err)
	}
	doc := report{Ver: 1}
	for _, spec := range u.Layers {
		path := filepath.Join(*src, fmt.Sprintf("%d.json", spec.Meta.N))
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("layer %d: skip: %v\n", spec.Meta.N, err)
			continue
		}
		var s jSchema
		if err := json.Unmarshal(data, &s); err != nil {
			panic(err)
		}
		lo, counts := compare(u, spec.Meta.N, s, *limit)
		doc.Layers = append(doc.Layers, lo)
		fmt.Printf("layer %d: %s\n", spec.Meta.N, counts)
	}
	if *out == "" {
		return
	}
	buf, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, append(buf, '\n'), 0600); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s\n", *out)
}

// compare builds one layer report.
func compare(u *gen.Uni, n gen.Layer, s jSchema, limit int) (layerOut, string) {
	var (
		same, diffID, onlyDrk, onlyOff int
		mapOK, shapeBad, gone          int
		inMap, inShape, inGone         []string
	)
	lo := layerOut{N: int(n)}
	official := map[string]uint32{}
	for _, key := range u.LayerKeys(n) {
		if d, ok := u.LayerDef(n, key); ok {
			official[key] = d.ID
		}
	}
	seen := map[string]bool{}
	for _, d := range s.defs() {
		name := d.name()
		id, err := d.id()
		if err != nil {
			continue
		}
		seen[name] = true
		if oid, ok := official[name]; ok {
			if oid == id {
				same++
			} else {
				diffID++
			}
			continue
		}
		onlyDrk++
		can, ok := u.CanonDef(name)
		if !ok {
			gone++
			lo.Items = append(lo.Items, item{ID: fmt.Sprintf("%08x", id), Name: name, Kind: "gone"})
			inGone = append(inGone, fmt.Sprintf("%s %08x", name, id))
			continue
		}
		if shapeTL(can) != shapeJSON(d) {
			shapeBad++
			lo.Items = append(lo.Items, item{ID: fmt.Sprintf("%08x", id), Name: name, Kind: "shape"})
			inShape = append(inShape, name)
			continue
		}
		mapOK++
		lo.Items = append(lo.Items, item{
			ID:   fmt.Sprintf("%08x", id),
			Name: name,
			Kind: "map",
			Sym:  name,
		})
		inMap = append(inMap, name)
	}
	for key := range official {
		if !seen[key] {
			onlyOff++
		}
	}
	sort.Strings(inGone)
	sort.Strings(inShape)
	sort.Strings(inMap)
	counts := fmt.Sprintf("same=%d diffID=%d onlyDrklo=%d onlyOff=%d map=%d shape=%d gone=%d",
		same, diffID, onlyDrk, onlyOff, mapOK, shapeBad, gone)
	printList("map", inMap, limit)
	printList("shape", inShape, limit)
	printList("gone", inGone, limit)
	return lo, counts
}

// printList prints a capped list.
func printList(title string, items []string, limit int) {
	if len(items) == 0 {
		return
	}
	show := items
	if len(show) > limit {
		show = show[:limit]
	}
	fmt.Printf("  %s: %s\n", title, strings.Join(show, ", "))
	if len(items) > limit {
		fmt.Printf("  %s: ... %d more\n", title, len(items)-limit)
	}
}
