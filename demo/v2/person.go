package v2

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	imp "github.com/mikelsr/bspl/implementation"

	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
)

type person struct {
	node   *nahs.Node
	coords coords

	rentals  chan string
	stations chan string

	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance
}

func newPerson(coords coords) person {
	p := person{coords: coords}
	p.rentals = make(chan string)
	p.stations = make(chan string)

	p.offeredServices = make(map[string]bspl.Protocol)
	p.consumedServices = make(map[string]bspl.Protocol)
	p.openInstances = make(map[string]bspl.Instance)
	p.droppedInstances = make(map[string]bspl.Instance)

	return p
}

func (p person) nearbyStations() (string, error) {
	protocol := stationSearchProtocol
	key := protocol.Key()
	locators := p.node.FindContact(key, "Locator")
	if len(locators) == 0 {
		return "", errors.New("No locators found")
	}
	id := locators[0]
	roles := bspl.Roles{"User": p.node.ID().Pretty(), "Locator": id.Pretty()}
	inputs := bspl.Values{"in coordinates": p.coords.String()}
	instance, err := p.Instantiate(protocol, roles, inputs)
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
	stationID := <-p.stations
	logger.Infof("Rented bike with ID '%s'", stationID)
	return stationID, nil
}

func (p person) rentBike(price float64, origin string) (string, error) {
	protocol := bikeRentalProtocol
	key := protocol.Key()
	renters := p.node.FindContact(key, "Renter")
	if len(renters) == 0 {
		return "", errors.New("No renters found")
	}
	id := renters[0]
	roles := bspl.Roles{"Client": p.node.ID().Pretty(), "Renter": id.Pretty()}
	inputs := bspl.Values{"in origin": origin, "in destination": ""}
	instance, err := p.Instantiate(protocol, roles, inputs)
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
	bikeID := <-p.rentals
	logger.Infof("Rented bike with ID '%s'", bikeID)
	return bikeID, nil
}

func (p person) Instantiate(protocol bspl.Protocol, roles bspl.Roles, inputs bspl.Values) (bspl.Instance, error) {
	switch protocol.Name {
	case "StationSearch":
		return p.instantiateStationSearch(roles, inputs)
	case "BikeRental":
		return p.instantiateBikeRental(roles, inputs)
	}
	return nil, fmt.Errorf("Protocol '%s' not supported", protocol.Name)
}

func (p *person) instantiateBikeRental(roles bspl.Roles, values bspl.Values) (bspl.Instance, error) {
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
	p.openInstances[i.Key()] = i
	return i, nil
}

func (p person) instantiateStationSearch(roles bspl.Roles, values bspl.Values) (bspl.Instance, error) {
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
	p.openInstances[i.Key()] = i
	return i, nil
}

func (p person) UpdateInstance(j bspl.Instance) error {
	i, found := p.openInstances[j.Key()]
	if !found {
		return fmt.Errorf("Instance not found: '%s'", j.Key())
	}
	actions, _, err := i.Diff(j)
	if err != nil {
		return err
	}
	switch j.Protocol().Name {
	case "StationSearch":
		err = p.updateStationSearch(i, actions)
	case "BikeRental":
		err = p.updateBikeRental(i, actions)
	}
	if err != nil {
		return err
	}
	i.Update(j)
	return nil
}

func (p person) updateStationSearch(i bspl.Instance, actions []bspl.Action) error {
	if len(actions) != 1 && actions[0].Name != "stationID" {
		return fmt.Errorf("Missing station ID for instance '%s'", i.Key())
	}
	stationID := i.GetValue("stationID")
	p.stations <- stationID
	return nil
}

func (p person) updateBikeRental(i bspl.Instance, actions []bspl.Action) error {
	if len(actions) != 1 && actions[0].Name != "stationID" {
		return fmt.Errorf("Missing station ID for instance '%s'", i.Key())
	}
	bikeID := i.GetValue("bikeID")
	p.rentals <- bikeID
	return nil
}
