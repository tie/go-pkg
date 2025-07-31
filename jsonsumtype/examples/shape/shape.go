package shape

type Shape interface {
	isShape()
}

type Circle struct {
	Radius int `json:"radius"`
}

func (Circle) isShape() {}

type Square struct {
	Side int `json:"side"`
}

func (Square) isShape() {}
