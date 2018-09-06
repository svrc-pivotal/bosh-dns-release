package healthexecutable

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"sync"

	"code.cloudfoundry.org/clock"
	"github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry/bosh-utils/system"
)

type HealthExecutableMonitor struct {
	healthExecutablePaths []string
	cmdRunner             system.CmdRunner
	clock                 clock.Clock
	interval              time.Duration
	shutdown              chan struct{}
	status                bool
	groupStatus           map[string]bool
	mutex                 *sync.Mutex
	logger                logger.Logger
}

func NewHealthExecutableMonitor(
	healthExecutablePaths []string,
	cmdRunner system.CmdRunner,
	clock clock.Clock,
	interval time.Duration,
	shutdown chan struct{},
	logger logger.Logger,
) *HealthExecutableMonitor {
	monitor := &HealthExecutableMonitor{
		healthExecutablePaths: healthExecutablePaths,
		cmdRunner:             cmdRunner,
		clock:                 clock,
		interval:              interval,
		shutdown:              shutdown,
		mutex:                 &sync.Mutex{},
		logger:                logger,
	}

	monitor.runChecks()
	go monitor.run()

	return monitor
}

func (m *HealthExecutableMonitor) Status() (bool, map[string]bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.status, m.groupStatus
}

func (m *HealthExecutableMonitor) run() {
	timer := m.clock.NewTimer(m.interval)
	m.logger.Debug("HealthExecutableMonitor", "starting monitor for [%s] with interval %v", strings.Join(m.healthExecutablePaths, ", "), m.interval)

	for {
		select {
		case <-m.shutdown:
			m.logger.Debug("HealthExecutableMonitor", "stopping")
			timer.Stop()
			return
		case <-timer.C():
			m.runChecks()
			timer.Reset(m.interval)
		}
	}
}

func (m *HealthExecutableMonitor) runChecks() {
	var allSucceeded = true

	jobStatus := map[string]bool{}

	for _, executable := range m.healthExecutablePaths {
		jobSucceeded := true

		_, _, exitStatus, err := m.runExecutable(executable)
		if err != nil {
			jobSucceeded = false
			m.logger.Error("HealthExecutableMonitor", "Error occurred executing '%s': %v", executable, err)
		} else if exitStatus != 0 {
			jobSucceeded = false
		}

		if strings.HasPrefix(executable, "/var/vcap/jobs") {
			// TODO(db,mx): this is not windows safe
			job := strings.SplitN(strings.TrimPrefix(executable, "/var/vcap/jobs/"), "/", 2)[0]

			_, exists := jobStatus[job]
			jobStatus[job] = jobSucceeded && (jobStatus[job] || !exists)
		}

		allSucceeded = allSucceeded && jobSucceeded
	}

	groupStatus := map[string]bool{}

	// TODO(db,mx) configurable glob path for windows/testability?
	boshLinksPaths, err := filepath.Glob("/var/vcap/jobs/*/bosh/links")
	if err != nil {
		panic("could not glob */bosh/links") // TODO(db,mx) don't panic
	}

	for _, boshLinksPath := range boshLinksPaths {
		job := strings.SplitN(strings.TrimPrefix(boshLinksPath, "/var/vcap/jobs/"), "/", 2)[0]
		linkBytes, err := ioutil.ReadFile(boshLinksPath)
		if err != nil {
			m.logger.Error("HealthExecutableMonitor", "Error reading '%s': %v", boshLinksPath, err)
			continue
		}

		var parsedResponse []struct {
			BaseAddress string `json:"base_address"`
		}

		err = json.Unmarshal(linkBytes, &parsedResponse)
		if err != nil {
			m.logger.Error("HealthExecutableMonitor", "Error unmarshalling '%s': %v", boshLinksPath, err)
			continue
		}

		for _, link := range parsedResponse {
			if jobStatus[job] {
				groupStatus[link.BaseAddress] = jobStatus[job]
			} else {
				groupStatus[link.BaseAddress] = allSucceeded
			}
		}
	}

	m.mutex.Lock()
	m.status = allSucceeded
	m.groupStatus = groupStatus
	m.mutex.Unlock()
}
