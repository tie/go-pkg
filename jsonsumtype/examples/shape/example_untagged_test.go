package shape

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func Example_untagged() {
	var in Shape = Square{Side: 42}
	data, err := json.Marshal(
		&in,
		json.WithMarshalers(
			json.MarshalToFunc(marshalShapeUntagged),
		),
	)
	if err != nil {
		panic(err)
	}
	if err := (*jsontext.Value)(&data).Canonicalize(); err != nil {
		panic(err)
	}

	var out1 Shape
	if err := json.Unmarshal(
		data,
		&out1,
		json.RejectUnknownMembers(true),
		json.WithUnmarshalers(
			json.UnmarshalFromFunc(unmarshalShapeUntagged),
		),
	); err != nil {
		panic(err)
	}

	var out2 Shape
	if err := json.Unmarshal(
		[]byte(`{"radius":42}`),
		&out2,
		json.RejectUnknownMembers(true),
		json.WithUnmarshalers(
			json.UnmarshalFromFunc(unmarshalShapeUntagged),
		),
	); err != nil {
		panic(err)
	}

	fmt.Println(string(data))
	fmt.Printf("%+v\n", out1)
	fmt.Printf("%+v\n", out2)

	// Output:
	// {"side":42}
	// {Side:42}
	// {Radius:42}
}
