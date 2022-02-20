package middleware

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

var (
	CSRFKey               = []byte(viper.GetString("http.csrfKey"))    // 32-byte Key
	JWTSigningKey         = []byte(viper.GetString("http.signingKey")) // 32-byte Key
	BearerTokenCookieName = "access-token"
	tokenTTL              = time.Hour * 12
)

// JWTClaim is used to create a new JTW Claim
type JWTClaim struct {
	Name string `json:"name"`
	jwt.StandardClaims
}

// User provides a struct for echo.Bind
type User struct {
	Password string `json:"password" form:"password"`
	Username string `json:"username" form:"username"`
}

// AdminUserAccount parses the credentials in the configuration
func LoadAdminAccount() *User {
	user := viper.GetString("http.username")
	pass := viper.GetString("http.password")
	hPass, _ := bcrypt.GenerateFromPassword([]byte(pass), 8)
	return &User{Password: string(hPass), Username: user}
}

func JWTRedirectError(err error, c echo.Context) error {
	return c.Redirect(http.StatusMovedPermanently, c.Echo().Reverse("adminLoginForm"))
}

func JWTAPIError(err error, c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, "Invalid Token")
}

// GenerateToken generates a new bearer token for the user
func GenerateToken(user *User, c echo.Context) error {
	token, expiration, err := user.generateAccessToken()
	if err != nil {
		return err
	}

	// set token cookie
	user.setCookie(BearerTokenCookieName, token, expiration, c)

	return nil
}

func (u *User) setCookie(name, value string, expiration time.Time, c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    value,
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
	})
}

func (u *User) generateAccessToken() (string, time.Time, error) {
	expiration := time.Now().Add(tokenTTL)
	claims := &JWTClaim{
		Name: u.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiration.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	bearer, err := token.SignedString(JWTSigningKey)
	if err != nil {
		return "", time.Now(), err
	}

	return bearer, expiration, nil
}
