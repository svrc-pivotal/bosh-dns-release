package healthserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry/bosh-utils/system"
	"github.com/pivotal-cf/paraphernalia/secure/tlsconfig"
)

type HealthServer interface {
	Serve(config *HealthCheckConfig)
}

type HealthExecutable interface {
	Status() (bool, map[string]bool)
}

type concreteHealthServer struct {
	logger             boshlog.Logger
	fs                 system.FileSystem
	healthJsonFileName string
	healthExecutable   HealthExecutable
}

const logTag = "healthServer"

func NewHealthServer(logger boshlog.Logger, fs system.FileSystem, healthFileName string, healthExecutable HealthExecutable) HealthServer {
	return &concreteHealthServer{
		logger:             logger,
		fs:                 fs,
		healthJsonFileName: healthFileName,
		healthExecutable:   healthExecutable,
	}
}

func (c *concreteHealthServer) Serve(config *HealthCheckConfig) {
	http.HandleFunc("/health", c.healthEntryPoint)

	caCert, err := ioutil.ReadFile(config.CAFile)
	if err != nil {
		log.Fatal(err)
		return
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(config.CertificateFile, config.PrivateKeyFile)
	if err != nil {
		log.Fatal(err)
		return
	}

	tlsConfig := tlsconfig.Build(
		tlsconfig.WithIdentity(cert),
		tlsconfig.WithInternalServiceDefaults(),
	)

	serverConfig := tlsConfig.Server(tlsconfig.WithClientAuthentication(caCertPool))
	serverConfig.BuildNameToCertificate()

	server := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", config.Address, config.Port),
		TLSConfig: serverConfig,
	}
	server.SetKeepAlivesEnabled(false)

	serveErr := server.ListenAndServeTLS("", "")
	c.logger.Error(logTag, "http healthcheck ending with %s", serveErr)
}

func (c *concreteHealthServer) healthEntryPoint(w http.ResponseWriter, r *http.Request) {
	// Should not be possible to get here without having a peer certificate
	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	if cn != CN {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("TLS certificate common name does not match"))
		return
	}

	healthRaw, err := ioutil.ReadFile(c.healthJsonFileName)

	if err != nil {
		c.logger.Error(logTag, "Failed to read healthcheck data %s. error: %s", healthRaw, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var health struct {
		State string `json:"state"`
	}

	err = json.Unmarshal(healthRaw, &health)
	if err != nil {
		c.logger.Error(logTag, "Failed to unmarshal healthcheck data %s. error: %s", healthRaw, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	status, groupStatus := c.healthExecutable.Status()
	stateString := health.State

	if !status {
		stateString = "job-health-executable-fail"
	}

	healthResponse := struct {
		State      string            `json:"state"`
		GroupState map[string]string `json:"group_state"`
	}{
		State:      stateString,
		GroupState: map[string]string{},
	}

	for groupName, groupStatus := range groupStatus {
		healthResponse.GroupState[groupName] = map[bool]string{true: health.State, false: "job-health-executable-fail"}[groupStatus]
	}

	responseBytes, err := json.Marshal(healthResponse)
	if err != nil {
		c.logger.Error(logTag, "Failed to marshal response data %s. error: %s", responseBytes, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(responseBytes)
}
