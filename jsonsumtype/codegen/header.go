package codegen

import (
	"cmp"
	"maps"
	"slices"
)

// PackageImport represents an import statement in Go.
type PackageImport struct {
	// PackageName is an optional alias name used when importing.
	PackageName GoIdentifier
	// ImportPath is the import path of the Go package.
	ImportPath string
}

// Header represents metadata for generating a Go source file.
type Header struct {
	// PackageName is the name of the generated Go package.
	PackageName GoIdentifier
	// JSONPackage specifies the JSON package implementation that should be
	// used.
	JSONPackage JSONPackage
	// ExtraImports contains additional packages to import.
	ExtraImports []PackageImport
}

// Imports returns a deduplicated and sorted list of package imports
// required by the generated file.
func (h *Header) Imports() []PackageImport {
	errorsPackage := PackageImport{ImportPath: "errors"}
	reflectPackage := PackageImport{ImportPath: "reflect"}
	strconvPackage := PackageImport{ImportPath: "strconv"}
	stringsPackage := PackageImport{ImportPath: "strings"}

	jsonPackage := PackageImport{ImportPath: "encoding/json/v2"}
	jsontextPackage := PackageImport{ImportPath: "encoding/json/jsontext"}
	if h.JSONPackage == JSONPackageExperiment {
		jsonPackage = PackageImport{
			ImportPath: "github.com/go-json-experiment/json",
		}
		jsontextPackage = PackageImport{
			ImportPath: "github.com/go-json-experiment/json/jsontext",
		}
	}

	imports := map[PackageImport]struct{}{
		errorsPackage:   {},
		reflectPackage:  {},
		strconvPackage:  {},
		stringsPackage:  {},
		jsonPackage:     {},
		jsontextPackage: {},
	}
	for _, imp := range h.ExtraImports {
		imports[imp] = struct{}{}
	}

	return slices.SortedFunc(maps.Keys(imports), func(a, b PackageImport) int {
		return cmp.Or(
			cmp.Compare(a.ImportPath, b.ImportPath),
			cmp.Compare(a.PackageName, b.PackageName),
		)
	})
}
