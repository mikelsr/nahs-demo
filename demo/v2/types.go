package v2

import "fmt"

// Coords represents the coordinates of an agent
type Coords struct {
	X, Y float64
}

func (c Coords) String() string {
	return fmt.Sprintf("%f,%f", c.X, c.Y)
}

type bikeQueue []*Bike

func (q *bikeQueue) push(b *Bike) {
	*q = append(*q, b)
}

func (q *bikeQueue) pop() *Bike {
	if len(*q) == 0 {
		return nil
	}
	queue := *q
	b := queue[0]
	*q = queue[1:]
	return b
}

type bikeStorage struct {
	available *bikeQueue
	reserved  map[string]*Bike
}

func newBikeStorage() bikeStorage {
	avalable := make(bikeQueue, 0)
	reserved := make(map[string]*Bike)
	return bikeStorage{available: &avalable, reserved: reserved}
}

func (bs bikeStorage) dock(b *Bike) {
	bs.available.push(b)
}

func (bs bikeStorage) reserveBike() *Bike {
	r := bs.available.pop()
	if r == nil {
		return nil
	}
	bs.reserved[r.ID()] = r
	return r
}

func (bs bikeStorage) releaseBike(bikeID string) {
	delete(bs.reserved, bikeID)
}

func (bs bikeStorage) has(bikeID string) bool {
	if _, found := bs.reserved[bikeID]; found {
		return true
	}
	for _, b := range *bs.available {
		if b.ID() == bikeID {
			return true
		}
	}
	return false
}
