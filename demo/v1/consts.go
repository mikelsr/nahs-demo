package v1

import (
	log "github.com/ipfs/go-log"
	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs/net"

	demo "github.com/mikelsr/nahs-demo/demo"
)

const (
	bikeRentalFile = "bike_rental.bspl"

	logName = "nahs-demo/v1"

	zoneA = "stationA"
	zoneB = "stationB"
	zoneC = "stationC"
)

var (
	bikeRentalProtocol = demo.GetProtocol(bikeRentalFile)
	bikeRenterService  = net.Service{
		Roles:    []bspl.Role{"Renter"},
		Protocol: bikeRentalProtocol,
	}
	logger = log.Logger(logName)
)
