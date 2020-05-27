package v2

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mikelsr/bspl"
	imp "github.com/mikelsr/bspl/implementation"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
	"github.com/mikelsr/nahs/net"
)

// Person is an agent representing human person
type Person struct {
	Node     *nahs.Node
	reasoner *personReasoner
}

// NewPerson is the default constructor for Person
func NewPerson() Person {
	p := Person{}
	// the cycle of life
	p.reasoner = newPersonReasoner()
	//p.Node = nahs.NewNode(p.reasoner)
	p.Node = net.LocalNode(p.reasoner)
	p.reasoner.Node = p.Node

	logger.Debugf("\tCreated person with ID %s (%s)", shortID(p), p.ID())
	return p
}

// ID of the person
func (p Person) ID() string {
	return p.Node.ID().Pretty()
}

// Travel from src to dst
func (p Person) Travel(src Coords, dst Coords) error {

	// find nearest station
	logger.Infof("\t[%s] Search for station", shortID(p))
	result := make(chan string)
	errc := make(chan error)
	defer close(result)
	defer close(errc)
	go p.reasoner.stationSearch(src, result, errc)
	var station string
	select {
	case station = <-result:
		logger.Infof("\t[%s] Nearest station found: %s", shortID(p), shortStr(station))
	case err := <-errc:
		logger.Errorf("\t[%s] Couldn't find station: %s", shortID(p), err)
		return err
	}

	// request bike from station
	rID := p.reasoner.bikeRental(station, "", result, errc)
	var bikeID string
	select {
	case bikeID = <-result:
		if bikeID == "" {
			errMsg := "Bike found but rejected."
			return errors.New(errMsg)
		}
		logger.Infof("\t[%s] Bike with id %s rented", shortID(p), shortStr(bikeID))
	case err := <-errc:
		logger.Errorf("\t[%s] Couldn't rent bike: %s", shortID(p), err)
		return err
	}

	// pick the bike
	p.reasoner.pickBike(bikeID, rID)

	// ride bike

	// find a place to drop the bike
	go p.reasoner.stationSearch(dst, result, errc)
	select {
	case station = <-result:
		logger.Infof("\t[%s] Nearest station found: %s", shortID(p), shortStr(station))
	case err := <-errc:
		logger.Errorf("\t[%s] Couldn't find station: %s", shortID(p), err)
		return err
	}
	logger.Infof("\t[%s] Dropping bike %s at station %s", shortID(p), shortStr(bikeID), shortStr(station))

	// drop the bike

	return nil
}

type personReasoner struct {
	Node *nahs.Node

	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	stationSearches map[string]chan string
	rentalRequests  map[string]chan string

	maxPrice float64
}

func newPersonReasoner() *personReasoner {
	p := &personReasoner{}
	// initialize maps
	p.offeredServices = make(map[string]bspl.Protocol)
	p.openInstances = make(map[string]bspl.Instance)
	p.droppedInstances = make(map[string]bspl.Instance)
	// rent bike, ride bike, search for a near station
	p.consumedServices = map[string]bspl.Protocol{
		bikeRentalProtocol.Key():    bikeRentalProtocol,
		bikeRequestProtocol.Key():   bikeRideProtocol,
		bikeRideProtocol.Key():      bikeRideProtocol,
		stationSearchProtocol.Key(): stationSearchProtocol,
	}

	p.stationSearches = make(map[string]chan string)
	p.rentalRequests = make(map[string]chan string)

	p.maxPrice = 0.2

	return p
}

// DropInstance cancels an Instance for whatever motive
func (pr *personReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := pr.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	pr.droppedInstances[instanceKey] = instance
	delete(pr.openInstances, instanceKey)
	return nil
}

// GetInstance returns an Instance given the instance key
func (pr *personReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	instance, found := pr.openInstances[instanceKey]
	return instance, found
}

// All instances of a Protocol
func (pr *personReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	instances := make([]bspl.Instance, len(pr.openInstances))
	i := 0
	for _, v := range pr.openInstances {
		instances[i] = v
		i++
	}
	return instances
}

