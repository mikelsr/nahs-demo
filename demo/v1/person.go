package v1

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/mikelsr/bspl"
	imp "github.com/mikelsr/bspl/implementation"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
	"github.com/mikelsr/nahs/net"
)

type offer struct {
	price       float64
	instanceKey string
}

// Person is a NaHS agent representing a single person
type Person struct {
	maxPrice  float64 // maximum €/min a person is willing to pay for a bike
	node      *nahs.Node
	reasoner  *personReasoner
	responses chan bool
}

// NewPerson creates a new Person NaHS agent
func NewPerson(maxPrice float64) Person {
	p := Person{maxPrice: maxPrice}
	// the cycle of life
	p.reasoner = newPersonReasoner()
	//p.node = nahs.NewNode(p.reasoner)
	p.node = net.LocalNode(p.reasoner)
	p.responses = make(chan bool)

	go p.handleOffers()

	logger.Debugf("Created person with ID: '%s'", p.node.ID())
	return p
}

// RentBike looks for a node that plays the Renter role in the
// Bike Rental protocol and requests it
func (p Person) RentBike(origin, destination string) (string, error) {
	protocol := bikeRentalProtocol
	key := protocol.Key()
	renters := p.node.FindContact(key, "Renter")
	if len(renters) == 0 {
		return "", errors.New("No renters found")
	}
	id := renters[0]
	roles := bspl.Roles{"Client": p.node.ID().Pretty(), "Renter": id.Pretty()}
	inputs := bspl.Values{"in origin": origin, "in destination": destination}
	instance, err := p.reasoner.Instantiate(protocol, roles, inputs)
	if err != nil {
		return "", err
	}
	p.node.OpenInstances[instance.Key()] = id
	event := events.MakeNewEvent(instance)
	ok, err := p.node.SendEvent(id, event)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("Instance already existed in renter node")
	}
	ok = <-p.responses
	if ok {
		bikeID := instance.GetValue("bikeID")
		logger.Infof("Rented bike with ID '%s'", bikeID)
		return bikeID, nil
	}
	return "", errors.New("The renter did not reply")
}

func (p Person) handleOffers() {
	for {
		o := <-p.reasoner.offers
		target := p.node.OpenInstances[o.instanceKey]
		instance := p.reasoner.openInstances[o.instanceKey]
		var rID string
		if o.price > p.maxPrice {
			logger.Debugf("Rejected offer from '%s' for price '%.2f'",
				target, o.price)
			rID = "reject"
		} else {
			logger.Debugf("Accepted offer from '%s' for price '%.2f'",
				target, o.price)
			rID = "accept"
		}
		instance.SetValue("rID", rID)
		event := events.MakeUpdateEvent(instance)
		ok, _ := p.node.SendEvent(target, event)
		p.responses <- ok
	}
}

type personReasoner struct {
	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	dropInstanceChan chan string
	offers           chan offer
}

func newPersonReasoner() *personReasoner {
	p := new(personReasoner)

	p.offeredServices = make(map[string]bspl.Protocol)
	p.consumedServices = make(map[string]bspl.Protocol)
	p.openInstances = make(map[string]bspl.Instance)
	p.droppedInstances = make(map[string]bspl.Instance)

	p.dropInstanceChan = make(chan string)
	p.offers = make(chan offer)

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
		i++
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

func (pr *personReasoner) RegisterInstance(i bspl.Instance) error {
	return nil
}

func (pr *personReasoner) UpdateInstance(j bspl.Instance) error {
	i, found := pr.openInstances[j.Key()]
	if !found {
		return fmt.Errorf("Instance not found: '%s'", j.Key())
	}
	actions, _, err := i.Diff(j)
	if err != nil {
		return err
	}
	if len(actions) != 1 && actions[0].Name != "offer" {
		return fmt.Errorf("Invalid update for instance '%s'", i.Key())
	}
	i.Update(j)
	priceStr := i.GetValue("price")
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return err
	}
	logger.Debugf("Received offer for price: '%.2f'", price)
	pr.offers <- offer{instanceKey: i.Key(), price: price}
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
	i := imp.NewInstance(bikeRentalProtocol, roles)
	i.SetValue("ID", id)
	i.SetValue("destination", params["in destination"])
	i.SetValue("origin", params["in origin"])
	pr.openInstances[i.Key()] = i
	return i, nil
}
