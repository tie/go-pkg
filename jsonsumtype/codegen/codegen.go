// Package codegen implements code generator for encoding and decoding sum types
// represented in JSON, supporting multiple common representation formats.
package codegen

import (
	"io"
)

// Config defines a configuration for code generation template execution.
type Config struct {
	// Header contains settings for the file header.
	Header Header
	// Functions is the list of function to be generated.
	Functions []Function
}

// Function defines configuration a specific function code generation.
type Function struct {
	// Name is a Go identifier that will be used as a function name.
	Name GoIdentifier
	// SumType is a Go type used as a sum type. The generated function
	// receives a pointer to a value of this type.
	SumType GoType
	// TypeParams is a list of type parameters for generic types. It is used
	// in function declaration and allows generating code for sum type and
	// variant types with type parameters.
	TypeParams GoTypeParamList
	// Operation is the operation that the function should perform (i.e.
	// encode or decode).
	Operation Operation
	// Tagging determines how type information is embedded in JSON.
	Tagging Tagging
	// Variants is a list of sum type variants that the function supports.
	Variants []Variant
	// TagField is the name of the JSON object member used for tagging
	// purposes. Used for TaggingInternal and TaggingAdjacent.
	TagField string
	// ContentField is the name of the JSON object member that contains
	// variant payload. Used for TaggingAdjacent.
	ContentField string
}

// Variant represents a specific variant (case) of a sum type.
type Variant struct {
	// Tag is a string used to distinguish different variants of the sum
	// type. Unused for TaggingUntagged.
	Tag string
	// Type is the Go type associated with this variant.
	Type GoType
}

// Generate executes code generation template with the given configuration.
func Generate(w io.Writer, c Config) error {
	return Template().Execute(w, &c)
}
