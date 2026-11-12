package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gotd/td/gen"
)

// jField is one Java field.
type jField struct {
	Java string
	Name string
}

// jClass is one TL class.
type jClass struct {
	Name   string
	ID     uint32
	Extend string
	Fields map[string]jField
	Body   string
	Result string
	Method bool
}

var (
	classRe = regexp.MustCompile(`public\s+(?:static\s+)?(?:abstract\s+)?class\s+([A-Za-z0-9_\.]+)\s*(?:extends\s+([A-Za-z0-9_\.]+))?\s*\{`)
	idRe    = regexp.MustCompile(`constructor\s*=\s*0x([0-9a-fA-F]+)`)
	fieldRe = regexp.MustCompile(`public\s+(static\s+)?([A-Za-z0-9_\.\[\]<>, ]+?)\s+([a-z][A-Za-z0-9_]*)\s*(?:=[^;]*)?;`)
	respRe  = regexp.MustCompile(`return\s+([A-Za-z0-9_\.]+)\.TLdeserialize`)
)

// braceFrom returns the block after start.
func braceFrom(src string, start int) (string, int) {
	open := strings.IndexByte(src[start:], '{')
	if open < 0 {
		return "", len(src)
	}
	depth := 0
	for i := start + open; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start+open+1 : i], i + 1
			}
		}
	}
	return src[start+open+1:], len(src)
}

// splitJava parses classes.
func splitJava(src string) []*jClass {
	var out []*jClass
	for _, m := range classRe.FindAllStringSubmatchIndex(src, -1) {
		name := src[m[2]:m[3]]
		ext := ""
		if m[4] >= 0 {
			ext = src[m[4]:m[5]]
		}
		body, _ := braceFrom(src, m[1]-1)
		c := &jClass{
			Name:   name,
			Extend: ext,
			Body:   body,
			Fields: map[string]jField{},
		}
		if id := idRe.FindStringSubmatch(body); id != nil {
			var v uint64
			fmt.Sscanf(id[1], "%x", &v)
			c.ID = uint32(v)
		}
		if r := respRe.FindStringSubmatch(body); r != nil {
			c.Result = r[1]
		}
		c.Method = c.Result != ""
		for _, f := range fieldRe.FindAllStringSubmatch(body, -1) {
			if f[1] != "" {
				continue
			}
			typ := strings.TrimSpace(f[2])
			if strings.Contains(typ, "(") || typ == "void" || typ == "class" {
				continue
			}
			c.Fields[f[3]] = jField{Java: typ, Name: f[3]}
		}
		out = append(out, c)
	}
	return out
}

// serBody returns serializeToStream body.
func (c *jClass) serBody() string {
	i := strings.Index(c.Body, "serializeToStream(OutputSerializedData stream)")
	if i < 0 {
		return ""
	}
	body, _ := braceFrom(c.Body, i)
	return body
}

// param is one TL parameter.
type param struct {
	Name  string
	Type  string
	Flag  string
	Index int
}

// flagOp is one setFlag call.
type flagOp struct {
	Field string
	Word  string
	Bit   int
	Bool  bool
}

