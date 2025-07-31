package codegen

import (
	"encoding"
	"fmt"
)

var _ interface {
	fmt.Stringer
	encoding.TextMarshaler
	encoding.TextUnmarshaler
} = (*Operation)(nil)

// Operation is an enumeration of actions that can be performed during data
// processing.
type Operation int

const (
	// OperationDecode represents the process of converting serialized data
	// into structured in-memory values.
	OperationDecode Operation = iota

	// OperationEncode represents the process of converting structured
	// in-memory values into a serialized format.
	OperationEncode
)

// String implements the [fmt.Stringer] interface.
func (o Operation) String() string {
	switch o {
	case OperationDecode:
		return "decode"
	case OperationEncode:
		return "encode"
	}
	return ""
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (o Operation) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (o *Operation) UnmarshalText(text []byte) error {
	switch string(text) {
	case "decode":
		*o = OperationDecode
	case "encode":
		*o = OperationEncode
	default:
		return fmt.Errorf("unknown operation %q", text)
	}
	return nil
}