// Instantiate a protocol. Check if the assigned role is a role
// the reasoner is willing to play.
func (pr *personReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	if _, offered := pr.consumedServices[p.Key()]; !offered {
		return nil, fmt.Errorf("Protocol '%s' not supported by this Node", p.Key())
	}
	switch p.Key() {
	case bikeRentalProtocol.Key():
		return pr.instantiateBikeRental(roles, ins)
	case bikeRideProtocol.Key():
		return pr.instantiateBikeRide(roles, ins)
	case stationSearchProtocol.Key():
		return pr.instantiateStationSearch(roles, ins)
	}
	return nil, fmt.Errorf("Unkown protocol '%s'", p.Key())
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

func (pr *personReasoner) instantiateBikeRide(roles bspl.Roles, values bspl.Values) (bspl.Instance, error) {
	id := uuid.New().String()
	params := make(map[string]string)
	required := []string{"in rentalID"}
	for _, r := range required {
		v, found := values[r]
		if !found {
			return nil, fmt.Errorf("Missing parameter: '%s'", r)
		}
		params[r] = v
	}
	i := imp.NewInstance(bikeRideProtocol, roles)
	i.SetValue("ID", id)
	i.SetValue("rentalID", params["in rentalID"])
	pr.openInstances[i.Key()] = i
	return i, nil
}

func (pr *personReasoner) instantiateStationSearch(roles bspl.Roles, values bspl.Values) (bspl.Instance, error) {
	id := uuid.New().String()
	params := make(map[string]string)
	required := []string{"in coordinates"}
	for _, r := range required {
		v, found := values[r]
		if !found {
			return nil, fmt.Errorf("Missing parameter: '%s'", r)
		}
		params[r] = v
	}
	params["out ID key"] = id
	i := imp.NewInstance(stationSearchProtocol, roles)
	i.SetValue("ID", id)
	i.SetValue("coordinates", params["in coordinates"])
	pr.openInstances[i.Key()] = i
	return i, nil
}

// RegisterInstance registers an Instance created by another Reasoner
func (pr *personReasoner) RegisterInstance(i bspl.Instance) error {
	return nil
}

// UpdateInstance updates an instance with a newer version of itself
// as long as a valid run from one to the other.
func (pr *personReasoner) UpdateInstance(newVersion bspl.Instance) error {
	i, found := pr.openInstances[newVersion.Key()]
	if !found {
		return fmt.Errorf("Instance not found: '%s'", newVersion.Key())
	}
	actions, _, err := i.Diff(newVersion)
	if err != nil {
		return err
	}
	switch i.Protocol().Key() {
	case bikeRentalProtocol.Key():
		err = pr.updateBikeRental(i, newVersion, actions)
	case stationSearchProtocol.Key():
		err = pr.updateStationSearch(newVersion, actions)
	default:
		err = fmt.Errorf("Unkown protocol in update: %s", i.Protocol().Key())
	}
	if err != nil {
		return err
	}
	i.Update(newVersion)
	return nil
}

func (pr *personReasoner) updateStationSearch(i bspl.Instance, actions []bspl.Action) error {
	if len(actions) != 1 && actions[0].Name != "stationID" {
		return fmt.Errorf("Missing station ID for instance '%s'", i.Key())
	}
	stationID := i.GetValue("stationID")
	pr.stationSearches[i.Key()] <- stationID
	return nil
}

func (pr *personReasoner) updateBikeRental(i, j bspl.Instance, actions []bspl.Action) error {
	if len(actions) != 1 && actions[0].Name != "offer" {
		return fmt.Errorf("Invalid update for instance '%s'", j.Key())
	}
	priceStr := j.GetValue("price")
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		errMsg := "Error parsing price"
		go sendEvent(events.MakeDropEvent(j.Key(), errMsg), j, pr.Node)
		return err
	}
	bikeID := j.GetValue("bikeID")
	if bikeID == "" {
		errMsg := "Missing bike ID"
		go sendEvent(events.MakeDropEvent(j.Key(), errMsg), j, pr.Node)
		return errors.New(errMsg)
	}
	logger.Debugf("[%s] Received offer for bike %s at price: '%.2f'",
		shortStr(pr.Node.ID().Pretty()), shortStr(bikeID), price)
	var rID string
	if price > pr.maxPrice {
		logger.Debugf("[%s] Rejected offer for price '%.2f'",
			shortStr(pr.Node.ID().Pretty()), price)
		rID = "reject"
		pr.rentalRequests[j.Key()] <- ""
	} else {
		logger.Debugf("[%s] Accepted offer for price '%.2f'",
			shortStr(pr.Node.ID().Pretty()), price)
		rID = "accept"
		pr.rentalRequests[j.Key()] <- bikeID
	}
	i.Update(j)
	i.SetValue("rID", rID)
	go sendEvent(events.MakeUpdateEvent(i), i, pr.Node)
	return nil
}

