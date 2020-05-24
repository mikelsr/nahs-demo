package demo

import (
	"testing"
)

var bikeRentalFile = "bike_rental.bspl"

// this is already done when loading the module but...
func TestGetProtocol(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error(r)
			t.FailNow()
		}
	}()
	GetProtocol(bikeRentalFile)
}
