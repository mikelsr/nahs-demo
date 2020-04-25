package demov1

import (
	"testing"

	"github.com/libp2p/go-libp2p-core/peerstore"
)

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
