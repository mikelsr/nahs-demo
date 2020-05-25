package v2

// Station that charges bikes
type Station struct {
	ID     string
	Coords Coords
	bikes  []*bike
}

func (s Station) hooks() []func() {
	return []func(){}
}
