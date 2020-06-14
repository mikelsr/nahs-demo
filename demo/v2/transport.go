package v2

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mikelsr/bspl"
	imp "github.com/mikelsr/bspl/implementation"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
	"github.com/mikelsr/nahs/net"
)

// Transport of bikes
type Transport struct {
	reasoner *transportReasoner
	Node     *nahs.Node
	coords   Coords
	// the transport already knows about stations
	stations []*Station
}

// NewTransport is the default constructor for Transport
func NewTransport(stations ...*Station) Transport {
	t := Transport{}
	t.stations = stations
	// the cycle of life
	t.reasoner = newTransportReasoner(stations...)
	//p.Node = nahs.NewNode(p.reasoner)
	t.Node = net.LocalNode(t.reasoner)
	t.reasoner.Node = t.Node
	logger.Debugf("\tCreated transport with ID %s (%s)", shortID(t.ID()), t.Node.ID())
	return t
}

// ID of the transport
func (t Transport) ID() string {
	return t.Node.ID().Pretty()
}

type transportReasoner struct {
	Node *nahs.Node

	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	coords   Coords
	stations []*Station
	speed    int
}

func newTransportReasoner(stations ...*Station) *transportReasoner {
	t := &transportReasoner{}
	// initialize maps
	t.openInstances = make(map[string]bspl.Instance)
	t.droppedInstances = make(map[string]bspl.Instance)
	t.consumedServices = map[string]bspl.Protocol{
		bikeRideProtocol.Key(): bikeRideProtocol,
	}
	t.offeredServices = map[string]bspl.Protocol{
		bikeTransportProtocol.Key(): bikeTransportProtocol,
	}
	t.coords = Coords{}
	t.stations = stations
	t.speed = 1
	return t
}

// DropInstance cancels an Instance for whatever motive
func (tr *transportReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := tr.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	tr.droppedInstances[instanceKey] = instance
	delete(tr.openInstances, instanceKey)
	return nil
}

// GetInstance returns an Instance given the instance key
func (tr *transportReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	instance, found := tr.openInstances[instanceKey]
	return instance, found
}

// All instances of a Protocol
func (tr *transportReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	instances := make([]bspl.Instance, len(tr.openInstances))
	i := 0
	for _, v := range tr.openInstances {
		instances[i] = v
		i++
	}
	return instances
}

func (tr *transportReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	if _, consumed := tr.consumedServices[p.Key()]; !consumed {
		return nil, fmt.Errorf("Protocol '%s' not supported by this Node", p.Key())
	}
	switch p.Key() {
	case bikeRideProtocol.Key():
		return tr.instantiateBikeRide(roles, ins)
	}
	return nil, fmt.Errorf("Unkown protocol '%s'", p.Key())
}

func (tr *transportReasoner) instantiateBikeRide(roles bspl.Roles, values bspl.Values) (bspl.Instance, error) {
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
	tr.openInstances[i.Key()] = i
	return i, nil
}

func (tr *transportReasoner) RegisterInstance(i bspl.Instance) error {
	if _, found := tr.openInstances[i.Key()]; found {
		return fmt.Errorf("Instance '%s' already existed", i.Key())
	}
	// TODO: verify who sends the message and assert the role ID is correct.
	// This should be done in the library, not the demo.
	if len(i.Roles()) < 2 {
		return fmt.Errorf("Missing roles for instance '%s'", i.Key())
	}
	found := false
	for _, proto := range tr.offeredServices {
		if proto.Key() == i.Protocol().Key() {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("Protocol '%s' not offered", i.Protocol().Key())
	}
	tr.openInstances[i.Key()] = i

	var err error
	switch i.Protocol().Key() {
	case bikeTransportProtocol.Key():
		err = tr.registerBikeTransport(i)
	}
	if err != nil {
		logger.Errorf("[%s] %s", shortID(tr.Node.ID()), err)
	}
	return err
}

func (tr *transportReasoner) registerBikeTransport(i bspl.Instance) error {
	// check bike number
	bikeNum := i.GetValue("bikeNum")
	n, err := strconv.ParseInt(bikeNum, 10, 64)
	if err != nil || n <= 0 {
		errMsg := fmt.Sprintf("Invalid bike number %s.", bikeNum)
		go sendEvent(events.MakeDropEvent(i.Key(), errMsg), i, tr.Node)
		return errors.New(errMsg)
	}
	// look for stations
	srcStr := i.GetValue("src")
	dstStr := i.GetValue("dst")
	var src, dst *Station
	for _, s := range tr.stations {
		if srcStr == s.ID() {
			src = s
		}
		if dstStr == s.ID() {
			dst = s
		}
	}
	if src == nil || dst == nil {
		errMsg := "One of the stations was not found."
		go sendEvent(events.MakeDropEvent(i.Key(), errMsg), i, tr.Node)
		return errors.New(errMsg)
	}
	// check time
	datetime := i.GetValue("datetime")
	dt, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		errMsg := fmt.Sprintf("Invalid datetime: %s", datetime)
		go sendEvent(events.MakeDropEvent(i.Key(), errMsg), i, tr.Node)
		return errors.New(errMsg)
	}
	tr.openInstances[i.Key()] = i
	// Arbitrary time estimation
	estimatedTime := time.Duration(
		math.Sqrt(math.Pow(src.Coords().X-dst.Coords().X, 2)+math.Pow(src.Coords().Y-dst.Coords().Y, 2))/100,
	) * time.Second
	// Time libraries are always magical
	dt = dt.Add(-estimatedTime)
	// dt now contains the estimated hour the bikes should be picked up to arrive at the requested time
	if dt.Before(time.Now()) {
		errMsg := fmt.Sprintf("[%s] Not enough time to transport destination, at least %f seconds required.",
			shortID(tr.Node.ID()), estimatedTime.Seconds())
		go sendEvent(events.MakeDropEvent(i.Key(), errMsg), i, tr.Node)
		return errors.New(errMsg)
	}
	renter := tr.Node.OpenInstances[i.Key()]
	go tr.scheduleTransport(src, dst, n, time.Until(dt), estimatedTime, renter.Pretty(), i.Key())
	i.SetValue("rID", "accept")
	go sendEvent(events.MakeUpdateEvent(i), i, tr.Node)
	return nil
}

