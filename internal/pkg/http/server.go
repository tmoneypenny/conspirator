package http

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/ziflex/lecho/v2"
	"golang.org/x/net/websocket"

	"github.com/tmoneypenny/conspirator/internal/pkg/encoding"
	apiv1 "github.com/tmoneypenny/conspirator/internal/pkg/http/api/v1"
	"github.com/tmoneypenny/conspirator/internal/pkg/http/controller"
	interaction "github.com/tmoneypenny/conspirator/internal/pkg/http/middleware"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
)

var (
	pollingRegex *regexp.Regexp
)

var (
	gracePeriod = 5 * time.Second
	apiVersion  = "1.0"
	burpVersion = "4"
)

var (
	failedAllowList = promauto.NewCounter(prometheus.CounterOpts{
		Name: "denied_ips_total",
		Help: "Total denied IPs",
	})

	pollingInteractionEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "polling_interaction_events_total",
		Help: "Total Polling Interaction Events",
	})
)

type server struct {
	HTTP           *echo.Echo
	AllowList      *[]string
	PollingDomain  *string
	PollingManager *polling.PollingServer
	Marshaller     *encoding.Marshal
	Version2       *bool
}

func checkAllowlist(ip string, allowed *[]string) bool {
	reqIP := net.ParseIP(ip)
	log.Debug().Msgf("Polling interaction from %s", reqIP.String())
	for _, checkIP := range *allowed {
		goodIP := net.ParseIP(checkIP)
		if reqIP.Equal(goodIP) {
			return true
		}
	}
	failedAllowList.Inc()
	return false
}

func newServer(spec *HTTPServerConfig) *server {
	server := &server{
		HTTP:           echo.New(),
		AllowList:      spec.AllowList,
		PollingDomain:  spec.PollingDomain,
		PollingManager: spec.PollingManager,
		Version2:       spec.Version2,
	}

	server.HTTP.Server.Addr = fmt.Sprintf("%s:%d", *spec.Address, *spec.Port)
	pollingRegex = regexp.MustCompile(fmt.Sprintf(`^(%s[\.]{1})[^\.].*`, *spec.PollingDomain))

	if spec.TLS != nil {
		cert, err := tls.LoadX509KeyPair(*spec.TLS.CertFile, *spec.TLS.KeyFile)
		if err != nil {
			log.Fatal().Msgf("Cannot load KeyPair: %v", err)
		}

		server.HTTP.TLSServer = &http.Server{
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
			Addr: fmt.Sprintf("%s:%d", *spec.Address, *spec.TLS.Port),
		}
	}
	return server
}

func (s *server) startListeners() {
	s.HTTP.HideBanner = true
	s.setupRouter()
	s.HTTP.Logger = lecho.From(log.Logger)      // Echo Wrapper
	s.HTTP.IPExtractor = echo.ExtractIPDirect() // No proxy

	s.HTTP.Use(middleware.RequestID())
	s.HTTP.Use(middleware.Recover())
	s.HTTP.Use(lecho.Middleware(lecho.Config{
		Logger: lecho.From(log.Logger),
	}))

	// Set default marshaller for polling requests
	s.Marshaller = encoding.NewMarshaller(encoding.Format(viper.Get("pollingEncoding").(string)))

	// static middleware
	if viper.GetBool("http.static.enable") {
		s.HTTP.Group(viper.GetString("http.static.prefix"),
			middleware.StaticWithConfig(middleware.StaticConfig{
				Root:   viper.GetString("http.static.path"),
				Browse: viper.GetBool("http.static.browsing"),
			}),
		)
	}

	s.HTTP.Use(interaction.InteractionMiddleware(interaction.InteractionConfig{
		Polling:    s.PollingManager,
		Marshaller: s.Marshaller,
		Skipper: func(c echo.Context) bool {
			reqHost, _, err := net.SplitHostPort(c.Request().Host)
			if err != nil {
				reqHost = c.Request().Host
			}

			if pollingRegex.MatchString(reqHost) {
				return true
			}

			if strings.HasPrefix(c.Path(), "/admin") ||
				strings.HasPrefix(c.Path(), "/api/v1") {
				return true
			}

			return false
		},
		Version: apiVersion,
	}))

	// Start HTTP
	go func() {
		if err := s.HTTP.StartServer(s.HTTP.Server); err != nil {
			log.Fatal().Msgf("%v", err)
		}

		log.Info().Msg("HTTP server started")
	}()

	// Start HTTPS
	if s.HTTP.TLSServer != nil {
		go func() {
			// Enable HTTP/2 support
			if *s.Version2 {
				s.HTTP.TLSServer.TLSConfig.NextProtos = append(s.HTTP.TLSServer.TLSConfig.NextProtos, "h2")
			}

			if err := s.HTTP.StartServer(s.HTTP.TLSServer); err != nil {
				log.Fatal().Msgf("%v", err)
			}

			log.Info().Msg("HTTP server started")
		}()
	}
}

