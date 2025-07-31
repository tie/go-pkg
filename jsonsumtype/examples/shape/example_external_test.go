package shape

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func Example_externallyTagged() {
	var in Shape = Square{Side: 42}
	data, err := json.Marshal(
		&in,
		json.WithMarshalers(
			json.MarshalToFunc(marshalShapeExternallyTagged),
		),
	)
	if err != nil {
		panic(err)
	}
	if err := (*jsontext.Value)(&data).Canonicalize(); err != nil {
		panic(err)
	}

	var out Shape
	if err := json.Unmarshal(
		data,
		&out,
		json.RejectUnknownMembers(true),
		json.WithUnmarshalers(
			json.UnmarshalFromFunc(unmarshalShapeExternallyTagged),
		),
	); err != nil {
		panic(err)
	}

	fmt.Println(string(data))
	fmt.Printf("%+v\n", out)

	// Output:
	// {"square":{"side":42}}
	// {Side:42}
}
