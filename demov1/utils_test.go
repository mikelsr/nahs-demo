package demov1

import (
	"testing"

	log "github.com/ipfs/go-log"
	"github.com/mikelsr/nahs/net"
)

func TestMain(m *testing.M) {
	log.SetAllLoggers(log.LevelWarn)
	log.SetLogLevel(logName, "debug")
	log.SetLogLevel(net.LogName, "error")

	m.Run()
}

// this is already done when loading the module but...
func TestGetProtocol(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error(r)
			t.FailNow()
		}
	}()
	getProtocol(bikeRentalFile)
}
