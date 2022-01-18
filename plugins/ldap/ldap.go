package main

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
	"github.com/tmoneypenny/conspirator/pkg/config"
	"github.com/tmoneypenny/conspirator/pkg/wrapper"
)

/*
PluginType: LDAP
*/

// builsSpecFromConfig parses the default configuration
// to build the .so used for plugins
func buildSpecFromConfig() *LDAPConfig {
	var ldapConfig LDAPConfig
	// Load config
	config.InitConfig("conspirator", "")

	// build listeners
	for i, v := range viper.Get("plugins.ldap.listeners").([]interface{}) {
		address := v.(map[string]interface{})["address"].(string)
		port := int(v.(map[string]interface{})["port"].(float64))
		tls := v.(map[string]interface{})["tls"]

		ldapConfig.Configs = append(ldapConfig.Configs, LDAPServerConfig{
			Address: &address,
			Port:    &port,
			TLS:     nil,
		})

		if tls != nil {
			cert := tls.(map[string]interface{})["publicKey"].(string)
			pk := tls.(map[string]interface{})["privateKey"].(string)
			ldapConfig.Configs[i].TLS = &TLSConfig{
				CertFile: &cert,
				KeyFile:  &pk,
			}
		}
	}

	// build DNs
	for _, v := range viper.Get("plugins.ldap.dn").([]interface{}) {
		baseDN := v.(map[string]interface{})["baseDN"].(string)
		attributes := v.(map[string]interface{})["attributes"].(map[string]interface{})

		ldapConfig.DNs = append(ldapConfig.DNs, DistinguishedNames{
			BaseDN:     baseDN,
			Attributes: attributes,
		})
	}

	return &ldapConfig
}

// Server is a facade to work with LDAP
type Server struct {
	listener *server
}

// DistinguishedNames is a mapping of BaseDN to Entries
type DistinguishedNames struct {
	BaseDN     string
	Attributes map[string]interface{} // Entries are KV-Pairs
}

// LDAPConfig holds all the server configurations
// for an LDAP Server
type LDAPConfig struct {
	DNs            []DistinguishedNames
	Configs        []LDAPServerConfig
	PollingManager *polling.PollingServer
	PublicAddress  string
}

// LDAPServerConfig contains fields necessary to build
// a listener to serve requests
type LDAPServerConfig struct {
	// Address is the IP address that the listener will be
	// bound to when starting the server.
	// Address cannot be nil. If it is requested to bind
	// to a generic interface, use 0.0.0.0 to represent *:<port>
	Address *string
	Port    *int
	TLS     *TLSConfig
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

// NewServer takes a wrapper config and returns an interface
// that implements the Start and Stop methods
func NewServer(cfg wrapper.Config) wrapper.Module {
	ldapConfig := buildSpecFromConfig()

	ldapConfig.PollingManager = cfg.PollingManager
	ldapConfig.PublicAddress = cfg.PublicAddress

	ldapFacade := &Server{
		listener: newServer(ldapConfig),
	}

	return ldapFacade
}

// Start method starts the LDAP server listeners
func (s *Server) Start() {
	log.Debug().Msg("Starting LDAP server")
	s.listener.startListeners()
}

// Stop method stops the LDAP server listeners
func (s *Server) Stop() {
	log.Debug().Msg("Stopping LDAP server")
	s.listener.stopListeners()
}
