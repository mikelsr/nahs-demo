package v2

import (
	"time"

	log "github.com/ipfs/go-log"

	demo "github.com/mikelsr/nahs-demo/demo"
)

const (
	bikeRentalFile    = "bike_rental.bspl"
	bikeRequestFile   = "bike_request.bspl"
	bikeRideFile      = "bike_ride.bspl"
	bikeStorageFile   = "bike_storage.bspl"
	bikeTransportFile = "bike_transport.bspl"
	stationSearchFile = "station_search.bspl"

	logName = "nahs-demo/v2"
)

var (
	bikeRequestProtocol   = demo.GetProtocol(bikeRequestFile)
	bikeRentalProtocol    = demo.GetProtocol(bikeRentalFile)
	bikeRideProtocol      = demo.GetProtocol(bikeRideFile)
	bikeStorageProtocol   = demo.GetProtocol(bikeStorageFile)
	bikeTransportProtocol = demo.GetProtocol(bikeTransportFile)
	stationSearchProtocol = demo.GetProtocol(stationSearchFile)

	logger = log.Logger(logName)
	// LocalNodes must be set to True if the used nodes are local nodes
	LocalNodes = false

	timeout = 2 * time.Second
)
