package demo

import (
	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs/net"
)

const (
	bikeRentalFile = "bike_rental.bspl"

	zoneA = "stationA"
	zoneB = "stationB"
	zoneC = "stationC"
)

var (
	bikeRentalProtocol = getProtocol(bikeRentalFile)

	bikeRenterService = net.Service{
		Roles:    []bspl.Role{"Renter"},
		Protocol: bikeRentalProtocol,
	}
)
