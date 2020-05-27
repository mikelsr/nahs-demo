package v2

import (
	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/net"
)

// Station that charges bikes
type Station struct {
	reasoner *stationReasoner
	Node     *nahs.Node
}

// NewStation is the default constructor for Station
func NewStation(c Coords) Station {
	s := Station{}
	// the cycle of life
	s.reasoner = newStationReasoner(c)
	//p.Node = nahs.NewNode(p.reasoner)
	s.Node = net.LocalNode(s.reasoner)
	s.reasoner.Node = s.Node
	logger.Debugf("Created station with ID %s (%s)", shortID(s), s.ID())
	return s
}

// ID of the station
func (s Station) ID() string {
	return s.Node.ID().Pretty()
}

// Coords of the station
func (s Station) Coords() Coords {
	return s.reasoner.coords
}

// Bikes retunrs a list of bikes docked at a station
/*func (s Station) Bikes() []*Bike {
	return s.reasoner.bikes
}
*/

// DockBike docks a bike to a station
func (s Station) DockBike(b *Bike) {
	s.reasoner.dockBike(b)
}

// ReleaseBike removes a bike from a station
/*func (s Station) ReleaseBike(b *Bike) {
	s.reasoner.releaseBike(b)
}*/

type stationReasoner struct {
	Node *nahs.Node

	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	coords Coords
	bikes  bikeStorage
}

func newStationReasoner(c Coords) *stationReasoner {
	s := stationReasoner{}
	s.coords = c
	s.bikes = newBikeStorage()
	return &s
}

// DropInstance cancels an Instance for whatever motive
func (sr *stationReasoner) DropInstance(instanceKey string, motive string) error {
	return nil
}

// GetInstance returns an Instance given the instance key
func (sr *stationReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	return nil, false
}

// All instances of a Protocol
func (sr *stationReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	return nil
}

// Instantiate a protocol. Check if the assigned role is a role
// the reasoner is willing to play.
func (sr *stationReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	return nil, nil
}

// RegisterInstance registers an Instance created by another Reasoner
func (sr *stationReasoner) RegisterInstance(i bspl.Instance) error {
	return nil
}

// UpdateInstance updates an instance with a newer version of itself
// as long as a valid run from one to the other.
func (sr *stationReasoner) UpdateInstance(newVersion bspl.Instance) error {
	return nil
}

func (sr *stationReasoner) dockBike(b *Bike) {
	if !sr.bikes.has(b.ID()) {
		logger.Infof("[%s] bike %s docked", shortStr(sr.Node.ID().Pretty()), shortID(b))
		sr.bikes.dock(b)
	}
}

func (sr *stationReasoner) releaseBike(b *Bike) {
	sr.releaseBike(b)
}