func (s *server) stopListeners() {
	ctx, cancel := context.WithTimeout(context.Background(), gracePeriod)
	defer cancel()
	s.HTTP.Server.Shutdown(ctx)
	if s.HTTP.TLSServer != nil {
		s.HTTP.TLSServer.Shutdown(ctx)
	}
}

func (s *server) setupRouter() {
	// setup template render
	templateRenderer := &controller.Template{
		Templates: template.Must(
			template.ParseGlob(
				fmt.Sprintf("%s/*.tmpl",
					viper.GetString("http.templatePath"),
				),
			),
		),
	}

	// HTTP Responder
	s.HTTP.Renderer = templateRenderer

	// API
	apiv1.Router(s.HTTP)

	// Controllers
	controller.Router(s.HTTP)

	// debug router
	s.HTTP.GET("/debug", s.httpDebug)

	// Catch-all
	s.HTTP.Any("/*", func(c echo.Context) (err error) {
		reqHost, _, err := net.SplitHostPort(c.Request().Host)
		if err != nil {
			reqHost = c.Request().Host
		}

		/* NOTE: The polling endpoint does not take a path!
		If a path is passed that matches an earlier router in
		the chain, then it will most likely be prioritized.
		*/
		if pollingRegex.MatchString(reqHost) {
			// c.RealIP is used since we do not use a proxy
			// if request IP is not in the allow list, then we
			// should return the default response
			log.Debug().Msg("Matched polling")
			if checkAllowlist(c.RealIP(), s.AllowList) {
				if c.IsWebSocket() {
					websocket.Handler(func(ws *websocket.Conn) {
						defer ws.Close()
						for {
							var msg string
							err = websocket.Message.Receive(ws, &msg)
							if err != nil {
								log.Warn().Msgf("wss error: %v", err)
								return
							}

							log.Info().Msgf("WSS Message: %s\n", msg)
							if events, err := s.pollingResults(); err == nil {
								websocket.Message.Send(ws, string(events))
							} else {
								websocket.Message.Send(ws, websocket.ErrNotSupported)
							}
						}
					}).ServeHTTP(c.Response(), c.Request())
					return nil
				} else {
					events, err := s.pollingResults()

					switch s.Marshaller.Marshaller.(type) {
					case *encoding.BurpMarshaller:
						c.Response().Header().Add("X-Collaborator-Version", burpVersion)
						c.Response().Header().Add("X-Collaborator-Time", fmt.Sprint(time.Now().UnixNano()/int64(time.Millisecond)))
						c.Response().Header().Add("Server", "Conspirator https://github.com/tmoneypenny/conspirator")
						c.Response().Header().Add("Content-Type", "application/json")
					}

					if len(events) == 0 {
						return c.JSONBlob(http.StatusOK, events)
					}

					if err == nil {
						return c.JSONBlob(http.StatusOK, events)
					} else {
						return c.NoContent(http.StatusInternalServerError)
					}
				}
			}
		}
		s.defaultResponder(c)
		return
	})
}

func (s *server) pollingResults() ([]byte, error) {
	events := s.PollingManager.GetAll()
	if len(events) == 0 {
		return s.Marshaller.EmptyResponse(), nil
	}

	pollingInteractionEvents.Inc()
	// If any field in events, or event itself, is of type []byte,
	// then the JSON marshaller will encode the response as a
	// base64-encoded strings
	if eventBlob, err := s.Marshaller.EventToBlob(events); err == nil {
		return eventBlob, nil
	} else {
		return []byte{}, err
	}
}

func (s *server) defaultResponder(c echo.Context) error {
	return c.HTML(http.StatusOK,
		"<html><body>"+
			fmt.Sprintf("%x", md5.Sum([]byte(c.Request().Host)))+
			"</body></html>",
	)
}

func (s *server) httpDebug(c echo.Context) error {
	req := c.Request()
	format := `
		<code>
		Protocol: %s<br>
		Host: %s<br>
		Remote Address: %s<br>
		Method: %s<br>
		Path: %s<br>
		</code>
 	`
	return c.HTML(http.StatusOK, fmt.Sprintf(format, req.Proto, req.Host, req.RemoteAddr, req.Method, req.URL.Path))
}
