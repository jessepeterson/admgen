package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dave/jennifer/jen"
)

type Key struct {
	Key     string `yaml:"key"`
	Type    string `yaml:"type"`
	SubKeys []Key  `yaml:"subkeys,omitempty"`
}

type DeclarationPayloadSchema struct {
	DeclarationType string `yaml:"declarationtype"`
}

type DeclarationSchema struct {
	Payload     DeclarationPayloadSchema `yaml:"payload"`
	PayloadKeys []Key                    `yaml:"payloadkeys"`
}

func walkForAssetRefs(list *[][]string, cur []string, keys []Key) {
	for _, key := range keys {
		if strings.HasSuffix(key.Key, "AssetReference") {
			*list = append(*list, append(cur, key.Key))
		} else if key.Type == "<dictionary>" {
			walkForAssetRefs(list, append(cur, key.Key), key.SubKeys)
		}
	}
}

func jenGo(pkgName, name string, refs map[string][][]string, w io.Writer) {
	file := jen.NewFile(pkgName)
	file.PackageComment("Code generated by \"admgenddmrefs\"; DO NOT EDIT.")
	d := jen.Dict{}
	for k, v := range refs {
		paths := []jen.Code{}
		for _, v2 := range v {
			elems := []jen.Code{}
			for _, elem := range v2 {
				elems = append(elems, jen.Lit(elem))
			}
			paths = append(paths, jen.Line().Values(elems...))
		}
		paths = append(paths, jen.Line())
		d[jen.Lit(k)] = jen.Values(paths...)
	}
	file.Comment(name + " is a map of declaration type to payload key paths.")
	file.Comment("These key paths contain the identifiers of dependent declarations.")
	file.Var().Id(name).Op("=").Map(jen.String()).Index().Index().String().Values(d)
	file.Render(w)
}

func walk(dir string) (map[string][][]string, error) {
	refs := make(map[string][][]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		if info.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}
		defer f.Close()

		d := &DeclarationSchema{}
		if err = yaml.NewDecoder(f).Decode(d); err != nil {
			return fmt.Errorf("decoding yaml in %s: %w", path, err)
		}

		if !strings.HasPrefix(d.Payload.DeclarationType, "com.apple.configuration.") {
			return nil
		}

		tlist := [][]string{}
		walkForAssetRefs(&tlist, nil, d.PayloadKeys)

		if len(tlist) > 0 {
			refs[d.Payload.DeclarationType] = tlist
		}

		return nil
	})

	return refs, err
}

func main() {
	var (
		flPkg  = flag.String("pkg", "main", "Name of generated package")
		flName = flag.String("name", "idRefs", "Name of variable")
		flOut  = flag.String("o", "-", "output filename; \"-\" for stdout")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <yaml-dir>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var output io.Writer = os.Stdout
	var err error
	if *flOut != "-" {
		output, err = os.OpenFile(*flOut, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening output file: %v\n", err)
			os.Exit(2)
		}
	}

	if len(flag.Args()) != 1 {
		fmt.Fprintln(os.Stderr, "ERROR: must specify exactly one path to yaml files")
		os.Exit(2)
	}

	refs, err := walk(flag.Args()[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: walking directory: %v\n", err)
		os.Exit(1)
	}

	// explicitly add the one non-configuration asset reference
	refs["com.apple.activation.simple"] = [][]string{{"StandardConfigurations"}}

	jenGo(*flPkg, *flName, refs, output)
}