func (tr *transportReasoner) UpdateInstance(j bspl.Instance) error {
	return errors.New("Transport does not expect instance updates")
}

func (tr *transportReasoner) scheduleTransport(src, dst *Station, n int64, waitUntil, estimatedDuration time.Duration, renter, key string) {
	select {
	case <-time.After(waitUntil):
		err := tr.transportBikes(src, dst, n, renter, key, estimatedDuration)
		if err != nil {
			logger.Errorf("[%s] Error running scheduled transport: %s", shortID(tr.Node.ID()), err)
		}
	}
	return
}

func (tr *transportReasoner) transportBikes(src, dst *Station, n int64, renter, key string, estimatedDuration time.Duration) error {
	// check availability of bikes
	available := src.reasoner.bikes.available.len()
	if int64(available) < n {
		return fmt.Errorf("%d bikes were requested but only %d were available", available, n)
	}

	// send ok to requester
	instance := tr.openInstances[key]
	instance.SetValue("rID", "accept")
	go sendEvent(events.MakeUpdateEvent(instance), instance, tr.Node)

	// asume location of transport is the first station
	tr.coords = src.Coords()

	keys := make([]string, n)
	bikes := make([]*Bike, n)
	// pick bikes
	for i := 0; int64(i) < n; i++ {
		b := src.reasoner.bikes.available.pop()
		instance := tr.pickBike(b.ID(), renter)
		keys[i] = instance.Key()
		src.reasoner.releaseBike(b)
		bikes[i] = b
		logger.Debugf("[%s] Picked up bike %s from %s", shortID(tr.Node.ID()), shortID(b.ID()), shortID(src.ID()))
	}

	// move bikes
	logger.Debugf("[%s] Moving from %v to %v", shortID(tr.Node.ID()), src.Coords(), dst.Coords())
	time.Sleep(estimatedDuration)
	logger.Debugf("[%s] Moved from %v to %v", shortID(tr.Node.ID()), src.Coords(), dst.Coords())
	tr.coords = dst.Coords()

	// drop bikes
	for i, b := range bikes {
		dst.reasoner.dockBike(b)
		tr.dropBike(b.ID(), dst.ID(), keys[i])
		logger.Debugf("[%s] Dropped bike %s at %s", shortID(tr.Node.ID()), shortID(b.ID()), shortID(dst.ID()))
	}

	instance.SetValue("result", "success")
	go sendEvent(events.MakeUpdateEvent(instance), instance, tr.Node)
	return nil
}

func (tr *transportReasoner) pickBike(bikeID, rentalID string) bspl.Instance {
	// wait until the bike node is found
	found := make(chan bool)
	defer close(found)
	go waitForContact(tr.Node, bikeID, found)
	_ = <-found
	// instantiate, send event
	roles := bspl.Roles{"Rider": tr.Node.ID().Pretty(), "Bike": bikeID}
	inputs := bspl.Values{"in rentalID": rentalID}
	i, _ := tr.Instantiate(bikeRideProtocol, roles, inputs)
	tr.Node.OpenInstances[i.Key()], _ = peer.IDB58Decode(bikeID)
	go sendEvent(events.MakeNewEvent(i), i, tr.Node)
	return i
}

func (tr *transportReasoner) dropBike(bikeID, stationID string, key string) {
	i, found := tr.openInstances[key]
	if !found {
		logger.Errorf("[%s] Instance with key '%s' not found", shortID(tr.Node.ID()), i.Key())
		return
	}
	i.SetValue("dropStation", stationID)
	go sendEvent((events.MakeUpdateEvent(i)), i, tr.Node)
}
