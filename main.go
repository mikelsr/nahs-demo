package main

import (
	"github.com/ipfs/go-log"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/mikelsr/bspl"
	common "github.com/mikelsr/nahs-demo/demo"
	demo "github.com/mikelsr/nahs-demo/demo/v2"
	"github.com/mikelsr/nahs/net"
)

const (
	bikeRentalFile    = "bike_rental.bspl"
	bikeRequestFile   = "bike_request.bspl"
	bikeRideFile      = "bike_ride.bspl"
	bikeStorageFile   = "bike_storage.bspl"
	bikeTransportFile = "bike_transport.bspl"
	stationSearchFile = "station_search.bspl"
)

var (
	bikeRequestProtocol   = common.GetProtocol(bikeRequestFile)
	bikeRentalProtocol    = common.GetProtocol(bikeRentalFile)
	bikeRideProtocol      = common.GetProtocol(bikeRideFile)
	bikeStorageProtocol   = common.GetProtocol(bikeStorageFile)
	bikeTransportProtocol = common.GetProtocol(bikeTransportFile)
	stationSearchProtocol = common.GetProtocol(stationSearchFile)

	bikeRenterService = net.Service{
		Roles:    []bspl.Role{"Renter"},
		Protocol: bikeRentalProtocol,
	}
	stationSearchService = net.Service{
		Roles:    []bspl.Role{"Locator"},
		Protocol: stationSearchProtocol,
	}
)

func main() {
	log.SetAllLoggers(log.LevelInfo)
	log.SetLogLevel("nahs/net", "info")
	log.SetLogLevel("nahs-demo/v2", "debug")

	s1 := &demo.Station{ID: "stationA", Coords: demo.Coords{X: 8, Y: 8}}
	s2 := &demo.Station{ID: "stationB", Coords: demo.Coords{X: 40, Y: 40}}

	renter := demo.NewRenter(s1, s2)
	person := demo.NewPerson()

	person.Node.Peerstore().AddAddrs(renter.Node.ID(), renter.Node.Addrs(), peerstore.PermanentAddrTTL)
	person.Node.AddContact(renter.Node.ID(), bikeRenterService)
	person.Node.AddContact(renter.Node.ID(), stationSearchService)
	person.Travel(demo.Coords{X: 15, Y: 15}, demo.Coords{X: 30, Y: 30})
}
