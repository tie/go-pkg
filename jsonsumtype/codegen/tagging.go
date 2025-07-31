package codegen

import (
	"encoding"
	"fmt"
)

var _ interface {
	fmt.Stringer
	encoding.TextMarshaler
	encoding.TextUnmarshaler
} = (*Tagging)(nil)

// Tagging is an enumeration of supported sum type tagging representations.
type Tagging int

// These representations determine how type information is embedded in JSON.
const (
	// TaggingExternal represents the variant as a single-key object, where
	// the key is the variant name and the value is the associated content.
	//
	// Example:
	//
	//   {"Circle": {"radius": 10}}
	//   {"Square": {"side": 5}}
	TaggingExternal Tagging = iota

	// TaggingTuple represents the variant as a two-element array (2-tuple),
	// where the first element is the variant name and the second is the
	// associated content.
	//
	// Example:
	//
	//   ["Circle", {"radius": 10}]
	//   ["Square", {"side": 5}]
	TaggingTuple

	// TaggingInternal represents the variant as an object with a tag field,
	// where the tag field holds the variant name and the remaining fields
	// hold the content.
	//
	// Example:
	//
	//   {"type": "Circle", "radius": 10}
	//   {"type": "Square", "side": 5}
	TaggingInternal

	// TaggingAdjacent represents the variant as an object with separate
	// fields for the variant name and its content.
	//
	// Example:
	//
	//   {"type": "Circle", "content": {"radius": 10}}
	//   {"type": "Square", "content": {"side": 5}}
	TaggingAdjacent

	// TaggingUntagged represents the variant as a raw object without any
	// explicit tag; the variant is inferred from the object’s structure.
	//
	// Example:
	//
	//   {"radius": 10}
	//   {"side": 5}
	TaggingUntagged
)

// String implements the [fmt.Stringer] interface.
func (r Tagging) String() string {
	switch r {
	case TaggingExternal:
		return "external"
	case TaggingTuple:
		return "tuple"
	case TaggingInternal:
		return "internal"
	case TaggingAdjacent:
		return "adjacent"
	case TaggingUntagged:
		return "untagged"
	}
	return ""
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (r Tagging) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (r *Tagging) UnmarshalText(text []byte) error {
	switch string(text) {
	case "external":
		*r = TaggingExternal
	case "tuple":
		*r = TaggingTuple
	case "internal":
		*r = TaggingInternal
	case "adjacent":
		*r = TaggingAdjacent
	case "untagged":
		*r = TaggingUntagged
	default:
		return fmt.Errorf("unknown tagging %q", text)
	}
	return nil
}
