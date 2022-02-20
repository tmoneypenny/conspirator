package controller

import (
	"net/http"
	"time"

	"github.com/gorilla/csrf"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	auth "github.com/tmoneypenny/conspirator/internal/pkg/http/middleware"
	"golang.org/x/crypto/bcrypt"
)

var (
	adminLoginEventsSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "admin_login_events_success_total",
		Help: "Successful admin login events",
	})

	adminLoginEventsFail = promauto.NewCounter(prometheus.CounterOpts{
		Name: "admin_login_events_fail_total",
		Help: "Failed admin login events",
	})
)

func adminLoginForm() echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("X-Csrf-Token", csrf.Token(c.Request()))
		return c.Render(http.StatusOK, "signin.tmpl", map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(c.Request()),
		})
	}
}

func adminLogin() echo.HandlerFunc {
	return func(c echo.Context) error {
		adminUser := auth.LoadAdminAccount()

		loginUser := new(auth.User)
		if err := c.Bind(loginUser); err != nil {
			adminLoginEventsFail.Inc()
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		if adminUser.Username != loginUser.Username {
			adminLoginEventsFail.Inc()
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid Username or Password")
		}

		if err := bcrypt.CompareHashAndPassword([]byte(adminUser.Password), []byte(loginUser.Password)); err != nil {
			adminLoginEventsFail.Inc()
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid Username or Password")
		}

		// generate token
		if err := auth.GenerateToken(adminUser, c); err != nil {
			adminLoginEventsFail.Inc()
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid Token")
		}

		adminLoginEventsSuccess.Inc()
		return c.Redirect(http.StatusMovedPermanently, "/admin/home")
	}
}

// adminLogout is a weak logout as the JWT will still be valid until it expires.
// typically this is done by adding the token to a denyList until it can naturally expire,
// but to save some dev cycles we chose do this :)
func adminLogout() echo.HandlerFunc {
	return func(c echo.Context) error {
		c.SetCookie(&http.Cookie{
			Name:     auth.BearerTokenCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Expires:  time.Now().Add(-7 * 24 * time.Hour),
		})
		return c.Redirect(http.StatusMovedPermanently, "/user/signin")
	}
}

// adminSettings - /admin/settings
func adminSettings(c echo.Context) error {
	c.Response().Header().Set("X-Csrf-Token", csrf.Token(c.Request()))
	accessCookie, _ := c.Cookie("access-token")
	return c.Render(http.StatusOK, "settings.tmpl", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(c.Request()),
		"AccessToken":    accessCookie.Value,
	})
}

// adminHome - /admin/home
func adminHome(c echo.Context) error {
	c.Response().Header().Set("X-Csrf-Token", csrf.Token(c.Request()))
	accessCookie, _ := c.Cookie("access-token")
	return c.Render(http.StatusOK, "adminHome.tmpl", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(c.Request()),
		"AccessToken":    accessCookie.Value,
	})
}

// addRoute - /admin/addRoute
func addRoute(c echo.Context) error {
	c.Response().Header().Set("X-Csrf-Token", csrf.Token(c.Request()))
	accessCookie, _ := c.Cookie("access-token")
	return c.Render(http.StatusOK, "addRoute.tmpl", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(c.Request()),
		"AccessToken":    accessCookie.Value,
	})
}

// deleteRoute - /admin/deleteRoute
func deleteRoute(c echo.Context) error {
	c.Response().Header().Set("X-Csrf-Token", csrf.Token(c.Request()))
	accessCookie, _ := c.Cookie("access-token")
	return c.Render(http.StatusOK, "deleteRoute.tmpl", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(c.Request()),
		"AccessToken":    accessCookie.Value,
	})
}

// showRoutes - /admin/showRoutes
func showRoutes(c echo.Context) error {
	c.Response().Header().Set("X-Csrf-Token", csrf.Token(c.Request()))
	accessCookie, _ := c.Cookie("access-token")
	return c.Render(http.StatusOK, "showRoutes.tmpl", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(c.Request()),
		"AccessToken":    accessCookie.Value,
	})
}

// pollServer - /admin/poll
func pollServer(c echo.Context) error {
	c.Response().Header().Set("X-Csrf-Token", csrf.Token(c.Request()))
	accessCookie, _ := c.Cookie("access-token")
	return c.Render(http.StatusOK, "polling.tmpl", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(c.Request()),
		"AccessToken":    accessCookie.Value,
	})
}
