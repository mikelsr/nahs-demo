package v2

import "fmt"

// Coords represents the coordinates of an agent
type Coords struct {
	X, Y float64
}

func (c Coords) String() string {
	return fmt.Sprintf("%f,%f", c.X, c.Y)
}

type response struct {
	instance string
	value    interface{}
}
