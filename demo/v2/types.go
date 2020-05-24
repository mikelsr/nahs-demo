package v2

import "fmt"

type coords struct {
	x, y float64
}

func (c coords) String() string {
	return fmt.Sprintf("%f,%f", c.x, c.y)
}

type response struct {
	instance string
	value    interface{}
}
