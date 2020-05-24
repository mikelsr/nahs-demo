package v2

type station struct {
	id     string
	coords coords
	bikes  []*bike
}

func (s station) hooks() []func() {
	return []func(){}
}
