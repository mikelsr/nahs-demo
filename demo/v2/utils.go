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
		shortID(n.ID()), e.Type(), shortID(e.ID()), shortID(target), i.Key())
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

func shortStr(str string) string {
	return color.New(color.Bold, color.FgGreen).Sprint(strings.ToUpper(str[len(str)-4:]))
}

func shortID(s interface{}) string {
	switch s.(type) {
	case string:
		return shortStr(s.(string))
	case peer.ID:
		return shortStr(s.(peer.ID).Pretty())
	}
	return ""
}

func waitForContact(n *nahs.Node, id string, result chan bool) {
	logger.Debugf("\t[%s] Waiting to discover node %s", shortID(n.ID().Pretty()), shortID(id))
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
			logger.Debugf("\t[%s] Discovered %s", shortID(n.ID().Pretty()), shortID(id))
			result <- true
			return
		}
	}
}
