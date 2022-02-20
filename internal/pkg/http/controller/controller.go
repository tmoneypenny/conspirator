package controller

import (
	"fmt"
	"html/template"
	"io"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	auth "github.com/tmoneypenny/conspirator/internal/pkg/http/middleware"
)

// Template contains a reference to the template.Template type
type Template struct {
	Templates *template.Template
}

// Render is used to render a template with data
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

// Router handles all built-in, non-API, routes
func Router(s *echo.Echo) {
	loginGroup := s.Group("/user")
	loginGroup.Use(middleware.HTTPSRedirect())
	loginGroup.Use(echo.WrapMiddleware(csrf.Protect(auth.CSRFKey)))
	loginGroup.GET("/signin", adminLoginForm()).Name = "adminLoginForm"
	loginGroup.POST("/signin", adminLogin())

	loginGroup.GET("/signout", adminLogout())

	adminGroup := s.Group("/admin")
	adminGroup.Use(middleware.HTTPSRedirect())
	adminGroup.Use(echo.WrapMiddleware(csrf.Protect(auth.CSRFKey)))

	jwtConfig := middleware.JWTConfig{
		Claims:                  &auth.JWTClaim{},
		SigningKey:              auth.JWTSigningKey,
		TokenLookup:             fmt.Sprintf("cookie:%s", auth.BearerTokenCookieName),
		ErrorHandlerWithContext: auth.JWTRedirectError,
	}

	adminGroup.Use(middleware.JWTWithConfig(jwtConfig))

	// Setup SwaggerUI docs
	adminGroup.GET("/docs/*", echoSwagger.WrapHandler)

	adminGroup.GET("/home", adminHome)
	adminGroup.GET("/settings", adminSettings)
	adminGroup.GET("/addRoute", addRoute)
	adminGroup.GET("/deleteRoute", deleteRoute)
	adminGroup.GET("/showRoutes", showRoutes)
	adminGroup.GET("/poll", pollServer)
}
