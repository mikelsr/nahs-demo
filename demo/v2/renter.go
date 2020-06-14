package v2

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mikelsr/bspl"
	imp "github.com/mikelsr/bspl/implementation"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
	"github.com/mikelsr/nahs/net"
)

// Renter of bikes, controls stations
type Renter struct {
	reasoner *renterReasoner
	Node     *nahs.Node
}

// NewRenter is the default constructor for Renter
func NewRenter(stations ...*Station) Renter {
	r := Renter{}
	// the cycle of life
	r.reasoner = newRenterReasoner(stations...)
	//p.Node = nahs.NewNode(p.reasoner)
	r.Node = net.LocalNode(r.reasoner)
	r.reasoner.Node = r.Node

	logger.Debugf("\tCreated renter with ID %s (%s)", shortID(r.ID()), r.Node.ID())
	return r
}

// ID of the renter
func (r Renter) ID() string {
	return r.Node.ID().Pretty()
}

type renterReasoner struct {
	Node *nahs.Node

	offeredServices  map[string]bspl.Protocol
	consumedServices map[string]bspl.Protocol
	openInstances    map[string]bspl.Instance
	droppedInstances map[string]bspl.Instance

	stationSearchRequests map[string]chan string
	transportRequests     map[string]chan string

	stations map[string]*Station
}

func newRenterReasoner(stations ...*Station) *renterReasoner {
	r := &renterReasoner{}
	// initialize maps
	r.openInstances = make(map[string]bspl.Instance)
	r.droppedInstances = make(map[string]bspl.Instance)
	r.consumedServices = map[string]bspl.Protocol{
		bikeTransportProtocol.Key(): bikeTransportProtocol,
	}
	// rent bike, ride bike, search for a near station
	r.offeredServices = map[string]bspl.Protocol{
		bikeRentalProtocol.Key():    bikeRentalProtocol,
		bikeRequestProtocol.Key():   bikeRequestProtocol,
		stationSearchProtocol.Key(): stationSearchProtocol,
	}
	r.stationSearchRequests = make(map[string]chan string)
	r.transportRequests = make(map[string]chan string)
	r.stations = make(map[string]*Station)
	for _, s := range stations {
		r.stations[s.ID()] = s
	}
	return r
}

// DropInstance cancels an Instance for whatever motive
func (rr *renterReasoner) DropInstance(instanceKey string, motive string) error {
	instance, found := rr.openInstances[instanceKey]
	if !found {
		return fmt.Errorf("Instance '%s' not found", instanceKey)
	}
	rr.droppedInstances[instanceKey] = instance
	delete(rr.openInstances, instanceKey)
	return nil
}

// GetInstance returns an Instance given the instance key
func (rr *renterReasoner) GetInstance(instanceKey string) (bspl.Instance, bool) {
	instance, found := rr.openInstances[instanceKey]
	return instance, found
}

// All instances of a Protocol
func (rr *renterReasoner) Instances(p bspl.Protocol) []bspl.Instance {
	instances := make([]bspl.Instance, len(rr.openInstances))
	i := 0
	for _, v := range rr.openInstances {
		instances[i] = v
		i++
	}
	return instances
}

func (rr *renterReasoner) Instantiate(p bspl.Protocol, roles bspl.Roles, ins bspl.Values) (bspl.Instance, error) {
	if _, consumed := rr.consumedServices[p.Key()]; !consumed {
		return nil, fmt.Errorf("Protocol '%s' not supported by this Node", p.Key())
	}
	switch p.Key() {
	case bikeTransportProtocol.Key():
		return rr.instantiateBikeTransport(roles, ins)
	}
	return nil, fmt.Errorf("Unkown protocol '%s'", p.Key())
}

