package v2

import (
	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
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
	logger.Debugf("\tCreated bike with ID %s (%s)", shortID(b), b.ID())
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
}

func newBikeReasoner() *bikeReasoner {
	b := bikeReasoner{}
	return &b
}

// DropInstance cancels an Instance for whatever motive
func (b *bikeReasoner) DropInstance(instanceKey string, motive string) error {
	return nil
}

// GetInstance returns an Instance given the instance key
func (b *bikeReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	return nil, false
}

// All instances of a Protocol
func (b *bikeReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	return nil
}

// Instantiate a protocol. Check if the assigned role is a role
// the reasoner is willing to play.
func (b *bikeReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	return nil, nil
}

// RegisterInstance registers an Instance created by another Reasoner
func (b *bikeReasoner) RegisterInstance(i bspl.Instance) error {
	return nil
}

// UpdateInstance updates an instance with a newer version of itself
// as long as a valid run from one to the other.
func (b *bikeReasoner) UpdateInstance(newVersion bspl.Instance) error {
	return nil
}
