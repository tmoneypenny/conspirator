package apiv1

import (
	"encoding/base64"
	"fmt"

	"github.com/labstack/echo/v4"
)

// addRouteOutput is returned to addRoute to construct
// a new exploit page
type addRouteOutput struct {
	Methods  []string
	Endpoint string
	Headers  map[string]string
	Body     string
}

func parseAddRouteInput(c echo.Context) (*addRouteOutput, error) {
	url := c.FormValue("urlPath")
	methods, _ := base64.StdEncoding.DecodeString(c.FormValue("methods"))
	headers, _ := base64.StdEncoding.DecodeString(c.FormValue("headers"))
	body, _ := base64.StdEncoding.DecodeString(c.FormValue("body"))

	if methods == nil || headers == nil || body == nil || url == "" {
		return nil, fmt.Errorf("form fields cannot be null")
	}

	return &addRouteOutput{
		Methods:  parseMethods(methods),
		Endpoint: parseUrl(url),
		Headers:  parseHeaders(string(headers)),
		Body:     string(body),
	}, nil
}
