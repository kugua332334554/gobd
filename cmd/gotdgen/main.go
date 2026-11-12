// Binary gotdgen generates go source code from TL schema.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotd/tl"

	"github.com/gotd/td/gen"
)

type formattedSource struct {
	Format bool
	Root   string
}

func (t formattedSource) WriteFile(name string, content []byte) error {
	out := content
	if t.Format {
		buf, err := format.Source(content)
		if err != nil {
			return err
		}
		out = buf
	}
	return os.WriteFile(filepath.Join(t.Root, name), out, 0600)
}

func main() {
	schemaPath := flag.String("schema", "", "Path to .tl file")
	targetDir := flag.String("target", "td", "Path to target dir")
	packageName := flag.String("package", "td", "Target package name")
	performFormat := flag.Bool("format", true, "Perform code formatting")
	clean := flag.Bool("clean", false, "Clean generated files before generation")

	layersMeta := flag.String("layers-meta", "", "Path to layers meta.json")
	layersDecide := flag.String("layers-decide", "", "Path to layers decide.json")
	layersPriv := flag.String("layers-priv", "", "Path to layers priv.json")
	layersTarget := flag.String("layers-target", "", "Path to layer package dir")
	layersPackage := flag.String("layers-package", "layerx", "Layer package name")
	layersAudit := flag.String("layers-audit", "", "Path to audit json")
	layersOnly := flag.Bool("layers-only", false, "Skip canonical generation")

	genOpts := gen.GeneratorOptions{}
	genOpts.RegisterFlags(flag.CommandLine)

	flag.Parse()
	if *schemaPath == "" {
		panic("no schema provided")
	}
	f, err := os.Open(*schemaPath)
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()

	schema, err := tl.Parse(f)
	if err != nil {
		panic(err)
	}
	files, err := os.ReadDir(*targetDir)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	if os.IsNotExist(err) {
		if err := os.Mkdir(*targetDir, 0750); err != nil {
			panic(err)
		}
	}
	if *clean {
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if !strings.HasSuffix(name, "_gen.go") {
				continue
			}
			if !strings.HasPrefix(name, "tl_") {
				continue
			}
			if err := os.Remove(filepath.Join(*targetDir, name)); err != nil {
				panic(err)
			}
		}
	}

	start := time.Now()
	fs := formattedSource{
		Root:   *targetDir,
		Format: *performFormat,
	}
	g, err := gen.NewGenerator(schema, genOpts)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	collectInfoTime := time.Since(start)

	start = time.Now()
	if !*layersOnly {
		if err := g.WriteSource(fs, *packageName, gen.Template()); err != nil {
			panic(fmt.Sprintf("%+v", err))
		}
	}
	writeTime := time.Since(start)

	fmt.Printf("Generation %s complete, collect time: %s, write time: %s\n",
		*packageName,
		collectInfoTime,
		writeTime,
	)

	if *layersMeta != "" {
		if *layersTarget == "" {
			panic("no layers target")
		}
		start = time.Now()
		u, err := gen.LoadUni(*layersMeta, *layersDecide, *layersPriv)
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}
		lg := gen.NewLayGen(g, u)
		fs := formattedSource{Root: *layersTarget, Format: *performFormat}
		if err := lg.Write(fs, *layersPackage); err != nil {
			panic(fmt.Sprintf("%+v", err))
		}
		if err := lg.OverBudget(); err != nil {
			panic(fmt.Sprintf("%+v", err))
		}
		if *layersAudit != "" {
			if err := lg.WriteAudit(*layersAudit); err != nil {
				panic(fmt.Sprintf("%+v", err))
			}
		}
		fmt.Printf("Layers %d-%d complete, write time: %s\n",
			u.Layers[0].Meta.N, u.Layers[len(u.Layers)-1].Meta.N, time.Since(start))
	}
}
