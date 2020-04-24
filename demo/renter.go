package demo

import (
	"fmt"

	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/net"
)

// Renter is a NaHS agent that rents bikes to Clients.
type Renter struct {
	node     *nahs.Node
	reasoner *renterReasoner
}

// NewRenter creates a new Renter NaHS agents
func NewRenter() Renter {
	r := Renter{}
	// the cycle of life
	r.reasoner = newRenterReasoner(nil)
	//r.node = nahs.NewNode(r.reasoner)
	r.node = net.LocalNode(r.reasoner)
	r.reasoner.node = r.node
	r.node.AddProtocol(bikeRentalProtocol)
	return r
}

type renterReasoner struct {
	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance
	node             *nahs.Node
}

func newRenterReasoner(n *nahs.Node) *renterReasoner {
	r := new(renterReasoner)
	r.offeredServices = make(map[string]bspl.Protocol)
	r.consumedServices = make(map[string]bspl.Protocol)
	r.openInstances = make(map[string]bspl.Instance)
	r.droppedInstances = make(map[string]bspl.Instance)
	r.node = n

	r.offeredServices[bikeRentalProtocol.Key()] = bikeRentalProtocol

	return r
}

func (rr *renterReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := rr.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	rr.droppedInstances[instanceKey] = instance
	delete(rr.openInstances, instanceKey)
	delete(rr.node.OpenInstances, instanceKey)
	return nil
}

func (rr *renterReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	instance, found := rr.openInstances[instanceKey]
	return instance, found
}

func (rr *renterReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	instances := make([]bspl.Instance, len(rr.openInstances))
	i := 0
	for _, v := range rr.openInstances {
		instances[i] = v
	}
	return instances
}

func (rr *renterReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	return nil, fmt.Errorf("Protocol '%s' not supported by this node", p.Key())
}

func (rr *renterReasoner) NewMessage(i bspl.Instance, a bspl.Action) (bspl.Message, error) {
	return nil, nil
}

func (rr *renterReasoner) RegisterInstance(i bspl.Instance) error {
	return nil
}

func (rr *renterReasoner) RegisterMessage(instanceKey string, m bspl.Message) error {
	return nil
}