var (
	writeRe = regexp.MustCompile(`stream\.write(Int32|Int64|String|ByteArray|Double|Bool)\(([A-Za-z0-9_\.]+)\)`)
	objRe   = regexp.MustCompile(`([A-Za-z0-9_]+)\.serializeToStream\(stream\)`)
	vecRe   = regexp.MustCompile(`Vector\.serialize(Long|Int|String)?\(stream,\s*([A-Za-z0-9_]+)\)`)
	setFlag = regexp.MustCompile(`(flags2?)\s*=\s*setFlag\((flags2?),\s*FLAG_(\d+),\s*([^;]+)\)`)
	hasFlag = regexp.MustCompile(`hasFlag\((flags2?),\s*FLAG_(\d+)\)`)
	javaVec = regexp.MustCompile(`^ArrayList<(.+)>$`)
	identRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)`)
)

// normName strips dots and underscores.
func normName(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), "_", "")
	return strings.ToLower(s)
}

// javaName converts a class name to TL.
func javaName(class string) string {
	n := strings.TrimPrefix(class, "TL_")
	for _, ns := range namespaces {
		if strings.HasPrefix(n, ns+"_") {
			return ns + "." + n[len(ns)+1:]
		}
	}
	return n
}

// namespaces are known TL namespaces.
var namespaces = []string{
	"account", "auth", "bots", "channels", "chatlists", "contacts", "folders",
	"help", "langpack", "messages", "payments", "photos", "premium", "stars",
	"stats", "stories", "updates", "upload", "users", "wallet",
}

// javaParser resolves type names.
type javaParser struct {
	norm map[string]string
	byID map[string]*jClass
}

// resolve maps a Java type.
func (p *javaParser) resolve(java string) (string, bool) {
	java = strings.TrimSpace(java)
	switch java {
	case "int":
		return "int", true
	case "long":
		return "long", true
	case "String":
		return "string", true
	case "byte[]":
		return "bytes", true
	case "double":
		return "double", true
	case "boolean":
		return "true", true
	case "int128":
		return "int128", true
	case "int256":
		return "int256", true
	}
	if m := javaVec.FindStringSubmatch(java); m != nil {
		inner, ok := p.resolve(m[1])
		if !ok {
			return "", false
		}
		return "Vector<" + inner + ">", true
	}
	key := strings.TrimPrefix(java, "TLRPC.")
	key = strings.TrimPrefix(key, "TL_")
	if tl, ok := p.norm[normName(key)]; ok {
		return tl, true
	}
	return "", false
}

// fieldType resolves a field.
func (p *javaParser) fieldType(c *jClass, name string) (string, bool) {
	for cur := c; cur != nil; {
		if f, ok := cur.Fields[name]; ok {
			return p.resolve(f.Java)
		}
		if cur.Extend == "" || cur.Extend == "TLObject" || cur.Extend == "Object" {
			break
		}
		cur = p.byID[cur.Extend]
	}
	return "", false
}

// buildParams parses fields in wire order.
func (p *javaParser) buildParams(c *jClass) ([]param, error) {
	body := c.serBody()
	if body == "" {
		return nil, fmt.Errorf("%s: no serialize body", c.Name)
	}
	var ops []flagOp
	for _, m := range setFlag.FindAllStringSubmatch(body, -1) {
		var bit int
		fmt.Sscanf(m[3], "%d", &bit)
		expr := strings.TrimSpace(m[4])
		name := identRe.FindString(expr)
		if name == "" {
			continue
		}
		isBool := fieldJava(c, name) == "boolean"
		ops = append(ops, flagOp{Field: name, Word: m[1], Bit: bit, Bool: isBool})
	}
	var (
		out   []param
		cond  []int
		lines = strings.Split(body, "\n")
	)
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "if (") || strings.HasPrefix(line, "if(") {
			if m := hasFlag.FindStringSubmatch(line); m != nil {
				var bit int
				fmt.Sscanf(m[2], "%d", &bit)
				cond = append(cond, bit)
			} else {
				cond = append(cond, -1)
			}
			continue
		}
		if line == "}" {
			if len(cond) > 0 {
				cond = cond[:len(cond)-1]
			}
			continue
		}
		if strings.HasPrefix(line, "flags = setFlag") || strings.HasPrefix(line, "flags2 = setFlag") {
			continue
		}
		cur := -1
		if len(cond) > 0 {
			cur = cond[len(cond)-1]
		}
		if m := writeRe.FindStringSubmatch(line); m != nil {
			arg := m[2]
			if arg == "constructor" {
				continue
			}
			if arg == "flags" || arg == "flags2" {
				out = append(out, param{Name: arg, Type: "#", Index: -1})
				continue
			}
			var t string
			switch m[1] {
			case "Int32":
				t = "int"
			case "Int64":
				t = "long"
			case "String":
				t = "string"
			case "ByteArray":
				t = "bytes"
			case "Double":
				t = "double"
			case "Bool":
				t = "Bool"
			}
			if jt := fieldJava(c, arg); jt == "boolean" {
				if m[1] == "Int32" {
					continue
				}
				t = "true"
			}
			pp := param{Name: arg, Type: t, Index: -1}
			if cur >= 0 {
				pp.Flag, pp.Index = "flags", cur
			}
			out = append(out, pp)
			continue
		}
		if m := vecRe.FindStringSubmatch(line); m != nil {
			arg := m[2]
			t := ""
			switch m[1] {
			case "Long":
				t = "Vector<long>"
			case "Int":
				t = "Vector<int>"
			case "String":
				t = "Vector<string>"
			}
			if t == "" {
				var ok bool
				t, ok = p.fieldType(c, arg)
				if !ok {
					return nil, fmt.Errorf("%s: field %s type", c.Name, arg)
				}
			}
			pp := param{Name: arg, Type: t, Index: -1}
			if cur >= 0 {
				pp.Flag, pp.Index = "flags", cur
			}
			out = append(out, pp)
			continue
		}
		if m := objRe.FindStringSubmatch(line); m != nil {
			arg := m[1]
			t, ok := p.fieldType(c, arg)
			if !ok {
				return nil, fmt.Errorf("%s: field %s type", c.Name, arg)
			}
			pp := param{Name: arg, Type: t, Index: -1}
			if cur >= 0 {
				pp.Flag, pp.Index = "flags", cur
			}
			out = append(out, pp)
		}
	}
	// Attach setFlag conditionals.
	byName := map[string]flagOp{}
	for _, o := range ops {
		byName[o.Field] = o
	}
	for i := range out {
		if o, ok := byName[out[i].Name]; ok && out[i].Index < 0 {
			out[i].Flag, out[i].Index = o.Word, o.Bit
		}
	}
	// Insert bit-only bools after flags marker.
	var extra []param
	for _, o := range ops {
		if !o.Bool {
			continue
		}
		found := false
		for _, q := range out {
			if q.Name == o.Field {
				found = true
				break
			}
		}
		if !found {
			extra = append(extra, param{Name: o.Field, Type: "true", Flag: o.Word, Index: o.Bit})
		}
	}
	if len(extra) > 0 {
		sort.Slice(extra, func(i, j int) bool { return extra[i].Index < extra[j].Index })
		var merged []param
		inserted := false
		for _, q := range out {
			merged = append(merged, q)
			if !inserted && q.Type == "#" {
				merged = append(merged, extra...)
				inserted = true
			}
		}
		if !inserted {
			merged = append(extra, merged...)
		}
		out = merged
	}
	return out, nil
}

// fieldJava returns raw Java type.
func fieldJava(c *jClass, name string) string {
	if f, ok := c.Fields[name]; ok {
		return f.Java
	}
	return ""
}

// tlLine renders one definition.
func tlLine(name string, id uint32, params []param, result string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s#%x", name, id)
	for _, p := range params {
		b.WriteByte(' ')
		if p.Type == "#" {
			b.WriteString(p.Name)
			b.WriteString(":#")
			continue
		}
		b.WriteString(p.Name)
		b.WriteByte(':')
		if p.Flag != "" {
			fmt.Fprintf(&b, "%s.%d?", p.Flag, p.Index)
		}
		b.WriteString(p.Type)
	}
	b.WriteString(" = ")
	b.WriteString(result)
	b.WriteString(";")
	return b.String()
}

// extractTL pulls private definitions from Java sources.
func extractTL(u *gen.Uni, javaDir, srcDir, privPath, out string) error {
	p, err := gen.LoadPriv(privPath)
	if err != nil {
		return err
	}
	ids := map[uint32]bool{}
	for _, it := range p.Items {
		if it.Go == "" && it.ID != "aadf159b" {
			continue
		}
		v, err := strconv.ParseUint(it.ID, 16, 32)
		if err != nil {
			continue
		}
		ids[uint32(v)] = true
	}
	var classes []*jClass
	err = filepath.Walk(javaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		classes = append(classes, splitJava(string(data))...)
		return nil
	})
	if err != nil {
		return err
	}
	parser := &javaParser{norm: map[string]string{}, byID: map[string]*jClass{}}
	put := func(name string) {
		key := normName(name)
		if old, ok := parser.norm[key]; ok && strings.Contains(old, ".") && !strings.Contains(name, ".") {
			return
		}
		parser.norm[key] = name
	}
	for _, c := range classes {
		put(strings.TrimPrefix(c.Name, "TL_"))
		parser.norm[normName(strings.TrimPrefix(c.Name, "TL_"))] = javaName(c.Name)
		parser.byID[c.Name] = c
	}
	data, err := os.ReadFile(filepath.Join(srcDir, "228.json"))
	if err == nil {
		var s jSchema
		if err := json.Unmarshal(data, &s); err == nil {
			for _, d := range s.defs() {
				put(d.name())
				if d.Type != "" {
					put(d.Type)
				}
			}
		}
	}
	for _, key := range u.CanonKeys() {
		put(key)
		if d, ok := u.CanonDef(key); ok {
			put(d.Type.String())
		}
	}
	seen := map[uint32]bool{}
	var sel []*jClass
	for _, c := range classes {
		if !ids[c.ID] || seen[c.ID] || c.serBody() == "" {
			continue
		}
		seen[c.ID] = true
		sel = append(sel, c)
		if !c.Method {
			parser.norm[normName(strings.TrimPrefix(c.Name, "TL_"))] = typeOf(c.Name)
		}
	}
	doc, err := parser.emitTL(sel)
	if err != nil {
		return err
	}
	if os.Getenv("LAYDEBUG") != "" {
		fmt.Printf("debug statsurl=%q\n", parser.norm["statsurl"])
	}
	if err := os.WriteFile(out, []byte(doc), 0600); err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d defs\n", out, len(sel))
	return nil
}

// typeOf derives the result type.
func typeOf(class string) string {
	n := javaName(class)
	if i := strings.LastIndex(n, "."); i >= 0 {
		return n[:i+1] + strings.ToUpper(n[i+1:i+2]) + n[i+2:]
	}
	return strings.ToUpper(n[:1]) + n[1:]
}

// emitTL builds a TL document.
func (p *javaParser) emitTL(classes []*jClass) (string, error) {
	var types, funcs []string
	for _, c := range classes {
		params, err := p.buildParams(c)
		if err != nil {
			return "", err
		}
		name := javaName(c.Name)
		if c.Method {
			result := c.Result
			if r, ok := p.resolve(result); ok {
				result = r
			}
			funcs = append(funcs, tlLine(name, c.ID, params, result))
			continue
		}
		result := c.Result
		if result == "" {
			result = typeOf(c.Name)
		} else if r, ok := p.resolve(result); ok {
			result = r
		} else {
			result = javaName(result)
		}
		types = append(types, tlLine(name, c.ID, params, result))
	}
	sort.Strings(types)
	sort.Strings(funcs)
	var b strings.Builder
	b.WriteString("// Pulled from DrKLO 12.9.1 by cmd/layandroid.\n")
	b.WriteString("// Client-only definitions missing from official layers.\n\n")
	b.WriteString("---functions---\n\n")
	for _, l := range funcs {
		b.WriteString(l)
		b.WriteString("\n\n")
	}
	b.WriteString("---types---\n\n")
	for _, l := range types {
		b.WriteString(l)
		b.WriteString("\n\n")
	}
	return b.String(), nil
}
