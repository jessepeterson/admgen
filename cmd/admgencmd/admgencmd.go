package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Key represents the "key" type of the Apple Device Management YAML.
type Key struct {
	Key      string `yaml:"key"`
	Type     string `yaml:"type"`
	Presence string `yaml:"presence,omitempty"`
	SubKeys  []Key  `yaml:"subkeys,omitempty"`
	Content  string `yaml:"content"`

	// used to override the name (and plist key) of the field for a dictionary type
	keyOverride string `yaml:"-"`
}

// Payload represents the "payload" section defined in the Apple
// Device Management YAML.
type Payload struct {
	RequestType string `yaml:"requesttype"`
	Content     string `yaml:"content"`
}

// Command represents an entire MDM command defined in the Apple
// Device Management YAML.
type Command struct {
	Payload      Payload `yaml:"payload"`
	PayloadKeys  []Key   `yaml:"payloadkeys"`
	ResponseKeys []Key   `yaml:"responsekeys"`
}

func main() {
	var flPkg = flag.String("pkg", "main", "Name of generated package")
	var flOut = flag.String("o", "-", "output filename; \"-\" for stdout")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags] <yaml-file>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		// use flag Output() as this error is related to command line input
		fmt.Fprint(flag.CommandLine.Output(), "error: one YAML path required\n")
		flag.Usage()
		os.Exit(1)
	}

	var output io.Writer = os.Stdout
	var err error
	if *flOut != "-" {
		output, err = os.OpenFile(*flOut, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening output file: %v\n", err)
			os.Exit(2)
		}
	}

	f, err := os.Open(flag.Args()[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading YAML: %v\n", err)
		os.Exit(2)
	}

	cmd := new(Command)

	err = yaml.NewDecoder(f).Decode(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decoding YAML: %v\n", err)
		os.Exit(2)
	}

	j := newJenBuilder(*flPkg)

	j.walkCommand(cmd.PayloadKeys, cmd.Payload.RequestType)
	j.walkResponse(cmd.ResponseKeys, cmd.Payload.RequestType)

	err = j.file.Render(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error rendering output: %v\n", err)
		os.Exit(2)
	}
}
