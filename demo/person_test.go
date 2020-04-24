package demo

import (
	"testing"
)

func TestPerson_RentBike(t *testing.T) {
	person := NewPerson()
	renter := NewRenter()
	person.node.AddContact(renter.node.ID(), bikeRenterService)

	_, err := person.RentBike(zoneA, zoneB)

	if err != nil {
		t.Error(err)
	}
}
