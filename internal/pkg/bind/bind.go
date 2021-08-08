package bind

import (
	"github.com/miekg/dns"
	"github.com/rs/zerolog/log"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
)

// Server is a facade to work with DNS
type Server struct {
	listener *server
}

// Record is used to update or delete a record from the zone
type Record struct {
	FQDN       *string     // Required. All addresses are converted to a FQDN
	RecordType *string     // Optional for Delete; Required for Upsert
	TTL        *uint32     // Optional for Delete; Required for Upsert
	Value      interface{} // Optional for Delete; Required for Upsert
}

// BindConfig holds all the server configurations
// and a list of zones to serve
type BindConfig struct {
	Configs        []BindServerConfig
	Zones          []string
	PollingManager *polling.PollingServer
	PublicAddress  string
}

// BindServerConfig contains fields necessary to build
// a listener to serve requests
type BindServerConfig struct {
	// Address is the IP address that the listener will be
	// bound to when starting the server.
	// Address cannot be nil. If it is requested to bind
	// to a generic interface, use 0.0.0.0 to represent *:<port>
	Address *string
	Port    *int
	Net     *string    // tcp, tcp-tls, udp
	TLS     *TLSConfig // If TLS is enabled, Net is forced to tcp-tls
}

// TLSConfig contains the path to PEM encoded certs
type TLSConfig struct {
	// CertFile must contain PEM encoded data. The CertFile
	// should contain intermediate certificates following the leaf
	// certificate to form a certificate chain.
	CertFile *string
	// KeyFile contains the private key that corresponds to the public key
	// in CertFile.
	KeyFile *string
}

// BindServer creates a new configured Server struct that can
// be started or stopped
func BindServer(cfg *BindConfig) *Server {
	bindFacade := &Server{
		listener: newServer(cfg),
	}

	return bindFacade
}

// Start will register listeners based on the server spec
// and start the defaultHandlers
func (s *Server) Start() {
	s.listener.startListeners()
}

// Stop will gracefully shutdown the server
func (s *Server) Stop() {
	s.listener.stopListeners()
}

// UpsertRRS will update or insert a resource record to override the
// the defaultHandler
func UpsertRRS(record *Record) {
	if err := upsertRRS(record); err != nil {
		log.Error().Msgf("Failed to upsert record: %v", err)
	}
}

// DeleteRRS will remove a resource record and convert the
// behavior of the record back to that of the defaultHandler
func DeleteRRS(record *Record) {
	deleteRecord(record)
}

// GetRRS will return the RR if it was found in the cache. Otherwise,
// GetRRS will return nil and false
func GetRRS(record *Record) (*Record, bool) {
	// findRecordInZone() // convert back to *Record
	if rec, found := findRecordInZone(record); found == nil {
		wireRecords := []string{}
		for _, record := range rec.Record.([]dns.RR) {
			wireRecords = append(wireRecords, record.String())
		}
		return &Record{
			FQDN:       &rec.Record.(dns.RR).Header().Name,
			RecordType: &rec.RecordType,
			TTL:        &rec.TTL,
			Value:      wireRecords,
		}, true
	}
	return nil, false
}

// import / export
