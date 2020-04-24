package demo

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/mikelsr/bspl"
	imp "github.com/mikelsr/bspl/implementation"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/net"
)

// Person is a NaHS agent representing a single person
type Person struct {
	node     *nahs.Node
	reasoner *personReasoner
}

// NewPerson creates a new Person NaHS agent
func NewPerson() Person {
	p := Person{}
	// the cycle of life
	p.reasoner = newPersonReasoner(nil)
	//p.node = nahs.NewNode(p.reasoner)
	p.node = net.LocalNode(p.reasoner)
	p.reasoner.node = p.node
	return p
}

// RentBike looks for a node that plays the Renter role in the
// Bike Rental protocol and requests it
func (p Person) RentBike(origin, destination string) (string, error) {
	protocol := bikeRentalProtocol
	key := protocol.Key()
	var id peer.ID
	renterFound := false
	for contact, services := range p.node.Contacts {
		service, found := services[key]
		if !found {
			continue
		}
		isRenter := false
		for _, role := range service.Roles {
			if role == "Renter" {
				isRenter = true
				break
			}
		}
		if isRenter {
			renterFound = true
			id = contact
			break
		}
	}
	if !renterFound {
		return "", errors.New("No renters found")
	}
	roles := bspl.Roles{"Client": p.node.ID().Pretty(), "Renter": id.Pretty()}
	inputs := bspl.Values{"in origin": origin, "in destination": destination}
	_, err := p.reasoner.Instantiate(protocol, roles, inputs)
	if err != nil {
		return "", err
	}
	return "", nil
}

type personReasoner struct {
	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance
	node             *nahs.Node
}

func newPersonReasoner(n *nahs.Node) *personReasoner {
	p := new(personReasoner)
	p.offeredServices = make(map[string]bspl.Protocol)
	p.consumedServices = make(map[string]bspl.Protocol)
	p.openInstances = make(map[string]bspl.Instance)
	p.droppedInstances = make(map[string]bspl.Instance)
	p.node = n

	p.consumedServices[bikeRentalProtocol.Key()] = bikeRentalProtocol

	return p
}

func (pr *personReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := pr.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	pr.droppedInstances[instanceKey] = instance
	delete(pr.openInstances, instanceKey)
	delete(pr.node.OpenInstances, instanceKey)
	return nil
}

func (pr *personReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	instance, found := pr.openInstances[instanceKey]
	return instance, found
}

func (pr *personReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	instances := make([]bspl.Instance, len(pr.openInstances))
	i := 0
	for _, v := range pr.openInstances {
		instances[i] = v
	}
	return instances
}

func (pr *personReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	if _, offered := pr.consumedServices[p.Key()]; !offered {
		return nil, fmt.Errorf("Protocol '%s' not supported by this node", p.Key())
	}
	switch p.Key() {
	case bikeRentalProtocol.Key():
		return pr.instantiateBikeRental(roles, ins)
	}
	return nil, fmt.Errorf("Unkown protocol '%s'", p.Key())
}

func (pr *personReasoner) NewMessage(i bspl.Instance, a bspl.Action) (bspl.Message, error) {
	return nil, nil
}

func (pr *personReasoner) RegisterInstance(i bspl.Instance) error {
	return nil
}

func (pr *personReasoner) RegisterMessage(instanceKey string, m bspl.Message) error {
	return nil
}

func (pr *personReasoner) instantiateBikeRental(roles bspl.Roles, values bspl.Values) (bspl.Instance, error) {
	id := uuid.New().String()
	params := make(map[string]string)
	required := []string{"in origin", "in destination"}
	for _, r := range required {
		v, found := values[r]
		if !found {
			return nil, fmt.Errorf("Missing parameter: '%s'", r)
		}
		params[r] = v
	}
	params["out ID key"] = id
	return imp.NewInstance(bikeRentalProtocol, roles, params), nil
}
