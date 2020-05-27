package v2

import (
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
)

func sendEvent(e events.Event, i bspl.Instance, n *nahs.Node) {
	target := n.OpenInstances[i.Key()]
	logger.Infof("\t[%s] Send event '%s:%s' to node %s (instance key: %s)",
		shortStr(n.ID().Pretty()), e.Type(), shortStr(e.ID()), shortStr(target.Pretty()), i.Key())
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

type agent interface {
	ID() string
}

func shortStr(str string) string {
	return color.New(color.Bold, color.FgGreen).Sprint(strings.ToUpper(str[len(str)-4:]))
}

func shortID(a agent) string {
	return shortStr(a.ID())
}

func waitForContact(n *nahs.Node, id string, result chan string) {
	logger.Debugf("\t[%s] Waiting to discover node %s", shortStr(n.ID().Pretty()), shortStr(id))
	pid, _ := peer.IDB58Decode(id)
	for {
		if LocalNodes {
			n.FindNodes()
		} else {
			// release cpu
			time.Sleep(100)
		}
		p := n.Peerstore().Addrs(pid)
		if len(p) != 0 {
			result <- id
			return
		}
	}
}
