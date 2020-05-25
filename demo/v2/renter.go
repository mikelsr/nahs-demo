package v2

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
	"github.com/mikelsr/nahs/net"
)

// Renter of bycicles, controls stations
type Renter struct {
	reasoner *renterReasoner
	Node     *nahs.Node
}

// NewRenter is the default constructor for Renter
func NewRenter(stations ...*Station) Renter {
	r := Renter{}
	// the cycle of life
	r.reasoner = newRenterReasoner(stations...)
	//p.Node = nahs.NewNode(p.reasoner)
	r.Node = net.LocalNode(r.reasoner)
	r.reasoner.Node = r.Node

	logger.Debugf("Created renter with ID: '%s'", r.Node.ID())
	return r
}

type renterReasoner struct {
	Node *nahs.Node

	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	stationSearchRequests map[string]chan string

	stations []*Station
}

func newRenterReasoner(stations ...*Station) *renterReasoner {
	r := &renterReasoner{}
	// initialize maps
	r.openInstances = make(map[string]bspl.Instance)
	r.droppedInstances = make(map[string]bspl.Instance)
	r.consumedServices = make(map[string]bspl.Protocol)
	// rent bike, ride bike, search for a near station
	r.offeredServices = map[string]bspl.Protocol{
		bikeRentalProtocol.Key():    bikeRentalProtocol,
		bikeRequestProtocol.Key():   bikeRideProtocol,
		stationSearchProtocol.Key(): stationSearchProtocol,
	}
	r.stationSearchRequests = make(map[string]chan string)
	r.stations = stations
	return r
}

// DropInstance cancels an Instance for whatever motive
func (rr *renterReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := rr.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	rr.droppedInstances[instanceKey] = instance
	delete(rr.openInstances, instanceKey)
	return nil
}

// GetInstance returns an Instance given the instance key
func (rr *renterReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	instance, found := rr.openInstances[instanceKey]
	return instance, found
}

// All instances of a Protocol
func (rr *renterReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	instances := make([]bspl.Instance, len(rr.openInstances))
	i := 0
	for _, v := range rr.openInstances {
		instances[i] = v
		i++
	}
	return instances
}

func (rr *renterReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	return nil, fmt.Errorf("Protocol '%s' not supported by this Node", p.Key())
}

func (rr *renterReasoner) UpdateInstance(j bspl.Instance) error {

	return nil
}

func (rr *renterReasoner) RegisterInstance(i bspl.Instance) error {
	if _, found := rr.openInstances[i.Key()]; found {
		return fmt.Errorf("Instance '%s' already existed", i.Key())
	}
	// TODO: verify who sends the message and assert the role ID is correct.
	// This should be done in the library, not the demo.
	if len(i.Roles()) < 2 {
		return fmt.Errorf("Missing roles for instance '%s'", i.Key())
	}
	found := false
	for _, proto := range rr.offeredServices {
		if proto.Key() == i.Protocol().Key() {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("Protocol '%s' not offered", i.Protocol().Key())
	}
	rr.openInstances[i.Key()] = i

	switch i.Protocol().Name {
	case stationSearchProtocol.Name:
		return rr.registerStationSearch(i)
	}
	return nil
}

func (rr *renterReasoner) registerStationSearch(i bspl.Instance) error {
	cStr := strings.Split(i.GetValue("coordinates"), ",")
	var errMsg string
	formatErr := "Incorrectly formatted coordinates"
	var x, y float64
	var err error
	if len(cStr) != 2 {
		errMsg = formatErr
	}
	if x, err = strconv.ParseFloat(cStr[0], 64); err != nil {
		errMsg = formatErr
	}
	if y, err = strconv.ParseFloat(cStr[0], 64); err != nil {
		errMsg = formatErr
	}
	if errMsg != "" {
		rr.DropInstance(i.Key(), errMsg)
		sendEvent(events.MakeDropEvent(i.Key(), errMsg), i, rr.Node)
		return errors.New(errMsg)
	}
	station := rr.nearestStation(Coords{X: x, Y: y})
	i.SetValue("stationID", station.ID)
	go sendEvent(events.MakeUpdateEvent(i), i, rr.Node)
	return nil
}

func (rr *renterReasoner) nearestStation(c Coords) *Station {
	if len(rr.stations) == 0 {
		return nil
	}
	s := rr.stations[0]
	minDist := math.Sqrt(math.Pow(s.Coords.X-c.X, 2) + math.Pow(s.Coords.Y-c.Y, 2))
	for _, ns := range rr.stations[1:] {
		dist := math.Sqrt(math.Pow(ns.Coords.X-c.X, 2) + math.Pow(ns.Coords.Y-c.Y, 2))
		if dist < minDist {
			minDist = dist
			s = ns
		}
	}
	return s
}
