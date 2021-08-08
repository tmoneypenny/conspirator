package middleware

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"github.com/tmoneypenny/conspirator/internal/pkg/encoding"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
)

var (
	interactionEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "http_interaction_events_total",
		Help: "Total Interaction Events",
	})
)

// InteractionConfig configures the InteractionMiddleware
type InteractionConfig struct {
	Version    string
	Polling    *polling.PollingServer
	Skipper    middleware.Skipper
	Marshaller *encoding.Marshal
}

type responseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Write implementers the responseWriter Write method
func (w *responseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// WriteHeader implementers the responseWriter WriteHeader method
func (w *responseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

// InteractionMiddleware captures interactions with the server
func InteractionMiddleware(config InteractionConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}
			reqBody := []byte{}
			c.Response().Header().Set("X-Server-Version", config.Version)

			if c.Request().Body != nil {
				reqBody, _ = ioutil.ReadAll(c.Request().Body)
			}
			c.Request().Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))

			resBody := new(bytes.Buffer)
			mw := io.MultiWriter(c.Response().Writer, resBody)
			writer := &responseWriter{Writer: mw, ResponseWriter: c.Response().Writer}
			c.Response().Writer = writer

			if err = next(c); err != nil {
				c.Error(err)
			}

			interactionEvents.Inc()
			config.interactionHandler(c, resBody.Bytes())

			return
		}
	}
}

// interactionHandler handles encoding the interaction in
// the desired format before publishing to the polling server
func (cfg *InteractionConfig) interactionHandler(c echo.Context, res []byte) {
	log.Debug().Msgf("Response Body in Handler: %s", string(res))
	jsonData, err := cfg.Marshaller.MarshalToJSON(&encoding.HTTPInput{Ctx: c, Response: res})
	if err != nil {
		log.Error().Msg("error marshalling ctx data")
	}
	cfg.Polling.Publish(jsonData)
}
