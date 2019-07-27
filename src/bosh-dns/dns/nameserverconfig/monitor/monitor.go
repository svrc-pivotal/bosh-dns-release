package monitor

import (
	"bosh-dns/dns/manager"

	"code.cloudfoundry.org/clock"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	"github.com/fsnotify/fsnotify"
)

type Monitor struct {
	logger     boshlog.Logger
	dnsManager manager.DNSManager
	signal     clock.Ticker
	watcher    *fsnotify.Watcher
}

func NewMonitor(logger boshlog.Logger,
	dnsManager manager.DNSManager,
	signal clock.Ticker,
	watcher *fsnotify.Watcher) Monitor {
	return Monitor{
		logger:     logger,
		dnsManager: dnsManager,
		signal:     signal,
		watcher:    watcher,
	}
}

func (c Monitor) RunOnce() error {
	err := c.dnsManager.SetPrimary()
	if err != nil {
		return bosherr.WrapError(err, "Updating nameserver configs")
	}

	return nil
}

func (c Monitor) Run(shutdown chan struct{}) {
	for {
		select {
		case <-shutdown:
			return
		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				err := c.RunOnce()
				if err != nil {
					c.logger.Error("NameserverConfigMonitor", "running: %s", err)
				}
			}
		case err, ok := <-c.watcher.Errors:
			if !ok {
				return
			}
			c.logger.Error("NameserverConfigMonitor", "fsnotify emitted: %s", err)
		}
	}
}
