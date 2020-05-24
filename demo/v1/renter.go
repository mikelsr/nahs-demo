package v1

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
	"github.com/mikelsr/nahs/net"
)

var prices = []float64{0.01, 0.02, 0.03}

// Renter is a NaHS agent that rents bikes to Clients.
type Renter struct {
	prices   []float64
	node     *nahs.Node
	reasoner *renterReasoner
}

// NewRenter creates a new Renter NaHS agents
func NewRenter() Renter {
	r := Renter{}
	r.prices = prices

	// the cycle of life
	r.reasoner = newRenterReasoner()
	//r.node = nahs.NewNode(r.reasoner)
	r.node = net.LocalNode(r.reasoner)
	r.node.AddProtocol(bikeRentalProtocol)

	go r.dropInstances()
	go r.offerBikes()

	logger.Debugf("Created renter with ID: '%s'", r.node.ID())

	return r
}

func (r Renter) dropInstances() {
	for {
		instanceKey := <-r.reasoner.dropInstanceChan
		logger.Debugf("Drop instance '%s'", instanceKey)
		delete(r.node.OpenInstances, instanceKey)
	}
}

func (r Renter) offerBike(instanceKey string) bool {
	rand.Seed(time.Now().Unix())
	// offer a random price to the client
	price := r.prices[rand.Intn(len(r.prices))]
	instance, found := r.reasoner.GetInstance(instanceKey)
	if !found {
		err := fmt.Errorf("Instance '%s' not found", instanceKey)
		logger.Error(err)
		panic(err)
	}
	instance.SetValue("price", fmt.Sprint(price))
	instance.SetValue("bikeID", "testBike")
	event := events.MakeUpdateEvent(instance)
	target := r.node.OpenInstances[instanceKey]
	logger.Debugf("Offer bike to '%s'", target)
	ok, err := r.node.SendEvent(target, event)
	if err != nil {
		logger.Error(err)
		panic(err)
	}
	return ok
}

func (r Renter) offerBikes() {
	for {
		instanceKey := <-r.reasoner.pendingOffers
		ok := r.offerBike(instanceKey)
		target := r.node.OpenInstances[instanceKey]
		if ok {
			logger.Debugf("Success offering bike to '%s'", target)
		} else {
			logger.Errorf("Failed to offer bike to '%s'", target)
		}
	}
}

type renterReasoner struct {
	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	dropInstanceChan chan string
	pendingOffers    chan string
}

func newRenterReasoner() *renterReasoner {
	r := new(renterReasoner)
	r.offeredServices = make(map[string]bspl.Protocol)
	r.consumedServices = make(map[string]bspl.Protocol)
	r.openInstances = make(map[string]bspl.Instance)
	r.droppedInstances = make(map[string]bspl.Instance)

	r.offeredServices[bikeRentalProtocol.Key()] = bikeRentalProtocol

	r.dropInstanceChan = make(chan string)
	r.pendingOffers = make(chan string)

	return r
}

func (rr *renterReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := rr.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	rr.droppedInstances[instanceKey] = instance
	delete(rr.openInstances, instanceKey)
	rr.dropInstanceChan <- instanceKey
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

func (rr *renterReasoner) UpdateInstance(j bspl.Instance) error {
	i, found := rr.openInstances[j.Key()]
	if !found {
		return fmt.Errorf("Instance '%s' not found", j.Key())
	}
	actions, _, err := i.Diff(j)
	if err != nil {
		return err
	}
	if len(actions) != 2 {
		return errors.New("Unepected actions")
	}
	if actions[0].Name == "accept" {
		if actions[1].Name != "reject" {
			return errors.New("Unepected actions")
		}
	} else if actions[0].Name == "reject" {
		if actions[1].Name != "reject" {
			return errors.New("Unepected actions")
		}
	} else {
		return errors.New("Unepected actions")
	}
	i.Update(j)
	client := i.Roles()["Client"]
	bikeID := i.GetValue("bikeID")
	rID := i.GetValue("rID")
	logger.Infof("Response from '%s' for bike '%s' offer: '%s'", client, bikeID, rID)
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
	rr.openInstances[i.Key()] = i
	if i.Protocol().Key() == bikeRentalProtocol.Key() {
		rr.pendingOffers <- i.Key()
	}
	return nil
}
