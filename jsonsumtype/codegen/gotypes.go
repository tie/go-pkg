package codegen

import (
	"encoding"
)

var (
	_ encoding.TextUnmarshaler = (*GoIdentifier)(nil)
	_ encoding.TextUnmarshaler = (*GoType)(nil)
	_ encoding.TextUnmarshaler = (*GoTypeParamList)(nil)
)

// GoIdentifier validates Go syntax for identifier.
//
// See https://go.dev/ref/spec#identifier
type GoIdentifier string

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (g *GoIdentifier) UnmarshalText(text []byte) error {
	// TODO: implement
	*g = GoIdentifier(text)
	return nil
}

// GoType validates Go syntax for Type.
//
// See https://go.dev/ref/spec#Type
type GoType string

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (g *GoType) UnmarshalText(text []byte) error {
	// TODO: implement
	*g = GoType(text)
	return nil
}

// GoTypeParamList validates Go syntax for TypeParamList.
//
// See https://go.dev/ref/spec#TypeParamList
type GoTypeParamList string

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (g *GoTypeParamList) UnmarshalText(text []byte) error {
	// TODO: implement
	*g = GoTypeParamList(text)
	return nil
}
