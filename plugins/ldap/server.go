package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/nmcclain/ldap"
	"github.com/rs/zerolog/log"
	"github.com/tmoneypenny/conspirator/internal/pkg/encoding"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
	"github.com/tmoneypenny/conspirator/internal/pkg/util"
)

const (
	fileAttributePrefix = "file://"
)

type server struct {
	LDAP          *ldap.Server
	PollingServer *polling.PollingServer
	DNs           []DistinguishedNames
	PublicAddress string
	Marshaller    *encoding.Marshal
	ServerConfigs []LDAPServerConfig
}

type handler struct {
	server *server
}

var ldapEntries []*ldap.Entry

// newServer takes a specification and returns a new ldap.server
func newServer(specs *LDAPConfig) *server {
	return &server{
		PollingServer: specs.PollingManager,
		PublicAddress: specs.PublicAddress,
		Marshaller:    encoding.NewMarshaller(encoding.Format("burp")),
		DNs:           specs.DNs,
		LDAP:          ldap.NewServer(),
		ServerConfigs: specs.Configs,
	}
}

func (s *server) startListeners() {
	customDN := handler{server: s}

	for _, d := range s.DNs {
		log.Debug().Msgf("Adding Fns for DN: %s", d.BaseDN)
		s.LDAP.BindFunc(d.BaseDN, customDN.server)
		s.LDAP.SearchFunc(d.BaseDN, customDN.server)
		s.LDAP.CloseFunc(d.BaseDN, customDN.server)
	}

	s.LDAP.EnforceLDAP = true

	// Build a list of entries from DNs
	customDN.buildDatabase()

	// Start LDAP
	for i := range s.ServerConfigs {
		log.Info().Msgf("Starting LDAP listener %v:%d", *s.ServerConfigs[i].Address, *s.ServerConfigs[i].Port)
		go func(svr LDAPServerConfig) {
			if svr.TLS != nil {
				if err := s.LDAP.ListenAndServeTLS(
					fmt.Sprintf("%s:%d", *svr.Address, *svr.Port),
					*svr.TLS.CertFile,
					*svr.TLS.KeyFile,
				); err != nil {
					log.Fatal().Msgf("failed to start LDAPs: %s", err.Error())
				}
			} else {
				if err := s.LDAP.ListenAndServe(
					fmt.Sprintf("%s:%d", *svr.Address, *svr.Port),
				); err != nil {
					log.Fatal().Msgf("failed to start LDAP: %s", err.Error())
				}
			}
		}(s.ServerConfigs[i])
	}
	log.Info().Msg("LDAP listeners started")
}

func (s *server) stopListeners() {
	s.LDAP.QuitChannel(s.LDAP.Quit)
}

func (s *handler) buildDatabase() {
	for e := range s.server.DNs {
		var ldapEntryAttributes []*ldap.EntryAttribute
		for attr, value := range s.server.DNs[e].Attributes {
			switch v := value.(type) {
			case string:
				if strings.HasPrefix(v, fileAttributePrefix) {
					// FileReader will error if the file is not found
					value, _ = util.FileReader(strings.TrimPrefix(v, fileAttributePrefix))
					value = fmt.Sprintf("< %v", string(value.([]byte)))
				}
				ldapEntryAttributes = append(ldapEntryAttributes, &ldap.EntryAttribute{
					Name:   attr,
					Values: []string{value.(string)},
				})
			case []interface{}:
				var values []string
				for k := range v {
					values = append(values, v[k].(interface{}).(string))
				}
				ldapEntryAttributes = append(ldapEntryAttributes, &ldap.EntryAttribute{
					Name:   attr,
					Values: values,
				})
			default:
				log.Warn().Msgf("Attribute type is %T and not supported", v)
			}
		}

		ldapEntries = append(ldapEntries, &ldap.Entry{
			DN:         s.server.DNs[e].BaseDN,
			Attributes: ldapEntryAttributes,
		})
	}
}

// Bind implements LDAP Bind from RFC4510. Currently, this only implements anonymous binds
func (s *server) Bind(bindDN, bindPW string, conn net.Conn) (ldap.LDAPResultCode, error) {
	log.Debug().Msgf("BindDN from BindFn: %s", bindDN)
	// Allow anonymous binds
	return ldap.LDAPResultSuccess, nil
}

// Search implements LDAP Search from RFC4510
func (s *server) Search(boundDN string, searchReq ldap.SearchRequest, conn net.Conn) (ldap.ServerSearchResult, error) {
	log.Debug().Msgf("caught search request: %s, %v", searchReq.BaseDN, searchReq)

	results := ldap.ServerSearchResult{
		Entries:    ldapEntries,
		Referrals:  []string{},
		Controls:   []ldap.Control{},
		ResultCode: ldap.LDAPResultSuccess,
	}

	go interactionHandler(
		s, // server
		searchReq.BaseDN+"/"+searchReq.Filter,
		ldapResponsePrinter(results.Entries...), // why [0]?
		conn.RemoteAddr().String(),
	)

	return results, nil
}

// Close the network connection for the DN
func (s *server) Close(boundDN string, conn net.Conn) error {
	return conn.Close()
}

func ldapResponsePrinter(entries ...*ldap.Entry) string {
	var result strings.Builder
	for i := range entries {
		dn := entries[i].DN
		fmt.Fprintf(&result, "DN: %s\n", dn)
		for j := range entries[i].Attributes {
			fmt.Fprintf(&result, "%s: %s\n", entries[i].Attributes[j].Name, entries[i].Attributes[j].Values)
		}
	}
	return result.String()
}

func interactionHandler(s *server, request, response string, clientIP string) {
	input := &encoding.RawInput{
		InteractionURI: "ldap://" + request,
		Protocol:       "ldap",
		ClientIP:       clientIP, // conn.RemoteAddr
		Request:        []byte(request),
		Response:       []byte(response),
	}

	jsonData, err := s.Marshaller.MarshalToJSON(input)

	if err != nil {
		log.Error().Msg("error marshalling LDAP data")
	}

	s.PollingServer.Publish(jsonData)
}