func (pr *personReasoner) bikeRental(origin, destination string, result chan string, errc chan error) string {
	protocol := bikeRentalProtocol
	key := protocol.Key()
	renters := pr.Node.FindContact(key, "Renter")
	if len(renters) == 0 {
		errc <- errors.New("No renters found")
		return ""
	}
	id := renters[0]
	roles := bspl.Roles{"Client": pr.Node.ID().Pretty(), "Renter": id.Pretty()}
	inputs := bspl.Values{"in origin": origin, "in destination": destination}
	instance, err := pr.Instantiate(protocol, roles, inputs)
	if err != nil {
		errc <- err
		return ""
	}
	pr.Node.OpenInstances[instance.Key()] = id
	event := events.MakeNewEvent(instance)
	logger.Infof("[%s] Sent rent request to %s", shortStr(pr.Node.ID().Pretty()), shortStr(id.Pretty()))
	// send event without blocking execution
	okChan, errChan := sendEventWithResults(pr.Node, id, event)
	select {
	case err := <-errChan:
		errc <- err
		return ""
	case ok := <-okChan:
		if !ok {
			errc <- fmt.Errorf("Instance already existed in renter node")
		}
	}
	pr.rentalRequests[instance.Key()] = result
	return instance.GetValue("ID")
}

func (pr *personReasoner) stationSearch(c Coords, result chan string, errc chan error) {
	protocol := stationSearchProtocol
	key := protocol.Key()
	locators := pr.Node.FindContact(key, "Locator")
	if len(locators) == 0 {
		errc <- errors.New("No locators found")
		return
	}
	id := locators[0]
	roles := bspl.Roles{"User": pr.Node.ID().Pretty(), "Locator": id.Pretty()}
	inputs := bspl.Values{"in coordinates": c.String()}
	instance, err := pr.Instantiate(protocol, roles, inputs)
	if err != nil {
		errc <- err
		return
	}
	pr.Node.OpenInstances[instance.Key()] = id
	event := events.MakeNewEvent(instance)
	// send event without blocking execution
	okChan, errChan := sendEventWithResults(pr.Node, id, event)
	select {
	case err := <-errChan:
		errc <- err
		return
	case ok := <-okChan:
		if !ok {
			errc <- fmt.Errorf("Instance already existed in renter node")
		}
	}
	pr.stationSearches[instance.Key()] = result
}

func (pr *personReasoner) pickBike(bikeID, rentalID string) {
	// wait until the bike node is found
	found := make(chan string)
	defer close(found)
	go waitForContact(pr.Node, bikeID, found)
	_ = <-found
	// instantiate, send event
	roles := bspl.Roles{"Rider": pr.Node.ID().Pretty(), "Bike": bikeID}
	inputs := bspl.Values{"in rentalID": rentalID}
	i, _ := pr.Instantiate(bikeRideProtocol, roles, inputs)
	pr.Node.OpenInstances[i.Key()], _ = peer.IDB58Decode(bikeID)
	go sendEvent(events.MakeNewEvent(i), i, pr.Node)
}
