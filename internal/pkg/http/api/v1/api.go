package apiv1

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/tmoneypenny/conspirator/internal/pkg/http/api/v1/docs"
	auth "github.com/tmoneypenny/conspirator/internal/pkg/http/middleware"
)

//customRoutes []customRouteInfo
var customRoutes = make(map[string]map[string]string) // endpoint: Method: content-type?

type customRouteInfo struct {
	//Path    string
	Method string
}

// @title API
// @description Provides an API for interacting with the server
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api/v1
// @version v1
// @securitydefinitions.apikey AuthToken
// @in header
// @name Authorization
// @scope.admin

// Router defines a new subRouter for the API version
func Router(s *echo.Echo) {
	s.Pre(middleware.Rewrite(map[string]string{
		"/metrics":        "/api/v1/metrics",
		"/api/v1/healthz": "/healthz",
	}))

	s.GET("/healthz", healthCheck)

	jwtConfig := middleware.JWTConfig{
		Claims:                  &auth.JWTClaim{},
		SigningKey:              auth.JWTSigningKey,
		ErrorHandlerWithContext: auth.JWTAPIError,
	}

	apiV1 := s.Group("/api/v1")
	apiV1.Use(middleware.JWTWithConfig(jwtConfig))

	apiV1.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	apiV1.POST("/addRoute", func(c echo.Context) (err error) {
		addRoute(s, c, "s")
		return
	})

	apiV1.POST("/deleteRoute", func(c echo.Context) (err error) {
		deleteRoute(s, c, "s")
		return
	})

	apiV1.GET("/showRoutes", func(c echo.Context) (err error) {
		showRoutes(s, c, "s")
		return
	})
}

// metrics godoc
// @Summary Add route
// @Description add a new route
// @Tags routes
// @Accept mpfd
// @Param urlPath formData string true "absolute URL path, e.g. /test or /test.jpg"
// @Param methods formData string true "list of b64 encoded HTTP methods, e.g. GET,POST,PUT"
// @Param headers formData string true "list of b64 encoded headers separated by \r\n"
// @Param body formData string true "base64 encoded body"
// @Produce json
// @Success 200 {object} string "OK"
// @Failure 401 {string} string "Invalid Token"
// @Failure 500 {string} string "Internal Server Error"
// @security AuthToken
// @Router /addRoute [post]
func addRoute(t *echo.Echo, c echo.Context, s string) error {
	r, err := parseAddRouteInput(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status": fmt.Sprint(err),
		})
	}

	// default content-type to use if unset
	contentType := "text/html"
	if _, ok := customRoutes[r.Endpoint]; !ok {
		customRoutes[r.Endpoint] = make(map[string]string)
	}

	for m := range r.Methods {
		for k, v := range r.Headers {
			if strings.EqualFold(k, "content-type") {
				contentType = v
			}
		}

		customRoutes[r.Endpoint][r.Methods[m]] = contentType

		t.Router().Add(r.Methods[m], r.Endpoint, func(c echo.Context) (err error) {
			for h, v := range r.Headers {
				c.Response().Header().Add(h, v)
			}

			return c.HTML(http.StatusOK, r.Body)
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "OK",
	})
}

// metrics godoc
// @Summary Delete route
// @Description resets a route to default
// @Tags routes
// @Accept mpfd
// @Param urlPath formData string true "absolute URL path, e.g. /test or /test.jpg"
// @Param methods formData string true "list of b64 encoded HTTP methods, e.g. GET,POST,PUT"
// @Produce json
// @Success 200 {object} string "OK"
// @Failure 401 {string} string "Invalid Token"
// @Failure 500 {string} string "Internal Server Error"
// @security AuthToken
// @Router /deleteRoute [post]
func deleteRoute(t *echo.Echo, c echo.Context, s string) error {
	r, err := parseDelRouteInput(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status": fmt.Sprint(err),
		})
	}

	for m := range r.Methods {
		if len(customRoutes[r.Endpoint]) > 1 {
			delete(customRoutes[r.Endpoint], r.Methods[m])
		} else {
			delete(customRoutes, r.Endpoint)
		}

		t.Router().Add(r.Methods[m], r.Endpoint, func(c echo.Context) (err error) {
			return c.HTML(http.StatusOK,
				"<html><body>"+
					fmt.Sprintf("%x", md5.Sum([]byte(c.Request().Host)))+
					"</body></html>")
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "OK",
	})
}

// metrics godoc
// @Summary Show routes
// @Description show all added routes
// @Tags routes
// @Accept mpfd
// @Produce json
// @Success 200 {object} string "OK"
// @Failure 401 {string} string "Invalid Token"
// @Failure 500 {string} string "Internal Server Error"
// @security AuthToken
// @Router /showRoutes [get]
func showRoutes(t *echo.Echo, c echo.Context, s string) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"Routes": customRoutes,
	})
}

// healthCheck godoc
// @Summary Show the status of server.
// @Description get the status of server.
// @Tags status
// @Accept */*
// @Produce json
// @Success 200 {object} string "OK"
// @Failure 500 {string} string "Internal Server Error"
// @Router /healthz [get]
func healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"status": "OK"})
}

// metrics godoc
// @Summary Get metrics
// @Description get server metrics in prometheus format
// @Tags status
// @Accept */*
// @Produce plain
// @Success 200 {object} string "OK"
// @Failure 401 {string} string "Invalid Token"
// @Failure 500 {string} string "Internal Server Error"
// @security AuthToken
// @Router /metrics [get]
func metrics(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}
