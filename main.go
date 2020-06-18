package main

import (
	"time"

	"github.com/ipfs/go-log"
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
	bikeTransportService = net.Service{
		Roles:    []bspl.Role{"Transport"},
		Protocol: bikeTransportProtocol,
	}
	bikeRequestService = net.Service{
		Roles:    []bspl.Role{"Renter"},
		Protocol: bikeRequestProtocol,
	}
)

func main() {
	log.SetAllLoggers(log.LevelInfo)
	// log.SetLogLevel("nahs/net", "info")
	log.SetLogLevel("nahs-demo/v2", "debug")

	b1 := demo.NewBike()
	b2 := demo.NewBike()
	b3 := demo.NewBike()
	b4 := demo.NewBike()

	s1 := demo.NewStation(demo.Coords{X: 8, Y: 8})
	s1.DockBike(&b1)
	s1.DockBike(&b2)
	s2 := demo.NewStation(demo.Coords{X: 40, Y: 40})
	s2.DockBike(&b3)
	s2.DockBike(&b4)

	transport := demo.NewTransport(&s1, &s2)
	university := demo.NewUniversity(&s1)

	renter := demo.NewRenter(&s1, &s2)
	person := demo.NewPerson()

	common.IntroduceNodes(
		b1.Node, b2.Node, b3.Node, b4.Node,
		s1.Node, s2.Node,
		person.Node,
		transport.Node,
		university.Node,
		renter.Node,
	)

	person.Node.AddContact(renter.Node.ID(), bikeRenterService)
	person.Node.AddContact(renter.Node.ID(), stationSearchService)
	renter.Node.AddContact(transport.Node.ID(), bikeTransportService)
	university.Node.AddContact(renter.Node.ID(), bikeRequestService)

	person.Travel(demo.Coords{X: 15, Y: 15}, demo.Coords{X: 30, Y: 30})
	university.RequestBikes(2, time.Now().Add(2*time.Second))
	// person.Travel(demo.Coords{X: 30, Y: 30}, demo.Coords{X: 15, Y: 15})
}
