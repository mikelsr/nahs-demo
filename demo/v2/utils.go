package v2

import (
	"github.com/mikelsr/bspl"
	"github.com/mikelsr/nahs"
	"github.com/mikelsr/nahs/events"
)

func sendEvent(e events.Event, i bspl.Instance, n *nahs.Node) {
	target := n.OpenInstances[i.Key()]
	logger.Infof("Send event '%s:%s' to node '%s'", e.Type(), e.ID(), n.ID().Pretty())
	n.SendEvent(target, e)
}
