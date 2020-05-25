package v2

type bike struct {
	Coords Coords
	id     string
	free   bool
}

func (b bike) hooks() []func() {
	return []func(){}
}
