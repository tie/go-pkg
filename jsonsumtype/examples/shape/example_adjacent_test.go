package shape

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

func Example_adjacentlyTagged() {
	var in Shape = Square{Side: 42}
	data, err := json.Marshal(
		&in,
		json.WithMarshalers(
			json.MarshalToFunc(marshalShapeAdjacentlyTagged),
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
			json.UnmarshalFromFunc(unmarshalShapeAdjacentlyTagged),
		),
	); err != nil {
		panic(err)
	}

	fmt.Println(string(data))
	fmt.Printf("%+v\n", out)

	// Output:
	// {"content":{"side":42},"tag":"square"}
	// {Side:42}
}
