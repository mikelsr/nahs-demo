package v1

import (
	"testing"

	log "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/mikelsr/nahs/net"
)

func TestMain(m *testing.M) {
	log.SetAllLoggers(log.LevelWarn)
	log.SetLogLevel(logName, "debug")
	log.SetLogLevel(net.LogName, "error")

	m.Run()
}

func TestPerson_RentBike(t *testing.T) {
	person := NewPerson(0.02)
	renter := NewRenter()

	person.node.Peerstore().AddAddrs(renter.node.ID(), renter.node.Addrs(), peerstore.PermanentAddrTTL)

	person.node.AddContact(renter.node.ID(), bikeRenterService)

	_, err := person.RentBike(zoneA, zoneB)
	if err != nil {
		t.Error(err)
	}
}