func (rr *renterReasoner) instantiateBikeTransport(roles bspl.Roles, values bspl.Values) (bspl.Instance, error) {
	id := uuid.New().String()
	params := make(map[string]string)
	required := []string{"in bikeNum", "in src", "in dst", "in datetime"}
	for _, r := range required {
		v, found := values[r]
		if !found {
			return nil, fmt.Errorf("Missing parameter: '%s'", r)
		}
		params[r] = v
	}
	params["out ID key"] = id
	i := imp.NewInstance(bikeTransportProtocol, roles)
	i.SetValue("ID", id)
	i.SetValue("dst", params["in dst"])
	i.SetValue("src", params["in src"])
	i.SetValue("datetime", params["in datetime"])
	i.SetValue("bikeNum", params["in bikeNum"])
	rr.openInstances[i.Key()] = i
	return i, nil
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
	found := false
	for _, proto := range rr.offeredServices {
		if proto.Key() == i.Protocol().Key() {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("Protocol '%s' not offered", i.Protocol().Key())
	}
	rr.openInstances[i.Key()] = i

	var err error
	switch i.Protocol().Key() {
	case bikeRentalProtocol.Key():
		err = rr.registerBikeRental(i)
	case bikeRequestProtocol.Key():
		err = rr.registerBikeRequest(i)
	case stationSearchProtocol.Key():
		err = rr.registerStationSearch(i)
	}
	if err != nil {
		logger.Errorf("[%s] %s", shortID(rr.Node.ID()), err)
	}
	return err
}

func (rr *renterReasoner) registerBikeRental(i bspl.Instance) error {
	stationID := i.GetValue("origin")
	if stationID == "" || !rr.hasStation(stationID) {
		errMsg := fmt.Sprintf("Invalid or null origin station ID: '%s'", stationID)
		go sendEvent(events.MakeDropEvent(i.Key(), errMsg), i, rr.Node)
		return errors.New(errMsg)
	}
	i.SetValue("price", fmt.Sprint(rr.calculatePrice()))
	// TODO: check that station is found
	station := rr.stations[stationID]
	bike := station.reasoner.bikes.reserveBike()
	if bike == nil {
		return fmt.Errorf("No available bikes in station '%s'", station.ID())
	}
	i.SetValue("bikeID", bike.ID()) // TODO: select available bike
	go sendEvent(events.MakeUpdateEvent(i), i, rr.Node)
	return nil
}

func (rr *renterReasoner) registerBikeRequest(i bspl.Instance) error {
	bikeNumStr := i.GetValue("bikeNum")
	dtStr := i.GetValue("datetime")
	stationID := i.GetValue("station")
	logger.Debugf("[%s] Received request for %s bikes at station %s and time %s",
		shortID(rr.Node.ID()), bikeNumStr, shortID(stationID), dtStr)

	// check param validity
	bikeNum, err := strconv.ParseInt(bikeNumStr, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid bikeNum: '%s'", bikeNumStr)
	}
	dt, err := time.Parse(time.RFC3339, dtStr)
	if err != nil {
		return fmt.Errorf("Invalid datetime: '%s'", dtStr)
	}
	var station *Station
	for _, s := range rr.stations {
		if s.ID() == stationID {
			station = s
			break
		}
	}
	if station == nil {
		return fmt.Errorf("Station '%s' not found", stationID)
	}
	errc := make(chan error)
	result := make(chan string)

	go rr.requestTransport(int(bikeNum), station, dt, result, errc)

	var rID string

	select {
	case rID = <-result:
		break
	case err = <-errc:
		logger.Errorf("\t[%s] Couldn't request transport to '%s', err: '%s'",
			shortID(rr.Node.ID()), shortID(stationID), err)
		return err
	}
	offerNum := strconv.Itoa(int(bikeNum))
	if rID == "accept" {
		logger.Infof("[%s] Accepting request '%s'", shortID(rr.Node.ID()), i.Key())
		i.SetValue("rID", "accept")
		i.SetValue("offerNum", offerNum)
	} else {
		logger.Infof("[%s] Rejecting request '%s'", shortID(rr.Node.ID()), i.Key())
		i.SetValue("rID", "reject")
		i.SetValue("offerNum", offerNum)
	}
	go sendEvent(events.MakeUpdateEvent(i), i, rr.Node)
	return nil
}

func (rr *renterReasoner) registerStationSearch(i bspl.Instance) error {
	cStr := strings.Split(i.GetValue("coordinates"), ",")
	var errMsg string
	formatErr := "Incorrectly formatted coordinates"
	var x, y float64
	var err error
	if len(cStr) != 2 {
		errMsg = formatErr
	}
	if x, err = strconv.ParseFloat(cStr[0], 64); err != nil {
		errMsg = formatErr
	}
	if y, err = strconv.ParseFloat(cStr[0], 64); err != nil {
		errMsg = formatErr
	}
	if errMsg != "" {
		rr.DropInstance(i.Key(), errMsg)
		go sendEvent(events.MakeDropEvent(i.Key(), errMsg), i, rr.Node)
		return errors.New(errMsg)
	}
	station := rr.nearestStation(Coords{X: x, Y: y})
	i.SetValue("stationID", station.ID())
	go sendEvent(events.MakeUpdateEvent(i), i, rr.Node)
	return nil
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
	switch j.Protocol().Key() {
	case bikeRentalProtocol.Key():
		err = rr.updateBikeRental(j, actions)
	case bikeTransportProtocol.Key():
		err = rr.updateBikeTransport(j, actions)
	}
	if err != nil {
		return err
	}
	i.Update(j)
	return nil
}

func (rr *renterReasoner) updateBikeRental(j bspl.Instance, actions []bspl.Action) error {
	if len(actions) != 2 {
		return errors.New("Unexpected actions")
	}
	if actions[0].Name == "accept" {
		if actions[1].Name != "reject" {
			return errors.New("Unexpected actions")
		}
	} else if actions[0].Name == "reject" {
		if actions[1].Name != "reject" {
			return errors.New("Unexpected actions")
		}
	} else {
		return errors.New("Unexpected actions")
	}
	client := j.Roles()["Client"]
	bikeID := j.GetValue("bikeID")
	rID := j.GetValue("rID")
	logger.Debugf("[%s] Response from %s for bike %s offer: %s", shortID(rr.Node.ID()),
		shortID(client), shortID(bikeID), rID)
	return nil
}

func (rr *renterReasoner) updateBikeTransport(j bspl.Instance, actions []bspl.Action) error {
	rID := j.GetValue("rID")
	result := j.GetValue("result")
	if rID != "" {
		if result != "" {
			// success or failure
		} else {
			// accept/reject
			requestResult := rr.transportRequests[j.Key()]
			requestResult <- rID
		}
	} else {
		return errors.New("Empty rID")
	}
	return nil
}

func (rr renterReasoner) nearestStation(c Coords) *Station {
	if len(rr.stations) == 0 {
		return nil
	}

	minDist := math.MaxFloat64
	var s *Station
	for _, ns := range rr.stations {
		dist := math.Sqrt(math.Pow(ns.Coords().X-c.X, 2) + math.Pow(ns.Coords().Y-c.Y, 2))
		if dist < minDist {
			minDist = dist
			s = ns
		}
	}
	return s
}

func (rr renterReasoner) calculatePrice() float64 {
	possiblePrices := []float64{0.01, 0.02, 0.03}
	rand.Seed(time.Now().Unix())
	// offer a random price to the client
	price := possiblePrices[rand.Intn(len(possiblePrices))]
	return price
}

func (rr renterReasoner) hasStation(stationID string) bool {
	for _, s := range rr.stations {
		if s.ID() == stationID {
			return true
		}
	}
	return false
}

func (rr *renterReasoner) requestTransport(n int, dst *Station, dt time.Time, result chan string, errc chan error) {
	transports := rr.Node.FindContact(bikeTransportProtocol.Key(), "Transport")
	if len(transports) == 0 {
		errc <- errors.New("no transports fond")
		return
	}
	id := transports[0]
	logger.Debugf("[%s] Request bike transport from %s", shortID(rr.Node.ID()), shortID(id))
	// find an station with enough bikes
	var src *Station
	m := 0
	for _, s := range rr.stations {
		if s.ID() == dst.ID() {
			continue
		}
		if s.reasoner.bikes.available.len() > m {
			src = s
			m = s.reasoner.bikes.available.len()
		}
	}
	if src == nil || m < n {
		errc <- errors.New("couldn't find a station to take bikes from")
		return
	}
	t, err := dt.MarshalText()
	if err != nil {
		errc <- err
		return
	}
	roles := bspl.Roles{"Requester": rr.Node.ID().Pretty(), "Transport": id.Pretty()}
	inputs := bspl.Values{
		"in src":      src.ID(),
		"in dst":      dst.ID(),
		"in bikeNum":  strconv.Itoa(n),
		"in datetime": string(t),
	}
	protocol := bikeTransportProtocol
	instance, err := rr.Instantiate(protocol, roles, inputs)
	rr.Node.OpenInstances[instance.Key()] = id
	if err != nil {
		errc <- err
		return
	}

	go sendEvent(events.MakeNewEvent(instance), instance, rr.Node)
	rr.transportRequests[instance.Key()] = result
}
