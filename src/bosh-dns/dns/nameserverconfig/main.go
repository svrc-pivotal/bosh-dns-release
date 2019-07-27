package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bosh-dns/dns/nameserverconfig/monitor"

	"code.cloudfoundry.org/clock"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"

	"github.com/fsnotify/fsnotify"
)

func main() {
	var bindAddress string
	flag.StringVar(&bindAddress, "bindAddress", "", "address that our dns server is binding to")
	flag.Parse()

	if net.ParseIP(bindAddress) == nil {
		log.Fatalf("invalid ip: %s", bindAddress)
	}

	logger := boshlog.NewAsyncWriterLogger(boshlog.LevelDebug, os.Stdout)
	defer logger.FlushTimeout(5 * time.Second)

	shutdown := make(chan struct{})
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)

	fs := boshsys.NewOsFileSystem(logger)
	realClock := clock.NewClock()
	ticker := realClock.NewTicker(3 * time.Second)

	dnsManager := newDNSManager(bindAddress, logger, realClock, fs)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("failed to initialize fsnotify: %v", err)
	}
	defer watcher.Close()

	monitor := monitor.NewMonitor(
		logger,
		dnsManager,
		ticker,
		watcher,
	)
	go monitor.Run(shutdown)

	<-sigterm
	shutdown <- struct{}{}
}
