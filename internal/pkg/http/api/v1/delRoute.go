package apiv1

import (
	"encoding/base64"
	"fmt"

	"github.com/labstack/echo/v4"
)

type deleteRouteOutput struct {
	Methods  []string
	Endpoint string
}

func parseDelRouteInput(c echo.Context) (*deleteRouteOutput, error) {
	url := c.FormValue("urlPath")
	methods, _ := base64.StdEncoding.DecodeString(c.FormValue("methods"))

	if methods == nil || url == "" {
		return nil, fmt.Errorf("form fields cannot be null")
	}

	return &deleteRouteOutput{
		Methods:  parseMethods(methods),
		Endpoint: parseUrl(url),
	}, nil
}
