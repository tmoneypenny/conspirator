package bind

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"github.com/tmoneypenny/conspirator/internal/pkg/encoding"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
	"github.com/tmoneypenny/conspirator/internal/pkg/util"
)

var (
	seed, _     = rand.Prime(rand.Reader, 256) // ignore error since we pass > 2 bits
	gracePeriod = 5 * time.Second
)

var (
	dnsInteractionEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dns_interaction_events_total",
		Help: "Total DNS interactions",
	})
)

// server contains a slice of dns.Servers to start as listeners
// as well as the zones to serve
type server struct {
	DNS           []*dns.Server
	Zones         []string
	PollingServer *polling.PollingServer
	Marshaller    *encoding.Marshal
	PublicAddress string
}

// newServer takes a specification and returns a new dns.server
func newServer(specs *BindConfig) *server {
	var dnsServers []*dns.Server

	for _, c := range specs.Configs {
		switch c.TLS {
		case nil:
			dnsServers = append(dnsServers, &dns.Server{
				Net:       *c.Net,
				Addr:      fmt.Sprintf("%s:%d", *c.Address, *c.Port),
				ReusePort: true,
				NotifyStartedFunc: func() {
					log.Info().Msg("DNS server started")
				},
			})
		default:
			cert, err := tls.LoadX509KeyPair(*c.TLS.CertFile, *c.TLS.KeyFile)
			if err != nil {
				log.Fatal().Msgf("Cannot load KeyPair: %v", err)
			}
			c.Net = util.StrToPtr("tcp-tls") // overwrite setting
			dnsServers = append(dnsServers, &dns.Server{
				Net:       *c.Net,
				Addr:      fmt.Sprintf("%s:%d", *c.Address, *c.Port),
				ReusePort: true,
				NotifyStartedFunc: func() {
					log.Info().Msg("DNS server started")
				},
				TLSConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
				},
			})
		}
	}

	return &server{
		DNS:           dnsServers,
		Zones:         specs.Zones,
		PollingServer: specs.PollingManager,
		Marshaller:    encoding.NewMarshaller(encoding.Format("burp")),
		PublicAddress: specs.PublicAddress,
	}
}

// startListeners registers handlers and listeners
func (s *server) startListeners() {
	// register handler for each zone
	for _, z := range s.Zones {
		dns.HandleFunc(dns.Fqdn(z), s.routeHandler)
	}

	if len(s.DNS) == 0 {
		// Create a default listener?
		log.Fatal().Msg("")
	}

	for i := range s.DNS {
		log.Info().Msgf("Starting DNS %v listener...", s.DNS[i].Net)
		go func(svr *dns.Server) {
			if err := svr.ListenAndServe(); err != nil {
				log.Fatal().Msgf("%v", err)
			}
		}(s.DNS[i])
	}
}

// stopListeners gracefully shuts down the server
func (s *server) stopListeners() {
	ctx, cancel := context.WithTimeout(context.Background(), gracePeriod)
	defer cancel()
	for i := range s.DNS {
		s.DNS[i].ShutdownContext(ctx)
	}
}

func (s *server) routeHandler(w dns.ResponseWriter, r *dns.Msg) {
	if rr, err := findRecordInZone(&Record{FQDN: &r.Question[0].Name}); err == nil {
		s.zoneHandler(w, r, rr)
	} else {
		s.defaultHandler(w, r)
	}
}

func (s *server) zoneHandler(w dns.ResponseWriter, r *dns.Msg, rrs zoneRRS) {
	m := new(dns.Msg)
	m.SetReply(r)

	m.RecursionAvailable = true
	m.Authoritative = true

	log.Debug().Msgf("received request for %v from %v", m.Question, w.RemoteAddr())

	m.Answer = append(m.Answer, rrs.Record.([]dns.RR)...)
	go s.interactionHandler(r, m, w.RemoteAddr().String())
	if err := w.WriteMsg(m); err != nil {
		log.Error().Msgf("failed to response to DNS query: %v", m)
	}
}

// handler is responsible for writing requests
func (s *server) defaultHandler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)

	m.RecursionAvailable = true
	m.Authoritative = true

	// ignore error assuming the tcp/udp connection returns a well-formed address
	localIP, _, _ := net.SplitHostPort(w.LocalAddr().String())
	if localIP == "0.0.0.0" || localIP == "::" || localIP == "" {
		localIP = s.PublicAddress
	}

	switch r.Question[0].Qtype {
	case dns.TypeA:
		aDefaultRRS[0].Header().Name = r.Question[0].Name
		aDefaultRRS[0].(*dns.A).A = net.ParseIP(localIP)
		m.Answer = append(m.Answer, aDefaultRRS...)
	case dns.TypeAAAA:
		aaaaDefaultRRS[0].Header().Name = r.Question[0].Name
		m.Answer = append(m.Answer, aaaaDefaultRRS...)
	case dns.TypeCNAME:
		cnameDefaultRRS.Header().Name = r.Question[0].Name
		cnameDefaultRRS.Target = r.Question[0].Name
		m.Answer = append(m.Answer, cnameDefaultRRS)
	case dns.TypeTXT:
		txtDefaultRRS.Header().Name = r.Question[0].Name
		m.Answer = append(m.Answer, txtDefaultRRS)
	case dns.TypeMX:
		mxDefaultRRS[0].Header().Name = r.Question[0].Name
		mxDefaultRRS[0].(*dns.MX).Mx = r.Question[0].Name
		m.Answer = append(m.Answer, mxDefaultRRS...)
	case dns.TypeIXFR, dns.TypeAXFR:
		log.Warn().Msgf("received zone transfer request: \n\t%v : %v",
			w.RemoteAddr().String(),
			m.Question,
		)
		m.SetRcode(r, dns.RcodeRefused)
	case dns.TypeSRV:
		svrDefaultRRS[0].Header().Name = r.Question[0].Name
		svrDefaultRRS[0].(*dns.SRV).Target = r.Question[0].Name
		m.Answer = append(m.Answer, svrDefaultRRS...)
	default:
		log.Warn().Msgf("received an unhandled RR type: %v", m.Question[0].Qtype)
		m.SetRcode(r, dns.RcodeNotImplemented)
	}

	// DNS RFC allow multiple questions in question section, but in practice it
	// never works. DNS servers see multiple questions as an error so use zero index
	log.Debug().Msgf("replied to question %v with answer %v [status: %v]", m.Question, m.Answer, m.Rcode)

	//go s.PollingServer.Publish(fmt.Sprintf("%v:%v", r.Question[0].Name, m.Answer))
	go s.interactionHandler(r, m, w.RemoteAddr().String())
	if err := w.WriteMsg(m); err != nil {
		log.Error().Msgf("failed to response to DNS query: %v", m)
	}
}

func (s *server) interactionHandler(q, a *dns.Msg, clientIP string) {
	input := &encoding.DNSInput{
		SubdomainQuestion: q.Question[0].Name,
		RawRequest:        q.Question[0].String(),
		RequestType:       q.Question[0].Qtype,
		Answer:            "", // Default
		OpCode:            a.Opcode,
		ClientIP:          clientIP,
	}

	if a.Rcode < 1 {
		input.Answer = a.Answer[0].Header().Name
	}

	jsonData, err := s.Marshaller.MarshalToJSON(input)

	if err != nil {
		log.Error().Msg("error marshalling DNS data")
	}

	dnsInteractionEvents.Inc()
	s.PollingServer.Publish(jsonData)
}
