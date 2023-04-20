package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	var (
		flPkg         = flag.String("pkg", "main", "Name of generated package")
		flOut         = flag.String("o", "-", "output filename; \"-\" for stdout")
		flNoShared    = flag.Bool("no-shared", false, "no \"shared\" code (but depend on it)")
		flNoDepend    = flag.Bool("no-depend", false, "do not depend on \"shared\"")
		flNoResponses = flag.Bool("no-responses", false, "do not generate command responses")
	)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <yaml-file>\n", os.Args[0])
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

	var sources []string
	for _, arg := range flag.Args() {
		sources = append(sources, filepath.Base(arg))
	}

	j := newJenBuilder(*flPkg, sources, *flNoShared, *flNoDepend, *flNoResponses)

	if !*flNoShared {
		j.createShared()
	}

	for _, arg := range flag.Args() {
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading YAML: %v\n", err)
			continue
		}

		cmd := new(Command)

		err = yaml.NewDecoder(f).Decode(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error decoding YAML: %v\n", err)
			continue
		}

		j.walkCommand(cmd.PayloadKeys, cmd.Payload.RequestType)
		if !*flNoResponses {
			j.walkResponse(cmd.ResponseKeys, cmd.Payload.RequestType)
		}

		err = f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error closing YAML: %v\n", err)
			continue
		}
	}
	err = j.file.Render(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error rendering output: %v\n", err)
		os.Exit(2)
	}

}
