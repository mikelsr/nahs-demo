package v2

import (
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
)

func sendEvent(e events.Event, i bspl.Instance, n *nahs.Node) {
	target := n.OpenInstances[i.Key()]
	logger.Infof("Send event '%s:%s' to node '%s'", e.Type(), e.ID(), n.ID().Pretty())
	n.SendEvent(target, e)
}

func sendEventWithResults(node *nahs.Node, id peer.ID, event events.Event) (<-chan bool, <-chan error) {
	okChan := make(chan bool)
	errChan := make(chan error)
	go func() {
		defer close(okChan)
		defer close(errChan)
		ok, err := node.SendEvent(id, event)
		okChan <- ok
		errChan <- err

	}()
	return okChan, errChan
}
