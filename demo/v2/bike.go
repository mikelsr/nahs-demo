package v2

type bike struct {
	coords coords
	id     string
	free   bool
}

func (b bike) hooks() []func() {
	return []func(){}
}
