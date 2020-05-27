package v2

import (
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
	"github.com/mikelsr/nahs/net"
)

// Bike is an agent representing a Bike
type Bike struct {
	reasoner *bikeReasoner
	Node     *nahs.Node

	Coords Coords
	free   bool
}

// NewBike is the default constructor for Bike
func NewBike() Bike {
	b := Bike{}
	// the cycle of life
	b.reasoner = newBikeReasoner()
	//p.Node = nahs.NewNode(p.reasoner)
	b.Node = net.LocalNode(b.reasoner)
	b.reasoner.Node = b.Node
	logger.Debugf("\tCreated bike with ID %s (%s)", shortID(b.ID()), b.ID())
	return b
}

// ID of the bike
func (b Bike) ID() string {
	return b.Node.ID().Pretty()
}

type bikeReasoner struct {
	Node *nahs.Node

	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	currentRider   peer.ID
	currentStation string
}

func newBikeReasoner() *bikeReasoner {
	b := bikeReasoner{}
	// initialize maps
	b.openInstances = make(map[string]bspl.Instance)
	b.droppedInstances = make(map[string]bspl.Instance)
	b.consumedServices = make(map[string]bspl.Protocol)
	// rent bike, ride bike, search for a near station
	b.offeredServices = map[string]bspl.Protocol{
		bikeRideProtocol.Key(): bikeRideProtocol,
	}
	return &b
}

// DropInstance cancels an Instance for whatever motive
func (br *bikeReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := br.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	br.droppedInstances[instanceKey] = instance
	delete(br.openInstances, instanceKey)
	return nil
}

// GetInstance returns an Instance given the instance key
func (br *bikeReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	return nil, false
}

// All instances of a Protocol
func (br *bikeReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	return nil
}

// Instantiate a protocol. Check if the assigned role is a role
// the reasoner is willing to play.
func (br *bikeReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	return nil, nil
}

// RegisterInstance registers an Instance created by another Reasoner
func (br *bikeReasoner) RegisterInstance(i bspl.Instance) error {
	if _, found := br.openInstances[i.Key()]; found {
		return fmt.Errorf("Instance '%s' already existed", i.Key())
	}
	// TODO: verify who sends the message and assert the role ID is correct.
	// This should be done in the library, not the demo.
	if len(i.Roles()) < 2 {
		return fmt.Errorf("Missing roles for instance '%s'", i.Key())
	}
	found := false
	for _, proto := range br.offeredServices {
		if proto.Key() == i.Protocol().Key() {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("Protocol '%s' not offered", i.Protocol().Key())
	}
	br.openInstances[i.Key()] = i

	var err error
	switch i.Protocol().Key() {
	case bikeRideProtocol.Key():
		err = br.registerBikeRide(i)
	}
	if err != nil {
		logger.Errorf("[%s] %s", shortID(br.Node.ID()), err)
	}
	return err
}

// UpdateInstance updates an instance with a newer version of itself
// as long as a valid run from one to the other.
func (br *bikeReasoner) UpdateInstance(j bspl.Instance) error {
	i, found := br.openInstances[j.Key()]
	if !found {
		return fmt.Errorf("Instance '%s' not found", j.Key())
	}
	actions, _, err := i.Diff(j)
	if err != nil {
		return err
	}
	switch j.Protocol().Key() {
	case bikeRideProtocol.Key():
		err = br.updateBikeRide(j, actions)
	}
	if err != nil {
		return err
	}
	i.Update(j)
	return nil
}

func (br *bikeReasoner) updateBikeRide(j bspl.Instance, actions []bspl.Action) error {
	if len(actions) != 1 {
		return errors.New("Unexpected actions")
	}
	stationID := j.GetValue("dropStation")
	logger.Debugf("[%s]\tDropped at %s by %s",
		shortID(br.Node.ID()), shortID(stationID), shortID(br.currentRider))

	br.currentStation = stationID
	br.currentRider = peer.ID("")
	return nil
}

func (br *bikeReasoner) registerBikeRide(i bspl.Instance) error {
	var rider string
	for role, actor := range i.Roles() {
		if role == "Rider" {
			rider = actor
		}
	}
	riderID, err := peer.IDB58Decode(rider)
	if err != nil {
		motive := fmt.Sprintf("Invalid or null Rider '%s'", riderID)
		go sendEvent(events.MakeDropEvent(i.Key(), motive), i, br.Node)
		br.DropInstance(i.Key(), motive)
	}
	br.currentRider = riderID
	br.currentStation = ""

	logger.Infof("\t[%s] New rider %s", shortID(br.Node.ID()), shortID(riderID))
	return nil
}
