package codegen

import (
	"encoding"
	"fmt"
)

var _ interface {
	fmt.Stringer
	encoding.TextMarshaler
	encoding.TextUnmarshaler
} = (*JSONPackage)(nil)

// JSONPackage is an enumeration of supported JSON package implementations.
type JSONPackage int

const (
	// JSONPackageStd represents the encoding/json/v2 package from Go’s
	// standard library.
	JSONPackageStd JSONPackage = iota

	// JSONPackageExperiment represents the experimental JSON encoding
	// package from github.com/go-json-experiment/json.
	JSONPackageExperiment
)

// String implements the [fmt.Stringer] interface.
func (p JSONPackage) String() string {
	switch p {
	case JSONPackageStd:
		return "std"
	case JSONPackageExperiment:
		return "experiment"
	}
	return ""
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (p JSONPackage) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (p *JSONPackage) UnmarshalText(text []byte) error {
	switch string(text) {
	case "std":
		*p = JSONPackageStd
	case "experiment":
		*p = JSONPackageExperiment
	default:
		return fmt.Errorf("unknown JSON package %q", text)
	}
	return nil
}
