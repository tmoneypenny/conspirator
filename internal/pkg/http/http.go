package http

import "github.com/tmoneypenny/conspirator/internal/pkg/polling"

// Server contains all HTTP listeners
type Server struct {
	listener *server
}

// HTTPServerConfig contains fields necessary to build
// a listener to serve requests
type HTTPServerConfig struct {
	Address        *string
	Port           *int
	AllowList      *[]string
	PollingDomain  *string
	PollingManager *polling.PollingServer
	TLS            *TLSConfig
	Version2       *bool
}

// TLSConfig contains the path to PEM encoded certs
type TLSConfig struct {
	Port     *int
	CertFile *string
	KeyFile  *string
}

// HTTPServer takes HTTPServerConfig input and returns
// a server with listeners
func HTTPServer(cfg *HTTPServerConfig) *Server {
	httpFacade := &Server{
		listener: newServer(cfg),
	}

	return httpFacade
}

// Start starts all HTTP listeners specified in Server
func (s *Server) Start() {
	s.listener.startListeners()
}

// Stop gracefully stops all HTTP listeners
func (s *Server) Stop() {
	s.listener.stopListeners()
}
