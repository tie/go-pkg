// The jsonsumtype command is a code generator for sum type representations in
// JSON.
//
// Because JSON does not have a native sum types, data is typically represented
// using specific object structures. This code generator supports the following
// representations:
//
//   - Tagged Tuple:
//     ["circle", {"radius": 10}]
//     ["square", {"side": 5}]
//
//   - Externally Tagged Object:
//     {"circle": {"radius": 10}}
//     {"square": {"side": 5}}
//
//   - Internally Tagged Object:
//     {"tag": "circle", "radius": 10}
//     {"tag": "square", "side": 5}
//
//   - Adjacently Tagged Object:
//     {"tag": "Circle", "content": {"radius": 10}}
//     {"tag": "Square", "content": {"side": 5}}
//
//   - Untagged Value:
//     {"radius": 10}
//     {"side": 5}
//
// Marshal and unmarshal functions are generated for each configured tagging
// representation. In Go, sum types are expected to be represented as interfaces
// with private method to limit the set of types. For example:
//
//	type Shape interface {
//		isShape()
//	}
//
//	type Circle struct {
//		Radius int `json:"radius"`
//	}
//
//	func (Circle) isShape() {}
//
//	type Square struct {
//		Side int `json:"side"`
//	}
//
//	func (Square) isShape() {}
//
// By default, the code is generated for encoding/json/v2 package, but can
// optionally use github.com/go-json-experiment/json for older Go versions.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"

	"go.pact.im/x/jsonsumtype/codegen"
)

func main() {
	var config codegen.Config
	var outputPath string
	flag.StringVar(&outputPath, "o", "", "output path")
	flag.Func("config", "read configuration file", func(p string) error {
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, &config)
	})
	flag.Func("package", "override package name", func(p string) error {
		return config.Header.PackageName.UnmarshalText([]byte(p))
	})
	flag.Func("json", "override JSON package", func(p string) error {
		return config.Header.JSONPackage.UnmarshalText([]byte(p))
	})
	flag.Parse()

	if config.Header.PackageName == "" {
		// Environment variable is set by `go generate`.
		packageName := os.Getenv("GOPACKAGE")
		if err := config.Header.PackageName.UnmarshalText([]byte(packageName)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	var buf bytes.Buffer
	err := codegen.Generate(&buf, config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	source, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if outputPath == "" {
		fmt.Print(string(source))
		return
	}
	if err := os.WriteFile(outputPath, source, 0o666); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
