package v2

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mikelsr/bspl"
	imp "github.com/mikelsr/bspl/implementation"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
	"github.com/mikelsr/nahs/net"
)

// University is an agent representing human university
type University struct {
	Node          *nahs.Node
	reasoner      *universityReasoner
	nearestStaion *Station
}

// NewUniversity is the default constructor for University
func NewUniversity(nearest *Station) University {
	u := University{}
	// the cycle of life
	u.reasoner = newUniversityReasoner(nearest)
	//u.Node = nahs.NewNode(u.reasoner)
	u.Node = net.LocalNode(u.reasoner)
	u.reasoner.Node = u.Node

	logger.Debugf("\tCreated university with ID %s (%s)", shortID(u.ID()), u.ID())
	return u
}

// ID of the university
func (u University) ID() string {
	return u.Node.ID().Pretty()
}

// RequestBikes requests bikes for nearest station
func (u University) RequestBikes(n int, dt time.Time) error {

	resultc := make(chan int)
	errc := make(chan error)

	go u.reasoner.requestBikes(n, dt, resultc, errc)

	select {
	case result := <-resultc:
		if result > 0 {
			logger.Infof("\t[%s] Success requesting '%d' bikes", shortID(u.ID()), result)
		} else {
			logger.Infof("\t[%s] bike request denied", shortID(u.ID()))
		}
	case err := <-errc:
		logger.Errorf("\t[%s] error requesting bikes: %s", shortID(u.ID()), err)
		return err
	}
	return nil
}

type universityReasoner struct {
	Node *nahs.Node

	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	bikeRequests map[string]chan int

	nearest *Station
}

func newUniversityReasoner(nearest *Station) *universityReasoner {
	u := &universityReasoner{}
	// initialize maps
	u.offeredServices = make(map[string]bspl.Protocol)
	u.openInstances = make(map[string]bspl.Instance)
	u.droppedInstances = make(map[string]bspl.Instance)
	// rent bike, ride bike, search for a near station
	u.consumedServices = map[string]bspl.Protocol{
		bikeRequestProtocol.Key(): bikeRequestProtocol,
	}

	u.bikeRequests = make(map[string]chan int)
	u.nearest = nearest

	return u
}

// DropInstance cancels an Instance for whatever motive
func (ur *universityReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := ur.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	ur.droppedInstances[instanceKey] = instance
	delete(ur.openInstances, instanceKey)
	return nil
}

// GetInstance returns an Instance given the instance key
func (ur *universityReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	instance, found := ur.openInstances[instanceKey]
	return instance, found
}

// All instances of a Protocol
func (ur *universityReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	instances := make([]bspl.Instance, len(ur.openInstances))
	i := 0
	for _, v := range ur.openInstances {
		instances[i] = v
		i++
	}
	return instances
}

// Instantiate a protocol. Check if the assigned role is a role
// the reasoner is willing to play.
func (ur *universityReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	if _, consumed := ur.consumedServices[p.Key()]; !consumed {
		return nil, fmt.Errorf("Protocol '%s' not supported by this Node", p.Key())
	}
	switch p.Key() {
	case bikeRequestProtocol.Key():
		return ur.instantiateBikeRequest(roles, ins)
	}
	return nil, fmt.Errorf("Unkown protocol '%s'", p.Key())
}

func (ur *universityReasoner) instantiateBikeRequest(roles bspl.Roles, values bspl.Values) (bspl.Instance, error) {
	id := uuid.New().String()
	params := make(map[string]string)
	required := []string{"in bikeNum", "in datetime", "in station"}
	for _, r := range required {
		v, found := values[r]
		if !found {
			return nil, fmt.Errorf("Missing parameter: '%s'", r)
		}
		params[r] = v
	}
	i := imp.NewInstance(bikeRequestProtocol, roles)
	i.SetValue("ID", id)
	i.SetValue("bikeNum", params["in bikeNum"])
	i.SetValue("datetime", params["in datetime"])
	i.SetValue("station", params["in station"])
	ur.openInstances[i.Key()] = i
	return i, nil
}

// RegisterInstance registers an Instance created by another Reasoner
func (ur *universityReasoner) RegisterInstance(i bspl.Instance) error {
	return nil
}

// UpdateInstance updates an instance with a newer version of itself
// as long as a valid run from one to the other.
func (ur *universityReasoner) UpdateInstance(newVersion bspl.Instance) error {
	i, found := ur.openInstances[newVersion.Key()]
	if !found {
		return fmt.Errorf("Instance not found: '%s'", newVersion.Key())
	}
	actions, _, err := i.Diff(newVersion)
	if err != nil {
		return err
	}
	switch i.Protocol().Key() {
	case bikeRequestProtocol.Key():
		err = ur.updateBikeRequest(newVersion, actions)
	}
	if err != nil {
		return err
	}
	i.Update(newVersion)
	return nil
}

func (ur *universityReasoner) updateBikeRequest(j bspl.Instance, actions []bspl.Action) error {
	if len(actions) != 2 {
		return fmt.Errorf("Invalid update for instance '%s'", j.Key())
	}

	result := ur.bikeRequests[j.Key()]

	rID := j.GetValue("rID")
	if rID == "reject" {
		result <- 0
	} else if rID != "accept" {
		return fmt.Errorf("Invalid rID: '%s'", rID)
	}
	offerNumStr := j.GetValue("offerNum")
	offerNum, err := strconv.ParseInt(offerNumStr, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid offerNum: '%s'", offerNumStr)
	}
	result <- int(offerNum)
	return nil
}

func (ur *universityReasoner) requestBikes(n int, dt time.Time, result chan int, errc chan error) string {
	protocol := bikeRequestProtocol
	key := protocol.Key()
	renters := ur.Node.FindContact(key, "Renter")
	if len(renters) == 0 {
		errc <- errors.New("No renters found")
		return ""
	}
	id := renters[0]
	logger.Infof("\t[%s] Requesting %d bike(s) from %s to station %s at %v",
		shortID(ur.Node.ID()), n, shortID(id), shortID(ur.nearest.ID()), dt)
	roles := bspl.Roles{"Requester": ur.Node.ID().Pretty(), "Renter": id.Pretty()}
	t, err := dt.MarshalText()
	if err != nil {
		errc <- err
		return ""
	}
	inputs := bspl.Values{
		"in bikeNum":  strconv.Itoa(n),
		"in datetime": string(t),
		"in station":  ur.nearest.ID(),
	}
	instance, err := ur.Instantiate(protocol, roles, inputs)
	if err != nil {
		errc <- err
		return ""
	}
	ur.Node.OpenInstances[instance.Key()] = id
	event := events.MakeNewEvent(instance)
	// send event without blocking execution
	okChan, errChan := sendEventWithResults(ur.Node, id, event)
	select {
	case err := <-errChan:
		errc <- err
		return ""
	case ok := <-okChan:
		if !ok {
			errc <- fmt.Errorf("Instance already existed in renter node")
		}
	}
	ur.bikeRequests[instance.Key()] = result
	return instance.GetValue("ID")
}
